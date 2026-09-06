// Package sqlserver provides the Microsoft SQL Server database adapter.
package sqlserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/csvexport"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/jsonexport"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
)

const sqlServerBackupDirectory = "/var/opt/mssql/data"

const listTablesSQL = `
        SELECT TABLE_NAME
        FROM INFORMATION_SCHEMA.TABLES
        WHERE TABLE_SCHEMA = @p1
                AND TABLE_TYPE = 'BASE TABLE'
        ORDER BY TABLE_NAME`

const listSchemaObjectGroupsSQL = `
	SELECT schema_name, object_type
	FROM (
		SELECT schema_info.name AS schema_name, 'tables' AS object_type
		FROM sys.tables AS table_info
		JOIN sys.schemas AS schema_info
			ON schema_info.schema_id = table_info.schema_id
		WHERE table_info.is_ms_shipped = 0

		UNION

		-- Indexed views are ordinary views to this application: SQL Server has
		-- no materialized views, and ListViews returns them, so excluding them
		-- here would hide every view in a schema whose views are all indexed.
		SELECT schema_info.name AS schema_name, 'views' AS object_type
		FROM sys.views AS view_info
		JOIN sys.schemas AS schema_info
			ON schema_info.schema_id = view_info.schema_id
		WHERE view_info.is_ms_shipped = 0

		UNION

		SELECT schema_info.name AS schema_name, 'functions' AS object_type
		FROM sys.objects AS object_info
		JOIN sys.schemas AS schema_info
			ON schema_info.schema_id = object_info.schema_id
		WHERE object_info.is_ms_shipped = 0
			AND object_info.type IN ('FN', 'FS', 'FT', 'IF', 'TF')
	) AS object_groups
	WHERE schema_name NOT IN ('sys', 'INFORMATION_SCHEMA')
	ORDER BY schema_name,
		CASE object_type
			WHEN 'tables' THEN 1
			WHEN 'views' THEN 2
			WHEN 'functions' THEN 3
		END
`

const listColumnsSQL = `
	SELECT
		column_info.name AS column_name,
		column_info.column_id AS ordinal_position,
		type_info.name +
			CASE
				WHEN type_info.name IN ('char', 'varchar', 'binary', 'varbinary', 'nchar', 'nvarchar') THEN
					'(' + CASE
						WHEN column_info.max_length = -1 THEN 'max'
						WHEN type_info.name IN ('nchar', 'nvarchar') THEN CONVERT(varchar(10), column_info.max_length / 2)
						ELSE CONVERT(varchar(10), column_info.max_length)
					END + ')'
				WHEN type_info.name IN ('decimal', 'numeric') THEN
					'(' + CONVERT(varchar(10), column_info.precision) +
					',' + CONVERT(varchar(10), column_info.scale) + ')'
				WHEN type_info.name IN ('datetime2', 'datetimeoffset', 'time') THEN
					'(' + CONVERT(varchar(10), column_info.scale) + ')'
				ELSE ''
			END AS data_type,
		CASE WHEN column_info.is_identity = 1 THEN 'IDENTITY' ELSE '' END AS identity_value,
		COALESCE(column_info.collation_name, '') AS collation,
		CAST(CASE WHEN column_info.is_nullable = 0 THEN 1 ELSE 0 END AS bit) AS not_null,
		COALESCE(CONVERT(nvarchar(max), OBJECT_DEFINITION(column_info.default_object_id)), '') AS default_value,
		COALESCE(CONVERT(nvarchar(max), property_info.value), '') AS comment,
		CAST(CASE WHEN primary_key.column_id IS NULL THEN 0 ELSE 1 END AS bit) AS is_primary_key
	FROM sys.columns AS column_info
	JOIN sys.tables AS table_info
		ON table_info.object_id = column_info.object_id
	JOIN sys.schemas AS schema_info
		ON schema_info.schema_id = table_info.schema_id
	JOIN sys.types AS type_info
		ON type_info.user_type_id = column_info.user_type_id
	LEFT JOIN sys.indexes AS primary_index
		ON primary_index.object_id = table_info.object_id
			AND primary_index.is_primary_key = 1
	LEFT JOIN sys.index_columns AS primary_key
		ON primary_key.object_id = primary_index.object_id
			AND primary_key.index_id = primary_index.index_id
			AND primary_key.column_id = column_info.column_id
	LEFT JOIN sys.extended_properties AS property_info
		ON property_info.class = 1
			AND property_info.major_id = column_info.object_id
			AND property_info.minor_id = column_info.column_id
			AND property_info.name = 'MS_Description'
	WHERE schema_info.name = @p1
		AND table_info.name = @p2
	ORDER BY column_info.column_id`

