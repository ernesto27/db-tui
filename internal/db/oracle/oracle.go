// Package oracle provides an Oracle implementation of db.Database.
package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/csvexport"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/jsonexport"
	"github.com/ernestoponce27/db-tui/internal/logger"
	_ "github.com/sijms/go-ora/v2"
)

const listTablesSQL = `
	SELECT table_name
	FROM user_tables
	WHERE table_name NOT IN (SELECT mview_name FROM user_mviews)
	ORDER BY table_name`

const listViewsSQL = `SELECT view_name FROM user_views ORDER BY view_name`

const listMaterializedViewsSQL = `SELECT mview_name FROM user_mviews ORDER BY mview_name`

const listColumnsSQL = `
	SELECT
		column_info.column_name,
		column_info.column_id,
		column_info.data_type,
		identity_info.generation_type,
		NULL AS collation_name,
		CASE column_info.nullable WHEN 'N' THEN 1 ELSE 0 END AS not_null,
		column_info.data_default,
		comment_info.comments
	FROM user_tab_columns column_info
	LEFT JOIN user_tab_identity_cols identity_info
		ON identity_info.table_name = column_info.table_name
		AND identity_info.column_name = column_info.column_name
	LEFT JOIN user_col_comments comment_info
		ON comment_info.table_name = column_info.table_name
		AND comment_info.column_name = column_info.column_name
	WHERE column_info.table_name = :1
	ORDER BY column_info.column_id`

const listIndexesSQL = `
	SELECT
		index_info.index_name,
		index_column.column_name,
		index_column.table_name,
		index_info.index_type
	FROM user_indexes index_info
	JOIN user_ind_columns index_column
		ON index_column.index_name = index_info.index_name
		AND index_column.table_name = index_info.table_name
	WHERE index_info.table_name = :1
	ORDER BY index_info.index_name, index_column.column_position`

type config struct {
	host    string
	service string
}

type oracleDatabase struct {
	database *sql.DB
	logger   *logger.Logger
	config   config
}

// Connect opens an Oracle database using an oracle:// Easy Connect URL.
func Connect(ctx context.Context, dsn string) (db.Database, error) {
	config, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse Oracle DSN: %w", err)
	}

	queryLogger, err := logger.Open()
	if err != nil {
		return nil, fmt.Errorf("open query log: %w", err)
	}

	database, err := sql.Open("oracle", dsn)
	if err != nil {
		_ = queryLogger.Close()
		return nil, fmt.Errorf("open Oracle database: %w", err)
	}
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		_ = queryLogger.Close()
		return nil, fmt.Errorf("connect to Oracle: %w", err)
	}

	return &oracleDatabase{database: database, logger: queryLogger, config: config}, nil
}

func parseDSN(dsn string) (config, error) {
	parsed, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return config{}, err
	}
	if !strings.EqualFold(parsed.Scheme, db.EngineOracle) {
		return config{}, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	if parsed.User == nil || parsed.User.Username() == "" {
		return config{}, errors.New("username is required")
	}
	if parsed.Hostname() == "" {
		return config{}, errors.New("host is required")
	}
	service := strings.TrimPrefix(parsed.EscapedPath(), "/")
	service, err = url.PathUnescape(service)
	if err != nil {
		return config{}, fmt.Errorf("decode service name: %w", err)
	}
	if service == "" {
		return config{}, errors.New("service name is required")
	}
	if strings.Contains(service, "/") {
		return config{}, errors.New("service name cannot contain a slash")
	}
	if port := parsed.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return config{}, errors.New("port must be between 1 and 65535")
		}
	}
	return config{host: parsed.Hostname(), service: service}, nil
}

// Name returns the configured Oracle service name.
func (o *oracleDatabase) Name() string {
	return o.config.service
}

// Engine returns the Oracle database engine identifier.
func (o *oracleDatabase) Engine() string {
	return db.EngineOracle
}

// Host returns the configured Oracle network host.
func (o *oracleDatabase) Host() string {
	return o.config.host
}

