// Package sqlite provides a SQLite implementation of db.Database.
package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/csvexport"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/jsonexport"
	"github.com/ernestoponce27/db-tui/internal/logger"
	_ "modernc.org/sqlite"
)

const listTablesSQL = `
	SELECT name
	FROM sqlite_master
	WHERE type = 'table'
		AND name NOT LIKE 'sqlite_%'
	ORDER BY name`

const listColumnSQL = `SELECT
    column_info.name AS column_name,
    column_info.cid + 1 AS ordinal_position,
    column_info.type AS data_type,
    NULL AS identity,
    NULL AS collation_name,
    CASE
        WHEN column_info."notnull" <> 0 THEN 1
        ELSE 0
    END AS not_null,
    column_info.dflt_value AS default_expression,
    NULL AS comment
FROM pragma_table_xinfo(?) AS column_info
WHERE column_info."hidden" <> 1
ORDER BY column_info.cid`

type sqliteDatabase struct {
	database *sql.DB
	logger   *logger.Logger
	path     string
}

// Connect opens a local SQLite database file and verifies it is reachable.
func Connect(ctx context.Context, path string) (db.Database, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("SQLite database file is required")
	}
	if strings.EqualFold(path, ":memory:") || strings.HasPrefix(strings.ToLower(path), "file:") {
		return nil, errors.New("SQLite connections require a local database file")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("open SQLite database file: %s is a directory", path)
	}

	queryLogger, err := logger.Open()
	if err != nil {
		return nil, fmt.Errorf("open query log: %w", err)
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		_ = queryLogger.Close()
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		_ = queryLogger.Close()
		return nil, fmt.Errorf("connect to SQLite: %w", err)
	}

	return &sqliteDatabase{database: database, logger: queryLogger, path: path}, nil
}

// Name returns the database file name for display.
func (s *sqliteDatabase) Name() string {
	return filepath.Base(s.path)
}

// Engine returns the SQLite database engine identifier.
func (s *sqliteDatabase) Engine() string {
	return db.EngineSQLite
}

// Host returns an empty string because SQLite databases are local files.
func (s *sqliteDatabase) Host() string {
	return ""
}

// ListTables returns non-internal SQLite tables in alphabetical order.
func (s *sqliteDatabase) ListTables(ctx context.Context) ([]db.Table, error) {
	s.logger.Log(listTablesSQL)
	rows, err := s.database.QueryContext(ctx, listTablesSQL)
	if err != nil {
		return nil, fmt.Errorf("query SQLite tables: %w", err)
	}
	defer rows.Close()

	tables := make([]db.Table, 0)
	for rows.Next() {
		var table db.Table
		if err := rows.Scan(&table.Name); err != nil {
			return nil, fmt.Errorf("scan SQLite table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SQLite tables: %w", err)
	}
	return tables, nil
}

// ListColumns returns the columns defined by a SQLite table.
func (s *sqliteDatabase) ListColumns(ctx context.Context, table db.Table) ([]db.Column, error) {
	s.logger.Log(listColumnSQL)
	rows, err := s.database.QueryContext(ctx, listColumnSQL, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query SQLite columns: %w", err)
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
			return nil, fmt.Errorf("scan SQLite column: %w", err)
		}

		column.Identity = identity.String
		column.Collation = collation.String
		column.Default = defaultValue.String
		column.Comment = comment.String
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SQLite columns: %w", err)
	}

	return columns, nil
}

// GetRows returns an unordered, bounded page of rows from a table.
func (s *sqliteDatabase) GetRows(ctx context.Context, table db.Table, page db.PageRequest) (db.RowPage, error) {
	return s.getRows(ctx, table, &page)
}