const listIndexColumnsSQL = `
    SELECT
            index_info.name AS index_name,
            column_info.name AS column_name,
            table_info.name AS table_name,
            index_info.type_desc AS access_method
    FROM sys.indexes AS index_info
    JOIN sys.tables AS table_info
            ON table_info.object_id = index_info.object_id
    JOIN sys.schemas AS schema_info
            ON schema_info.schema_id = table_info.schema_id
    JOIN sys.index_columns AS index_column
            ON index_column.object_id = index_info.object_id
                    AND index_column.index_id = index_info.index_id
                    AND index_column.key_ordinal > 0
    JOIN sys.columns AS column_info
            ON column_info.object_id = index_column.object_id
                    AND column_info.column_id = index_column.column_id
    WHERE schema_info.name = @p1
            AND table_info.name = @p2
            AND index_info.is_hypothetical = 0
    ORDER BY index_info.name, index_column.key_ordinal`

const listViewsSQL = `
	SELECT
        view_info.name AS view_name
        FROM sys.views AS view_info
        JOIN sys.schemas AS schema_info
                ON schema_info.schema_id = view_info.schema_id
        WHERE schema_info.name = @p1
                AND view_info.is_ms_shipped = 0
        ORDER BY view_info.name`

const listFunctionsSQL = `
	SELECT
		function_info.name AS function_name,
		COALESCE(argument_info.arguments, '') AS arguments,
		CASE
			WHEN function_info.type IN ('IF', 'TF', 'FT') THEN 'TABLE'
			ELSE return_type_info.name +
				CASE
					WHEN return_type_info.name IN ('char', 'varchar', 'binary', 'varbinary', 'nchar', 'nvarchar') THEN
						'(' + CASE
							WHEN return_parameter.max_length = -1 THEN 'max'
							WHEN return_type_info.name IN ('nchar', 'nvarchar') THEN CONVERT(varchar(10), return_parameter.max_length / 2)
							ELSE CONVERT(varchar(10), return_parameter.max_length)
						END + ')'
					WHEN return_type_info.name IN ('decimal', 'numeric') THEN
						'(' + CONVERT(varchar(10), return_parameter.precision) +
						',' + CONVERT(varchar(10), return_parameter.scale) + ')'
					WHEN return_type_info.name IN ('datetime2', 'datetimeoffset', 'time') THEN
						'(' + CONVERT(varchar(10), return_parameter.scale) + ')'
					ELSE ''
				END
		END AS return_type,
		CASE
			WHEN function_info.type IN ('FS', 'FT') THEN 'CLR'
			ELSE 'SQL'
		END AS language,
		COALESCE(module_info.definition, '') AS definition
	FROM sys.objects AS function_info
	JOIN sys.schemas AS schema_info
		ON schema_info.schema_id = function_info.schema_id
	LEFT JOIN sys.parameters AS return_parameter
		ON return_parameter.object_id = function_info.object_id
			AND return_parameter.parameter_id = 0
	LEFT JOIN sys.types AS return_type_info
		ON return_type_info.user_type_id = return_parameter.user_type_id
	LEFT JOIN sys.sql_modules AS module_info
		ON module_info.object_id = function_info.object_id
	OUTER APPLY (
		SELECT STRING_AGG(
			parameter_info.name + ' ' + parameter_type_info.name +
				CASE
					WHEN parameter_type_info.name IN ('char', 'varchar', 'binary', 'varbinary', 'nchar', 'nvarchar') THEN
						'(' + CASE
							WHEN parameter_info.max_length = -1 THEN 'max'
							WHEN parameter_type_info.name IN ('nchar', 'nvarchar') THEN CONVERT(varchar(10), parameter_info.max_length / 2)
							ELSE CONVERT(varchar(10), parameter_info.max_length)
						END + ')'
					WHEN parameter_type_info.name IN ('decimal', 'numeric') THEN
						'(' + CONVERT(varchar(10), parameter_info.precision) +
						',' + CONVERT(varchar(10), parameter_info.scale) + ')'
					WHEN parameter_type_info.name IN ('datetime2', 'datetimeoffset', 'time') THEN
						'(' + CONVERT(varchar(10), parameter_info.scale) + ')'
					ELSE ''
				END +
				CASE WHEN parameter_info.is_output = 1 THEN ' OUTPUT' ELSE '' END,
			', '
		) WITHIN GROUP (ORDER BY parameter_info.parameter_id) AS arguments
		FROM sys.parameters AS parameter_info
		JOIN sys.types AS parameter_type_info
			ON parameter_type_info.user_type_id = parameter_info.user_type_id
		WHERE parameter_info.object_id = function_info.object_id
			AND parameter_info.parameter_id > 0
	) AS argument_info
	WHERE schema_info.name = @p1
		AND function_info.type IN ('FN', 'FS', 'FT', 'IF', 'TF')
	ORDER BY function_info.name`

