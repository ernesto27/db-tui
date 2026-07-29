// Package db defines database-engine-neutral types and session behavior.
package db

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// MaxPageSize is the largest number of rows a single page can contain.
const MaxPageSize = 100

// MaxQueryResultRows is the largest number of rows returned for a raw query.
const MaxQueryResultRows = 100

const filenameTimestampLayout = "20060102_150405"

// TimestampedFilename returns a filename with prefix, the current timestamp, and extension.
//
// The extension must not include a leading period.
func TimestampedFilename(prefix, extension string) string {
	return prefix + "_" + time.Now().Format(filenameTimestampLayout) + "." + extension
}

// SafeFilename returns a safe single filename component derived from name.
func SafeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(character rune) rune {
		switch {
		case character >= 'a' && character <= 'z':
			return character
		case character >= 'A' && character <= 'Z':
			return character
		case character >= '0' && character <= '9':
			return character
		case character == '-', character == '_', character == '.':
			return character
		default:
			return '_'
		}
	}, name)
	if strings.Trim(name, ".") == "" {
		return "export"
	}
	return name
}

// ValidateSelectQuery reports whether query contains SELECT.
func ValidateSelectQuery(query string) error {
	if !strings.Contains(strings.ToUpper(query), "SELECT") {
		return errors.New("only SELECT queries can be exported")
	}
	return nil
}

// Supported database engines.
const (
	EnginePostgreSQL = "postgres"
	EngineMySQL      = "mysql"
	EngineSQLite     = "sqlite"

	// ExportTypeCSV identifies CSV table exports.
	ExportTypeCSV  = "csv"
	ExportTypeJSON = "json"
)

// Table identifies a table available in a database session.
type Table struct {
	Name string
}

// PageRequest identifies a bounded range of rows to return.
//
// Offset must not be negative. Limit must be between 1 and MaxPageSize.
type PageRequest struct {
	Offset int
	Limit  int
}

// RowPage contains one page of rows from a table.
//
// Columns and each row's values have matching order. SQL NULL values are nil.
type RowPage struct {
	Columns []string
	Rows    [][]any
	HasMore bool
}

// QueryResult contains rows returned by a SQL statement and its command status.
// Columns is empty when the statement does not return rows.
type QueryResult struct {
	Columns    []string
	Rows       [][]any
	CommandTag string
}

// Database provides operations supported by a connected database.
type Database interface {
	// Name returns the connected database name for display.
	Name() string
	// Engine returns the connected database engine identifier.
	Engine() string
	// Host returns the configured network host, or an empty string for local databases.
	Host() string
	ListTables(ctx context.Context) ([]Table, error)
	// GetRows returns an unordered page of rows from table.
	GetRows(ctx context.Context, table Table, page PageRequest) (RowPage, error)
	// TableDDL returns a fresh executable structural DDL script for table.
	TableDDL(ctx context.Context, table Table) (string, error)
	// Execute runs SQL and returns its first rows and command status.
	Execute(ctx context.Context, sql string) (QueryResult, error)
	Dump(ctx context.Context) error
	// Export writes all table rows using typeVal as the export format.
	Export(ctx context.Context, table Table, typeVal string) error
	ExportQuery(ctx context.Context, sql string) error
	Close()
}