func (s *sqliteDatabase) getRows(ctx context.Context, table db.Table, page *db.PageRequest) (db.RowPage, error) {
	if table.Name == "" {
		return db.RowPage{}, errors.New("table name is required")
	}
	if page != nil {
		if page.Offset < 0 {
			return db.RowPage{}, errors.New("page offset cannot be negative")
		}
		if page.Limit < 1 || page.Limit > db.MaxPageSize {
			return db.RowPage{}, fmt.Errorf("page limit must be between 1 and %d", db.MaxPageSize)
		}
	}

	query := "SELECT * FROM " + quoteIdentifier(table.Name)
	args := make([]any, 0, 2)
	if page != nil {
		query += " LIMIT ? OFFSET ?"
		args = append(args, page.Limit+1, page.Offset)
	}
	s.logger.Log(query)
	rows, err := s.database.QueryContext(ctx, query, args...)
	if err != nil {
		return db.RowPage{}, fmt.Errorf("query SQLite rows: %w", err)
	}

	result, err := readRowPage(rows, 0)
	if err != nil {
		return db.RowPage{}, err
	}
	if page != nil && len(result.Rows) > page.Limit {
		result.HasMore = true
		result.Rows = result.Rows[:page.Limit]
	}
	return result, nil
}

// TableDDL returns SQLite's executable CREATE TABLE statement for table.
func (s *sqliteDatabase) TableDDL(ctx context.Context, table db.Table) (string, error) {
	if table.Name == "" {
		return "", errors.New("table name is required")
	}
	const statement = "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?"
	s.logger.Log(statement)
	var ddl sql.NullString
	if err := s.database.QueryRowContext(ctx, statement, table.Name).Scan(&ddl); err != nil {
		return "", fmt.Errorf("lookup SQLite table DDL: %w", err)
	}
	if !ddl.Valid || strings.TrimSpace(ddl.String) == "" {
		return "", fmt.Errorf("lookup SQLite table DDL: no DDL for %q", table.Name)
	}
	return ensureTrailingSemicolon(ddl.String), nil
}

// Execute runs arbitrary SQL and returns up to db.MaxQueryResultRows rows.
func (s *sqliteDatabase) Execute(ctx context.Context, statement string) (db.QueryResult, error) {
	s.logger.Log(statement)
	rows, err := s.database.QueryContext(ctx, statement)
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("execute SQLite query: %w", err)
	}
	return readQueryResult(rows, db.MaxQueryResultRows, commandTag(statement))
}

// Dump writes an SQLite .dump to a timestamped SQL file using sqlite3.
func (s *sqliteDatabase) Dump(ctx context.Context) error {
	filename := db.TimestampedFilename(db.SafeFilename(s.Name()), "sql")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create SQLite dump file: %w", err)
	}

	var stderr bytes.Buffer
	command := exec.CommandContext(ctx, "sqlite3", s.path, ".dump")
	command.Stdout = file
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		_ = file.Close()
		_ = os.Remove(filename)
		return fmt.Errorf("sqlite3 dump: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(filename)
		return fmt.Errorf("close SQLite dump file: %w", err)
	}
	return nil
}

// Export writes all table rows using the requested format.
func (s *sqliteDatabase) Export(ctx context.Context, table db.Table, typeVal string) error {
	data, err := s.getRows(ctx, table, nil)
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
func (s *sqliteDatabase) ExportQuery(ctx context.Context, statement string) error {
	if err := db.ValidateSelectQuery(statement); err != nil {
		return err
	}
	tx, err := s.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin SQLite export transaction: %w", err)
	}
	defer tx.Rollback()

	s.logger.Log(statement)
	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return fmt.Errorf("query SQLite export rows: %w", err)
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

// Close releases the database connection and query logger.
func (s *sqliteDatabase) Close() {
	_ = s.database.Close()
	_ = s.logger.Close()
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

type tabularResult struct {
	Columns []string
	Rows    [][]any
}

func readRows(rows *sql.Rows, rowLimit int) (tabularResult, error) {
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return tabularResult{}, fmt.Errorf("read SQLite columns: %w", err)
	}
	result := tabularResult{Columns: columns}
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
			return tabularResult{}, fmt.Errorf("read SQLite row: %w", err)
		}
		for index, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[index] = append([]byte(nil), bytes...)
			}
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return tabularResult{}, fmt.Errorf("iterate SQLite rows: %w", err)
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

var _ db.Database = (*sqliteDatabase)(nil)