type sqlserverDatabase struct {
	database *sql.DB
	config   msdsn.Config
}

var _ db.Database = (*sqlserverDatabase)(nil)

func Connect(ctx context.Context, dsn string) (db.Database, error) {
	config, err := msdsn.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse SQL Server DSN: %w", err)
	}

	connector := mssql.NewConnectorConfig(config)
	database := sql.OpenDB(connector)

	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("connect to SQL Server: %w", err)
	}

	return &sqlserverDatabase{
		database: database,
		config:   config,
	}, nil
}

func (s *sqlserverDatabase) Name() string {
	return s.config.Database
}

func (s *sqlserverDatabase) Engine() string {
	return db.EngineSQLServer
}

func (s *sqlserverDatabase) Host() string {
	return s.config.Host
}

// ListTables returns the base tables in schema.
func (s *sqlserverDatabase) ListTables(ctx context.Context, schema string) ([]db.Table, error) {
	rows, err := s.database.QueryContext(ctx, listTablesSQL, schema)
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

func (s *sqlserverDatabase) ListSchemaObjectGroups(ctx context.Context) ([]db.SchemaObjectGroup, error) {
	rows, err := s.database.QueryContext(ctx, listSchemaObjectGroupsSQL)
	if err != nil {
		return nil, fmt.Errorf("query SQL Server schema object groups: %w", err)
	}
	defer rows.Close()

	groups := make([]db.SchemaObjectGroup, 0)
	for rows.Next() {
		var objectType string
		var group db.SchemaObjectGroup
		if err := rows.Scan(&group.Schema, &objectType); err != nil {
			return nil, fmt.Errorf("scan SQL Server schema object group: %w", err)
		}
		group.Type = db.SchemaObjectType(objectType)
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SQL Server schema object groups: %w", err)
	}

	return groups, nil
}

func (s *sqlserverDatabase) GetRows(ctx context.Context, table db.Table, page db.PageRequest) (db.RowPage, error) {
	return s.getRows(ctx, table, &page)
}

// getRows returns all rows when page is nil, or a bounded page when it is provided.
func (s *sqlserverDatabase) getRows(ctx context.Context, table db.Table, page *db.PageRequest) (db.RowPage, error) {
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

	tableName := quoteIdentifier(table.Name)
	if table.Schema != "" {
		tableName = quoteIdentifier(table.Schema) + "." + tableName
	}

	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	args := make([]any, 0, 2)

	if page != nil {
		// SQL Server requires ORDER BY when using OFFSET/FETCH.
		query += " ORDER BY (SELECT NULL) OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY"
		args = append(args, page.Offset, page.Limit+1)
	}

	rows, err := s.database.QueryContext(ctx, query, args...)
	if err != nil {
		return db.RowPage{}, fmt.Errorf("query SQL Server rows: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return db.RowPage{}, fmt.Errorf("read SQL Server row columns: %w", err)
	}

	result := db.RowPage{Columns: columns}
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for index := range values {
			destinations[index] = &values[index]
		}

		if err := rows.Scan(destinations...); err != nil {
			return db.RowPage{}, fmt.Errorf("read SQL Server row: %w", err)
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return db.RowPage{}, fmt.Errorf("iterate SQL Server rows: %w", err)
	}

	if page != nil && len(result.Rows) > page.Limit {
		result.HasMore = true
		result.Rows = result.Rows[:page.Limit]
	}

	return result, nil
}

func quoteIdentifier(identifier string) string {
	return "[" + strings.ReplaceAll(identifier, "]", "]]") + "]"
}

func (s *sqlserverDatabase) TableDDL(ctx context.Context, table db.Table) (string, error) {
	if table.Name == "" {
		return "", errors.New("table name is required")
	}

	metadata, err := s.loadTableDDL(ctx, table)
	if err != nil {
		return "", err
	}

	return buildSQLServerTableDDL(metadata)
}

// Execute runs arbitrary SQL and returns up to db.MaxPageSize rows.
func (s *sqlserverDatabase) Execute(ctx context.Context, statement string) (db.QueryResult, error) {
	rows, err := s.database.QueryContext(ctx, statement)
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("execute SQL Server query: %w", err)
	}
	return readSQLServerQueryResult(rows, db.MaxPageSize, sqlServerCommandTag(statement))
}

func readSQLServerQueryResult(rows *sql.Rows, rowLimit int, commandTag string) (db.QueryResult, error) {
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("read SQL Server query columns: %w", err)
	}
	result := db.QueryResult{Columns: columns, CommandTag: commandTag}
	for rows.Next() {
		if rowLimit > 0 && len(result.Rows) == rowLimit {
			break
		}
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			return db.QueryResult{}, fmt.Errorf("read SQL Server query row: %w", err)
		}
		for index, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[index] = append([]byte(nil), bytes...)
			}
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return db.QueryResult{}, fmt.Errorf("iterate SQL Server query rows: %w", err)
	}
	return result, nil
}

