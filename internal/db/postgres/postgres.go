// Package postgres provides a PostgreSQL implementation of db.Database.
package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/csvexport"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/jsonexport"
	"github.com/ernestoponce27/db-tui/internal/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const listTablesSQL = `
	SELECT table_name
	FROM information_schema.tables
	WHERE table_schema = $1
		AND table_type = 'BASE TABLE'
	ORDER BY table_name`

const listSchemaObjectGroupsSQL = `
	SELECT schema_name, object_type
	FROM (
		SELECT table_schema AS schema_name, 'tables' AS object_type
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		UNION
		SELECT table_schema AS schema_name, 'views' AS object_type
		FROM information_schema.views
		UNION
		SELECT schemaname AS schema_name, 'materialized_views' AS object_type
		FROM pg_catalog.pg_matviews
		UNION
		SELECT namespace.nspname AS schema_name, 'functions' AS object_type
		FROM pg_catalog.pg_proc AS routine
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE routine.prokind = 'f'
	) AS object_groups
	WHERE schema_name NOT IN ('information_schema', 'pg_catalog')
		AND schema_name !~ '^pg_'
	ORDER BY schema_name,
		CASE object_type
			WHEN 'tables' THEN 1
			WHEN 'views' THEN 2
			WHEN 'materialized_views' THEN 3
			WHEN 'functions' THEN 4
		END`

const listColumnsSQL = `SELECT
	      attribute.attname AS column_name,
	      row_number() OVER (ORDER BY attribute.attnum) AS ordinal_position,
	      regexp_replace(
	          regexp_replace(
	              CASE type_catalog.typname
	                  WHEN 'int2' THEN 'int2'
	                  WHEN 'int4' THEN 'int4'
	                  WHEN 'int8' THEN 'int8'
	                  WHEN 'float4' THEN 'float4'
	                  WHEN 'float8' THEN 'float8'
	                  WHEN 'bool' THEN 'bool'
	                  ELSE pg_catalog.format_type(attribute.atttypid, attribute.atttypmod)
	              END,
	              '^character varying',
	              'varchar'
	          ),
	          '^character',
	          'char'
	      ) AS data_type,
	      CASE attribute.attidentity
	          WHEN 'a' THEN 'ALWAYS'
	          WHEN 'd' THEN 'BY DEFAULT'
	          ELSE NULL
	      END AS identity,
	      CASE
	          WHEN attribute.attcollation = 0 THEN NULL
	          WHEN attribute.attcollation = type_catalog.typcollation THEN 'default'
	          ELSE quote_ident(collation_schema.nspname)
	               || '.'
	               || quote_ident(collation_catalog.collname)
	      END AS collation_name,
	      attribute.attnotnull AS not_null,
	      pg_catalog.pg_get_expr(default_value.adbin, default_value.adrelid) AS default_expression,
	      pg_catalog.col_description(attribute.attrelid, attribute.attnum) AS comment,
	      COALESCE(pk.constraint_type IS NOT NULL, false) AS is_primary_key
	  FROM pg_catalog.pg_attribute AS attribute
	  JOIN pg_catalog.pg_class AS relation
	      ON relation.oid = attribute.attrelid
	  JOIN pg_catalog.pg_namespace AS relation_schema
	      ON relation_schema.oid = relation.relnamespace
	  JOIN pg_catalog.pg_type AS type_catalog
	      ON type_catalog.oid = attribute.atttypid
	  LEFT JOIN pg_catalog.pg_attrdef AS default_value
	      ON default_value.adrelid = attribute.attrelid
	     AND default_value.adnum = attribute.attnum
	  LEFT JOIN pg_catalog.pg_collation AS collation_catalog
	      ON collation_catalog.oid = attribute.attcollation
	  LEFT JOIN pg_catalog.pg_namespace AS collation_schema
	      ON collation_schema.oid = collation_catalog.collnamespace
	  LEFT JOIN (
	      SELECT conrelid, unnest(conkey) AS conkey_attnum, contype AS constraint_type
	      FROM pg_catalog.pg_constraint
	      WHERE contype = 'p'
	  ) AS pk ON pk.conrelid = attribute.attrelid AND pk.conkey_attnum = attribute.attnum
	  WHERE relation_schema.nspname = $1
	    AND relation.relname = $2
	    AND attribute.attnum > 0
	    AND NOT attribute.attisdropped
	  ORDER BY attribute.attnum`

const listIndexColumnsSQL = `SELECT
		index_class.relname AS index_name,
		pg_get_indexdef(index_class.oid, key.position, true) AS column,
		table_class.relname AS table_name,
		access_method.amname AS access_method
	FROM pg_index AS index_info
	JOIN pg_class AS table_class
		ON table_class.oid = index_info.indrelid
	JOIN pg_class AS index_class
		ON index_class.oid = index_info.indexrelid
	JOIN pg_namespace AS schema
		ON schema.oid = table_class.relnamespace
	JOIN pg_am AS access_method
		ON access_method.oid = index_class.relam
	CROSS JOIN LATERAL generate_series(1, index_info.indnkeyatts) AS key(position)
	WHERE schema.nspname = $1
		AND table_class.relname = $2
	ORDER BY index_name, key.position`

const listViewsSQL = `SELECT viewname AS view_name
	FROM pg_catalog.pg_views
	WHERE schemaname = $1
	ORDER BY viewname`

const listMaterializedViewsSQL = `SELECT matviewname AS materialized_view_name
  FROM pg_catalog.pg_matviews
  WHERE schemaname = $1
  ORDER BY matviewname`

const listFunctionsSQL = `SELECT
    routine.proname AS function_name,
    pg_get_function_identity_arguments(routine.oid) AS arguments,
    pg_get_function_result(routine.oid) AS returns,
    language.lanname AS language,
    pg_get_functiondef(routine.oid) AS definition
  FROM pg_proc AS routine
  JOIN pg_namespace AS namespace ON namespace.oid = routine.pronamespace
  JOIN pg_language AS language ON language.oid = routine.prolang
  WHERE routine.prokind = 'f'
    AND namespace.nspname = $1
  ORDER BY routine.proname`

type postgresql struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
	name   string
}

// Connect opens a PostgreSQL database using dsn and verifies that it is reachable.
func Connect(ctx context.Context, dsn string) (db.Database, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL DSN: %w", err)
	}

	logger, err := logger.Open()
	if err != nil {
		return nil, fmt.Errorf("open query log: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		_ = logger.Close()
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = logger.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return &postgresql{pool: pool, logger: logger, name: poolConfig.ConnConfig.Database}, nil
}

// Name returns the configured PostgreSQL database name.
func (p *postgresql) Name() string {
	return p.name
}

// Engine returns the PostgreSQL database engine identifier.
func (p *postgresql) Engine() string {
	return db.EnginePostgreSQL
}

// Host returns the configured PostgreSQL network host.
func (p *postgresql) Host() string {
	return p.pool.Config().ConnConfig.Host
}

// ListSchemaObjectGroups returns the non-empty object categories in every visible schema.
func (p *postgresql) ListSchemaObjectGroups(ctx context.Context) ([]db.SchemaObjectGroup, error) {
	p.logger.Log(listSchemaObjectGroupsSQL)
	rows, err := p.pool.Query(ctx, listSchemaObjectGroupsSQL)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL schema object groups: %w", err)
	}
	defer rows.Close()

	groups := make([]db.SchemaObjectGroup, 0)
	for rows.Next() {
		var group db.SchemaObjectGroup
		var objectType string
		if err := rows.Scan(&group.Schema, &objectType); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL schema object group: %w", err)
		}
		group.Type = db.SchemaObjectType(objectType)
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL schema object groups: %w", err)
	}

	return groups, nil
}

// ListTables returns the base tables in schema.
func (p *postgresql) ListTables(ctx context.Context, schema string) ([]db.Table, error) {
	p.logger.Log(listTablesSQL)
	rows, err := p.pool.Query(ctx, listTablesSQL, schema)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL tables: %w", err)
	}
	defer rows.Close()

	tables := make([]db.Table, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL table: %w", err)
		}
		tables = append(tables, db.Table{Schema: schema, Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL tables: %w", err)
	}

	return tables, nil
}

// ListViews returns ordinary PostgreSQL views in schema in alphabetical order.
func (p *postgresql) ListViews(ctx context.Context, schema string) ([]db.View, error) {
	p.logger.Log(listViewsSQL)
	rows, err := p.pool.Query(ctx, listViewsSQL, schema)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL views: %w", err)
	}
	defer rows.Close()

	views := make([]db.View, 0)
	for rows.Next() {
		var view db.View
		if err := rows.Scan(&view.Name); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL view: %w", err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL view: %w", err)
	}

	return views, nil
}

// ListMaterializedViews returns PostgreSQL materialized views in schema.
func (p *postgresql) ListMaterializedViews(ctx context.Context, schema string) ([]db.MaterializedView, error) {
	p.logger.Log(listMaterializedViewsSQL)
	rows, err := p.pool.Query(ctx, listMaterializedViewsSQL, schema)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL materialized views: %w", err)
	}
	defer rows.Close()

	materializedViews := make([]db.MaterializedView, 0)
	for rows.Next() {
		var view db.MaterializedView
		if err := rows.Scan(&view.Name); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL materialized view: %w", err)
		}
		materializedViews = append(materializedViews, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL materialized view: %w", err)
	}

	return materializedViews, nil
}

// ListFunctions returns functions in schema ordered by name.
func (p *postgresql) ListFunctions(ctx context.Context, schema string) ([]db.FunctionColumns, error) {
	rows, err := p.pool.Query(ctx, listFunctionsSQL, schema)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL functions: %w", err)
	}
	defer rows.Close()

	functionColumns := make([]db.FunctionColumns, 0)
	for rows.Next() {
		var function db.FunctionColumns
		if err := rows.Scan(&function.Name, &function.Arguments, &function.ReturnType, &function.Language, &function.Definition); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL function: %w", err)
		}
		functionColumns = append(functionColumns, function)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL functions: %w", err)
	}

	return functionColumns, nil
}

// ListColumns returns the columns defined by a PostgreSQL table.
func (p *postgresql) ListColumns(ctx context.Context, table db.Table) ([]db.Column, error) {
	p.logger.Log(listColumnsSQL)
	rows, err := p.pool.Query(ctx, listColumnsSQL, table.Schema, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL columns: %w", err)
	}
	defer rows.Close()

	columns := make([]db.Column, 0)
	for rows.Next() {
		var (
			column                            db.Column
			ordinalPosition                   int64
			identity, collation, defaultValue pgtype.Text
			comment                           pgtype.Text
		)

		err := rows.Scan(
			&column.Name,
			&ordinalPosition,
			&column.DataType,
			&identity,
			&collation,
			&column.NotNull,
			&defaultValue,
			&comment,
			&column.IsPrimaryKey,
		)
		if err != nil {
			return nil, fmt.Errorf("scan PostgreSQL column: %w", err)
		}

		column.OrdinalPosition = int(ordinalPosition)
		column.Identity = identity.String
		column.Collation = collation.String
		column.Default = defaultValue.String
		column.Comment = comment.String

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL columns: %w", err)
	}

	return columns, nil
}

// ListIndexes returns every indexed column of a PostgreSQL table.
func (p *postgresql) ListIndexes(ctx context.Context, table db.Table) ([]db.IndexColumns, error) {
	p.logger.Log(listIndexColumnsSQL)
	rows, err := p.pool.Query(ctx, listIndexColumnsSQL, table.Schema, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL Index columns: %w", err)
	}
	defer rows.Close()

	indexColumns := make([]db.IndexColumns, 0)

	for rows.Next() {
		var index db.IndexColumns

		err := rows.Scan(
			&index.Name,
			&index.Column,
			&index.Table,
			&index.AccessMethod,
		)
		if err != nil {
			return nil, fmt.Errorf("scan PostgreSQL index column: %w", err)
		}

		indexColumns = append(indexColumns, index)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL index columns: %w", err)
	}

	return indexColumns, nil

}

// GetRows returns an unordered page of rows from a public PostgreSQL table.
func (p *postgresql) GetRows(ctx context.Context, table db.Table, page db.PageRequest) (db.RowPage, error) {
	return p.getRows(ctx, table, &page)
}

// getRows returns all rows when page is nil, or a bounded page when it is provided.
func (p *postgresql) getRows(ctx context.Context, table db.Table, page *db.PageRequest) (db.RowPage, error) {
	if table.Name == "" {
		return db.RowPage{}, errors.New("table name is required")
	}
	if page != nil {
		if page.Offset < 0 {
			return db.RowPage{}, errors.New("page offset cannot be negative")
		}
		if page.Limit < 1 {
			return db.RowPage{}, errors.New("page limit must be positive")
		}
	}

	tableName := pgx.Identifier{table.Schema, table.Name}.Sanitize()
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	args := make([]any, 0, 2)
	if page != nil {
		query += " LIMIT $1 OFFSET $2"
		queryLimit := page.Limit + 1
		args = append(args, queryLimit, page.Offset)
		p.logger.Log(fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, queryLimit, page.Offset))
	} else {
		p.logger.Log(query)
	}

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return db.RowPage{}, fmt.Errorf("query PostgreSQL rows: %w", err)
	}
	defer rows.Close()

	result := db.RowPage{Columns: make([]string, len(rows.FieldDescriptions()))}
	for index, description := range rows.FieldDescriptions() {
		result.Columns[index] = description.Name
	}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return db.RowPage{}, fmt.Errorf("read PostgreSQL row: %w", err)
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return db.RowPage{}, fmt.Errorf("iterate PostgreSQL rows: %w", err)
	}

	if page != nil && len(result.Rows) > page.Limit {
		result.HasMore = true
		result.Rows = result.Rows[:page.Limit]
	}

	return result, nil
}

// Execute runs arbitrary SQL and returns its first 100 rows and command status.
func (p *postgresql) Execute(ctx context.Context, sql string) (db.QueryResult, error) {
	p.logger.Log(sql)
	rows, err := p.pool.Query(ctx, sql, pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("execute PostgreSQL query: %w", err)
	}

	return readQueryResult(rows, db.MaxPageSize)
}

func readQueryResult(rows pgx.Rows, rowLimit int) (db.QueryResult, error) {
	defer rows.Close()

	result := db.QueryResult{Columns: make([]string, len(rows.FieldDescriptions()))}
	for index, field := range rows.FieldDescriptions() {
		result.Columns[index] = field.Name
	}
	for rows.Next() {
		if rowLimit > 0 && len(result.Rows) == rowLimit {
			break
		}
		values, err := rows.Values()
		if err != nil {
			return db.QueryResult{}, fmt.Errorf("read PostgreSQL query row: %w", err)
		}
		result.Rows = append(result.Rows, values)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return db.QueryResult{}, fmt.Errorf("iterate PostgreSQL query rows: %w", err)
	}
	result.CommandTag = rows.CommandTag().String()
	return result, nil
}

// Close releases all connections held by the database.
func (p *postgresql) Close() {
	p.pool.Close()
	_ = p.logger.Close()
}

func (p *postgresql) Dump(ctx context.Context) error {
	connConfig := p.pool.Config().ConnConfig

	filename := db.TimestampedFilename(db.SafeFilename(connConfig.Database), "sql")
	cmd := exec.CommandContext(
		ctx,
		"pg_dump",
		"-h", connConfig.Host,
		"-p", strconv.Itoa(int(connConfig.Port)),
		"-U", connConfig.User,
		"-d", connConfig.Database,
		"-f", filename,
	)

	if connConfig.Password != "" {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+connConfig.Password)
	}

	_, err := cmd.CombinedOutput()
	if err != nil {
		// If command fails try docker exec
		containerID, err := dockerContainerIDForPort(ctx, int(connConfig.Port))
		if err != nil {
			return err
		}

		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("create dump file: %w", err)
		}

		var stderr bytes.Buffer
		args := []string{"exec"}
		if connConfig.Password != "" {
			args = append(args, "-e", "PGPASSWORD="+connConfig.Password)
		}
		args = append(args,
			containerID,
			"pg_dump",
			"-U", connConfig.User,
			"-d", connConfig.Database,
		)
		cmd := exec.CommandContext(ctx, "docker", args...)

		cmd.Stdout = file
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			_ = file.Close()
			_ = os.Remove(filename)
			return fmt.Errorf("docker pg_dump: %w: %s", err, stderr.String())
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(filename)
			return fmt.Errorf("close dump file: %w", err)
		}

		return nil
	}

	return nil
}

func (p *postgresql) Export(ctx context.Context, table db.Table, typeVal string) error {
	data, err := p.getRows(ctx, table, nil)
	if err != nil {
		return err
	}

	switch typeVal {
	case db.ExportTypeCSV:
		filename := db.TimestampedFilename(db.SafeFilename(table.Name), db.ExportTypeCSV)
		if err := csvexport.Write(filename, data.Columns, data.Rows); err != nil {
			return fmt.Errorf("write CSV export: %w", err)
		}
	case db.ExportTypeJSON:
		filename := db.TimestampedFilename(db.SafeFilename(table.Name), db.ExportTypeJSON)
		normalizeJSONValues(data.Rows)
		if err := jsonexport.Write(filename, table.Name, data.Columns, data.Rows); err != nil {
			return fmt.Errorf("write JSON export: %w", err)
		}
	}

	return nil
}

func normalizeJSONValues(rows [][]any) {
	for _, row := range rows {
		for index, value := range row {
			if infinity, ok := value.(pgtype.InfinityModifier); ok {
				row[index] = infinity.String()
			}
		}
	}
}

// ExportQuery re-runs a SELECT query in a read-only transaction and writes all rows to CSV.
func (p *postgresql) ExportQuery(ctx context.Context, statement string) error {
	if err := db.ValidateSelectQuery(statement); err != nil {
		return err
	}

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("begin PostgreSQL export transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	p.logger.Log(statement)
	rows, err := tx.Query(ctx, statement, pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		return fmt.Errorf("query PostgreSQL export rows: %w", err)
	}

	result, err := readQueryResult(rows, 0)
	if err != nil {
		return err
	}

	filename := db.TimestampedFilename("query", "csv")
	if err := csvexport.Write(filename, result.Columns, result.Rows); err != nil {
		return fmt.Errorf("write CSV query export: %w", err)
	}
	return nil
}

func dockerContainerIDForPort(ctx context.Context, port int) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"ps",
		"--filter", fmt.Sprintf("publish=%d", port),
		"--format", "{{.ID}}",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("list Docker containers: %w", err)
	}

	ids := strings.Fields(string(output))
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("not found")
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("multiple Docker containers publish port %d", port)
	}
}

func (p *postgresql) UpdateRow(ctx context.Context, table db.Table, setColumns map[string]any, whereColumns map[string]any) error {
	tableIdent := pgx.Identifier{table.Schema, table.Name}.Sanitize()

	setClauses := make([]string, 0, len(setColumns))
	args := make([]any, 0, len(setColumns)+len(whereColumns))
	argIdx := 1

	setNames := make([]string, 0, len(setColumns))
	for name := range setColumns {
		setNames = append(setNames, name)
	}
	sort.Strings(setNames)
	for _, name := range setNames {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", pgx.Identifier{name}.Sanitize(), argIdx))
		args = append(args, setColumns[name])
		argIdx++
	}

	whereClauses := make([]string, 0, len(whereColumns))
	whereNames := make([]string, 0, len(whereColumns))
	for name := range whereColumns {
		whereNames = append(whereNames, name)
	}
	sort.Strings(whereNames)
	for _, name := range whereNames {
		if whereColumns[name] == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", pgx.Identifier{name}.Sanitize()))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", pgx.Identifier{name}.Sanitize(), argIdx))
			args = append(args, whereColumns[name])
			argIdx++
		}
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableIdent,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	p.logger.Log(query)
	tag, err := p.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update PostgreSQL row: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("no row matched the WHERE clause; the row may have been modified or deleted")
	}
	return nil
}

func (p *postgresql) DeleteRow(ctx context.Context, table db.Table, whereColumns map[string]any) error {
	tableIdent := pgx.Identifier{table.Schema, table.Name}.Sanitize()

	columns, err := p.ListColumns(ctx, table)
	if err != nil {
		return fmt.Errorf("list PostgreSQL columns before delete: %w", err)
	}
	if err := db.ValidatePrimaryKeyWhere("delete PostgreSQL row", columns, whereColumns); err != nil {
		return err
	}

	whereNames := make([]string, 0, len(whereColumns))
	for name := range whereColumns {
		whereNames = append(whereNames, name)
	}
	sort.Strings(whereNames)

	whereClauses := make([]string, 0, len(whereNames))
	args := make([]any, 0, len(whereNames))

	for index, name := range whereNames {
		column := pgx.Identifier{name}.Sanitize()

		if whereColumns[name] == nil {
			whereClauses = append(whereClauses, column+" IS NULL")
			continue
		}

		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", column, index+1))
		args = append(args, whereColumns[name])
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		tableIdent,
		strings.Join(whereClauses, " AND "),
	)

	t, err := p.pool.Exec(ctx, query, args...)

	if err != nil {
		return fmt.Errorf("update PostgreSQL row: %w", err)
	}
	if t.RowsAffected() == 0 {
		return errors.New("no row matched the WHERE clause; the row may have been modified or deleted")
	}
	return nil

}
