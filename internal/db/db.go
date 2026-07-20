// Package db defines database-engine-neutral types and session behavior.
package db

import "context"

// MaxPageSize is the largest number of rows a single page can contain.
const MaxPageSize = 100

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

// Database provides operations supported by a connected database.
type Database interface {
	// Name returns the connected database name for display.
	Name() string
	ListTables(ctx context.Context) ([]Table, error)
	// GetRows returns an unordered page of rows from table.
	GetRows(ctx context.Context, table Table, page PageRequest) (RowPage, error)
	Close()
}