func sqlServerCommandTag(statement string) string {
	fields := strings.Fields(statement)
	if len(fields) == 0 {
		return ""
	}
	command := strings.ToUpper(strings.Trim(fields[0], "();"))
	if (command == "CREATE" || command == "ALTER" || command == "DROP") && len(fields) > 1 {
		return command + " " + strings.ToUpper(strings.Trim(fields[1], "();[]"))
	}
	return command
}

// Dump writes the connected SQL Server database to a timestamped BAK file.
//
// SQL Server creates the backup inside its Docker container, then the adapter
// copies the file to the current directory and removes the temporary copy.
func (s *sqlserverDatabase) Dump(ctx context.Context) error {
	if s.config.Database == "" {
		return errors.New("SQL Server database name is required")
	}

	filename := db.TimestampedFilename(db.SafeFilename(s.config.Database), "bak")
	containerID, err := dockerContainerIDForPort(ctx, strconv.FormatUint(s.config.Port, 10))
	if err != nil {
		return fmt.Errorf("find SQL Server Docker container: %w", err)
	}

	containerFilename := sqlServerBackupDirectory + "/" + filename
	if _, err := s.database.ExecContext(ctx, sqlServerBackupSQL(s.config.Database, containerFilename)); err != nil {
		return fmt.Errorf("back up SQL Server database: %w", err)
	}

	if err := copyFromDocker(ctx, containerID, containerFilename, filename); err != nil {
		_ = removeFromDocker(ctx, containerID, containerFilename)
		return err
	}
	if err := removeFromDocker(ctx, containerID, containerFilename); err != nil {
		return err
	}

	return nil
}

func sqlServerBackupSQL(databaseName, filename string) string {
	return fmt.Sprintf(
		"BACKUP DATABASE %s TO DISK = N'%s' WITH COPY_ONLY, INIT;",
		quoteIdentifier(databaseName),
		strings.ReplaceAll(filename, "'", "''"),
	)
}