// ListTables returns base tables visible to the current Oracle user.
func (o *oracleDatabase) ListTables(ctx context.Context) ([]db.Table, error) {
	o.logger.Log(listTablesSQL)
	rows, err := o.database.QueryContext(ctx, listTablesSQL)
	if err != nil {
		return nil, fmt.Errorf("query Oracle tables: %w", err)
	}
	defer rows.Close()

	tables := make([]db.Table, 0)
	for rows.Next() {
		var table db.Table
		if err := rows.Scan(&table.Name); err != nil {
			return nil, fmt.Errorf("scan Oracle table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Oracle tables: %w", err)
	}
	return tables, nil
}

// ListViews returns views visible to the current Oracle user.
func (o *oracleDatabase) ListViews(ctx context.Context) ([]db.View, error) {
	o.logger.Log(listViewsSQL)
	rows, err := o.database.QueryContext(ctx, listViewsSQL)
	if err != nil {
		return nil, fmt.Errorf("query Oracle views: %w", err)
	}
	defer rows.Close()

	views := make([]db.View, 0)
	for rows.Next() {
		var view db.View
		if err := rows.Scan(&view.Name); err != nil {
			return nil, fmt.Errorf("scan Oracle view: %w", err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Oracle views: %w", err)
	}
	return views, nil
}

// ListMaterializedViews returns materialized views visible to the current Oracle user.
func (o *oracleDatabase) ListMaterializedViews(ctx context.Context) ([]db.MaterializedView, error) {
	o.logger.Log(listMaterializedViewsSQL)
	rows, err := o.database.QueryContext(ctx, listMaterializedViewsSQL)
	if err != nil {
		return nil, fmt.Errorf("query Oracle materialized views: %w", err)
	}
	defer rows.Close()

	views := make([]db.MaterializedView, 0)
	for rows.Next() {
		var view db.MaterializedView
		if err := rows.Scan(&view.Name); err != nil {
			return nil, fmt.Errorf("scan Oracle materialized view: %w", err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Oracle materialized views: %w", err)
	}
	return views, nil
}

// ListColumns returns the columns defined by an Oracle table visible to the current user.
func (o *oracleDatabase) ListColumns(ctx context.Context, table db.Table) ([]db.Column, error) {
	o.logger.Log(listColumnsSQL)
	rows, err := o.database.QueryContext(ctx, listColumnsSQL, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query Oracle columns: %w", err)
	}
	defer rows.Close()

	columns := make([]db.Column, 0)
	for rows.Next() {
		var (
			column                                     db.Column
			identity, collation, defaultValue, comment sql.NullString
		)
		if err := rows.Scan(
			&column.Name,
			&column.OrdinalPosition,
			&column.DataType,
			&identity,
			&collation,
			&column.NotNull,
			&defaultValue,
			&comment,
		); err != nil {
			return nil, fmt.Errorf("scan Oracle column: %w", err)
		}
		column.Identity = identity.String
		column.Collation = collation.String
		column.Default = strings.TrimSpace(defaultValue.String)
		column.Comment = comment.String
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Oracle columns: %w", err)
	}
	return columns, nil
}

// ListIndexes returns every indexed column of an Oracle table visible to the current user.
func (o *oracleDatabase) ListIndexes(ctx context.Context, table db.Table) ([]db.IndexColumns, error) {
	o.logger.Log(listIndexesSQL)
	rows, err := o.database.QueryContext(ctx, listIndexesSQL, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query Oracle indexes: %w", err)
	}
	defer rows.Close()

	indexes := make([]db.IndexColumns, 0)
	for rows.Next() {
		var index db.IndexColumns
		if err := rows.Scan(&index.Name, &index.Column, &index.Table, &index.AccessMethod); err != nil {
			return nil, fmt.Errorf("scan Oracle index column: %w", err)
		}
		indexes = append(indexes, index)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Oracle index columns: %w", err)
	}
	return indexes, nil
}

// GetRows returns an unordered, bounded page of rows from an Oracle table.
func (o *oracleDatabase) GetRows(ctx context.Context, table db.Table, page db.PageRequest) (db.RowPage, error) {
	if table.Name == "" {
		return db.RowPage{}, errors.New("table name is required")
	}
	if page.Offset < 0 {
		return db.RowPage{}, errors.New("page offset cannot be negative")
	}
	if page.Limit < 1 || page.Limit > db.MaxPageSize {
		return db.RowPage{}, fmt.Errorf("page limit must be between 1 and %d", db.MaxPageSize)
	}

	query := "SELECT * FROM " + quoteIdentifier(table.Name) + " OFFSET :1 ROWS FETCH NEXT :2 ROWS ONLY"
	o.logger.Log(fmt.Sprintf("SELECT * FROM %s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", quoteIdentifier(table.Name), page.Offset, page.Limit+1))
	rows, err := o.database.QueryContext(ctx, query, page.Offset, page.Limit+1)
	if err != nil {
		return db.RowPage{}, fmt.Errorf("query Oracle rows: %w", err)
	}
	result, err := readRowPage(rows, 0)
	if err != nil {
		return db.RowPage{}, err
	}
	if len(result.Rows) > page.Limit {
		result.HasMore = true
		result.Rows = result.Rows[:page.Limit]
	}
	return result, nil
}

// TableDDL returns Oracle's executable table DDL for a table visible to the current user.
func (o *oracleDatabase) TableDDL(ctx context.Context, table db.Table) (string, error) {
	if table.Name == "" {
		return "", errors.New("table name is required")
	}
	const statement = "SELECT DBMS_METADATA.GET_DDL('TABLE', :1) FROM dual"
	o.logger.Log(statement)
	var ddl string
	if err := o.database.QueryRowContext(ctx, statement, table.Name).Scan(&ddl); err != nil {
		return "", fmt.Errorf("lookup Oracle table DDL: %w", err)
	}
	if strings.TrimSpace(ddl) == "" {
		return "", fmt.Errorf("lookup Oracle table DDL: no DDL for %q", table.Name)
	}
	return ensureTrailingSemicolon(ddl), nil
}

// Execute runs arbitrary SQL and returns up to db.MaxQueryResultRows rows.
func (o *oracleDatabase) Execute(ctx context.Context, statement string) (db.QueryResult, error) {
	o.logger.Log(statement)
	rows, err := o.database.QueryContext(ctx, statement)
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("execute Oracle query: %w", err)
	}
	return readQueryResult(rows, db.MaxQueryResultRows, commandTag(statement))
}

// Dump reports that Oracle Data Pump is not supported by db-tui.
func (o *oracleDatabase) Dump(context.Context) error {
	return errors.New("Oracle dumps are not supported; use Data Pump outside db-tui")
}

// Export writes all table rows using the requested format.
func (o *oracleDatabase) Export(ctx context.Context, table db.Table, typeVal string) error {
	if table.Name == "" {
		return errors.New("table name is required")
	}
	query := "SELECT * FROM " + quoteIdentifier(table.Name)
	o.logger.Log(query)
	rows, err := o.database.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query Oracle export rows: %w", err)
	}
	data, err := readRowPage(rows, 0)
	if err != nil {
		return err
	}
	filename := db.TimestampedFilename(db.SafeFilename(table.Name), typeVal)
	switch typeVal {
	case db.ExportTypeCSV:
		if err := csvexport.Write(filename, data.Columns, data.Rows); err != nil {
			return fmt.Errorf("write CSV export: %w", err)
		}
	case db.ExportTypeJSON:
		if err := jsonexport.Write(filename, table.Name, data.Columns, data.Rows); err != nil {
			return fmt.Errorf("write JSON export: %w", err)
		}
	}
	return nil
}

// ExportQuery re-runs a SELECT query and writes all result rows to CSV.
func (o *oracleDatabase) ExportQuery(ctx context.Context, statement string) error {
	if err := db.ValidateSelectQuery(statement); err != nil {
		return err
	}
	tx, err := o.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Oracle export transaction: %w", err)
	}
	defer tx.Rollback()

	o.logger.Log(statement)
	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return fmt.Errorf("query Oracle export rows: %w", err)
	}
	result, err := readQueryResult(rows, 0, "")
	if err != nil {
		return err
	}
	filename := db.TimestampedFilename("query", db.ExportTypeCSV)
	if err := csvexport.Write(filename, result.Columns, result.Rows); err != nil {
		return fmt.Errorf("write CSV query export: %w", err)
	}
	return nil
}

// Close releases the Oracle connection and query logger.
func (o *oracleDatabase) Close() {
	_ = o.database.Close()
	_ = o.logger.Close()
}

func (o *oracleDatabase) UpdateRow(ctx context.Context, table db.Table, setColumns map[string]any, whereColumns map[string]any) error {
	return errors.New("edit row not yet implemented for MySQL")
}

func readRowPage(rows *sql.Rows, rowLimit int) (db.RowPage, error) {
	result, err := readRows(rows, rowLimit)
	if err != nil {
		return db.RowPage{}, err
	}
	return db.RowPage{Columns: result.Columns, Rows: result.Rows}, nil
}

func readQueryResult(rows *sql.Rows, rowLimit int, tag string) (db.QueryResult, error) {
	result, err := readRows(rows, rowLimit)
	if err != nil {
		return db.QueryResult{}, err
	}
	return db.QueryResult{Columns: result.Columns, Rows: result.Rows, CommandTag: tag}, nil
}

func readRows(rows *sql.Rows, rowLimit int) (struct {
	Columns []string
	Rows    [][]any
}, error) {
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return struct {
			Columns []string
			Rows    [][]any
		}{}, fmt.Errorf("read Oracle columns: %w", err)
	}
	result := struct {
		Columns []string
		Rows    [][]any
	}{Columns: columns}
	for rows.Next() {
		if rowLimit > 0 && len(result.Rows) == rowLimit {
			break
		}
		values := make([]any, len(columns))
		destinations := make([]any, len(values))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			return result, fmt.Errorf("read Oracle row: %w", err)
		}
		for index, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[index] = string(bytes)
			}
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate Oracle rows: %w", err)
	}
	return result, nil
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func ensureTrailingSemicolon(statement string) string {
	statement = strings.TrimRight(statement, " \t\r\n")
	if strings.HasSuffix(statement, ";") {
		return statement
	}
	return statement + ";"
}

func commandTag(statement string) string {
	fields := strings.Fields(statement)
	if len(fields) == 0 {
		return ""
	}
	command := strings.ToUpper(strings.Trim(fields[0], "();"))
	if (command == "CREATE" || command == "ALTER" || command == "DROP") && len(fields) > 1 {
		return command + " " + strings.ToUpper(strings.Trim(fields[1], "();\""))
	}
	return command
}