func dockerContainerIDForPort(ctx context.Context, port string) (string, error) {
	command := exec.CommandContext(
		ctx,
		"docker",
		"ps",
		"--filter", "publish="+port,
		"--format", "{{.ID}}",
	)
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("list Docker containers: %w", err)
	}

	ids := strings.Fields(string(output))
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("no Docker container publishes port %s; SQL Server dump requires the server to run in a local Docker container", port)
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("multiple Docker containers publish port %s", port)
	}
}

func copyFromDocker(ctx context.Context, containerID, source, destination string) error {
	output, err := exec.CommandContext(ctx, "docker", "cp", containerID+":"+source, destination).CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy SQL Server backup from Docker: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func removeFromDocker(ctx context.Context, containerID, filename string) error {
	output, err := exec.CommandContext(ctx, "docker", "exec", containerID, "rm", "-f", filename).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove temporary SQL Server backup from Docker: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *sqlserverDatabase) Export(ctx context.Context, table db.Table, typeVal string) error {
	data, err := s.getRows(ctx, table, nil)
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
		if err := jsonexport.Write(filename, table.Name, data.Columns, data.Rows); err != nil {
			return fmt.Errorf("write JSON export: %w", err)
		}
	}

	return nil
}

func (s *sqlserverDatabase) ExportQuery(ctx context.Context, statement string) error {
	if err := db.ValidateSelectQuery(statement); err != nil {
		return err
	}

	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin SQL Server export transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return fmt.Errorf("query SQL Server export rows: %w", err)
	}

	result, err := readSQLServerQueryResult(rows, 0, "")
	if err != nil {
		return err
	}

	filename := db.TimestampedFilename("query", db.ExportTypeCSV)
	if err := csvexport.Write(filename, result.Columns, result.Rows); err != nil {
		return fmt.Errorf("write CSV query export: %w", err)
	}

	return nil
}

func (s *sqlserverDatabase) ListColumns(ctx context.Context, table db.Table) ([]db.Column, error) {
	rows, err := s.database.QueryContext(ctx, listColumnsSQL, table.Schema, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query sqlserver columns: %w", err)
	}
	defer rows.Close()

	columns := make([]db.Column, 0)
	for rows.Next() {
		var column db.Column
		if err := rows.Scan(
			&column.Name,
			&column.OrdinalPosition,
			&column.DataType,
			&column.Identity,
			&column.Collation,
			&column.NotNull,
			&column.Default,
			&column.Comment,
			&column.IsPrimaryKey,
		); err != nil {
			return nil, fmt.Errorf("scan SQL Server column: %w", err)
		}

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SQL Server columns: %w", err)
	}

	return columns, nil
}

func (s *sqlserverDatabase) ListIndexes(ctx context.Context, table db.Table) ([]db.IndexColumns, error) {
	rows, err := s.database.QueryContext(ctx, listIndexColumnsSQL, table.Schema, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query sqlserver Index columns: %w", err)
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
			return nil, fmt.Errorf("scan sqlserver index column: %w", err)
		}

		indexColumns = append(indexColumns, index)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlserver index columns: %w", err)
	}

	return indexColumns, nil
}

func (s *sqlserverDatabase) ListViews(ctx context.Context, schema string) ([]db.View, error) {
	rows, err := s.database.QueryContext(ctx, listViewsSQL, schema)
	if err != nil {
		return nil, fmt.Errorf("query sqlserver views: %w", err)
	}
	defer rows.Close()

	views := make([]db.View, 0)
	for rows.Next() {
		var view db.View
		if err := rows.Scan(&view.Name); err != nil {
			return nil, fmt.Errorf("scan sqlserver view: %w", err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlserver view: %w", err)
	}

	return views, nil
}

// ListMaterializedViews always reports no materialized views. SQL Server has
// indexed views rather than materialized views, so the application excludes it
// from the materialized-view navigator section and never calls this method.
func (s *sqlserverDatabase) ListMaterializedViews(context.Context, string) ([]db.MaterializedView, error) {
	return nil, nil
}

func (s *sqlserverDatabase) UpdateRow(ctx context.Context, table db.Table, setColumns, whereColumns map[string]any) error {
	columns, err := s.ListColumns(ctx, table)
	if err != nil {
		return fmt.Errorf("list SQL Server columns before update: %w", err)
	}
	if err := db.ValidatePrimaryKeyWhere("update SQL Server row", columns, whereColumns); err != nil {
		return err
	}
	if len(setColumns) == 0 {
		return errors.New("update SQL Server row requires at least one column")
	}

	tableName := quoteIdentifier(table.Name)
	if table.Schema != "" {
		tableName = quoteIdentifier(table.Schema) + "." + tableName
	}

	setNames := make([]string, 0, len(setColumns))
	for name := range setColumns {
		setNames = append(setNames, name)
	}
	sort.Strings(setNames)

	args := make([]any, 0, len(setColumns)+len(whereColumns))
	setClauses := make([]string, 0, len(setNames))
	for _, name := range setNames {
		args = append(args, setColumns[name])
		setClauses = append(setClauses, fmt.Sprintf("%s = @p%d", quoteIdentifier(name), len(args)))
	}

	whereNames := make([]string, 0, len(whereColumns))
	for name := range whereColumns {
		whereNames = append(whereNames, name)
	}
	sort.Strings(whereNames)

	whereClauses := make([]string, 0, len(whereNames))
	for _, name := range whereNames {
		if whereColumns[name] == nil {
			whereClauses = append(whereClauses, quoteIdentifier(name)+" IS NULL")
			continue
		}
		args = append(args, whereColumns[name])
		whereClauses = append(whereClauses, fmt.Sprintf("%s = @p%d", quoteIdentifier(name), len(args)))
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)
	result, err := s.database.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update SQL Server row: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read SQL Server update result: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("no row matched the WHERE clause; the row may have been modified or deleted")
	}
	return nil
}

func (s *sqlserverDatabase) DeleteRow(ctx context.Context, table db.Table, whereColumns map[string]any) error {
	columns, err := s.ListColumns(ctx, table)
	if err != nil {
		return fmt.Errorf("list SQL Server columns before delete: %w", err)
	}
	if err := db.ValidatePrimaryKeyWhere("delete SQL Server row", columns, whereColumns); err != nil {
		return err
	}

	tableName := quoteIdentifier(table.Name)
	if table.Schema != "" {
		tableName = quoteIdentifier(table.Schema) + "." + tableName
	}

	whereNames := make([]string, 0, len(whereColumns))
	for name := range whereColumns {
		whereNames = append(whereNames, name)
	}
	sort.Strings(whereNames)

	args := make([]any, 0, len(whereNames))
	whereClauses := make([]string, 0, len(whereNames))
	for _, name := range whereNames {
		if whereColumns[name] == nil {
			whereClauses = append(whereClauses, quoteIdentifier(name)+" IS NULL")
			continue
		}
		args = append(args, whereColumns[name])
		whereClauses = append(whereClauses, fmt.Sprintf("%s = @p%d", quoteIdentifier(name), len(args)))
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "),
	)
	result, err := s.database.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete SQL Server row: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read SQL Server delete result: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("no row matched the WHERE clause; the row may have been modified or deleted")
	}
	return nil
}

func (s *sqlserverDatabase) ListFunctions(ctx context.Context, schema string) ([]db.FunctionColumns, error) {
	rows, err := s.database.QueryContext(ctx, listFunctionsSQL, schema)
	if err != nil {
		return nil, fmt.Errorf("query sqlserver functions: %w", err)
	}
	defer rows.Close()

	functionColumns := make([]db.FunctionColumns, 0)
	for rows.Next() {
		var function db.FunctionColumns
		if err := rows.Scan(&function.Name, &function.Arguments, &function.ReturnType, &function.Language, &function.Definition); err != nil {
			return nil, fmt.Errorf("scan sqlserver function: %w", err)
		}
		functionColumns = append(functionColumns, function)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlserver functions: %w", err)
	}

	return functionColumns, nil
}

// Close releases all connections held by the database.
func (s *sqlserverDatabase) Close() {
	if s.database != nil {
		_ = s.database.Close()
	}
}
