// Package postgres provides a PostgreSQL implementation of db.Database.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/querylog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const listTablesSQL = `
	SELECT table_name
	FROM information_schema.tables
	WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
	ORDER BY table_name`

const queryLogPath = "queries.log"

type postgresql struct {
	pool   *pgxpool.Pool
	logger *querylog.Logger
}

// Connect opens a PostgreSQL database using dsn and verifies that it is reachable.
func Connect(ctx context.Context, dsn string) (db.Database, error) {
	logger, err := querylog.Open(queryLogPath)
	if err != nil {
		return nil, fmt.Errorf("open query log: %w", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		_ = logger.Close()
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = logger.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return &postgresql{pool: pool, logger: logger}, nil
}

// ListTables returns the base tables in the connected database's public schema.
func (p *postgresql) ListTables(ctx context.Context) ([]db.Table, error) {
	p.logger.Log(listTablesSQL)
	rows, err := p.pool.Query(ctx, listTablesSQL)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL tables: %w", err)
	}
	defer rows.Close()

	tables := make([]db.Table, 0)
	for rows.Next() {
		var table db.Table
		if err := rows.Scan(&table.Name); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL tables: %w", err)
	}

	return tables, nil
}

// GetRows returns an unordered page of rows from a public PostgreSQL table.
func (p *postgresql) GetRows(ctx context.Context, table db.Table, page db.PageRequest) (db.RowPage, error) {
	if table.Name == "" {
		return db.RowPage{}, errors.New("table name is required")
	}
	if page.Offset < 0 {
		return db.RowPage{}, errors.New("page offset cannot be negative")
	}
	if page.Limit < 1 || page.Limit > db.MaxPageSize {
		return db.RowPage{}, fmt.Errorf("page limit must be between 1 and %d", db.MaxPageSize)
	}

	tableName := pgx.Identifier{"public", table.Name}.Sanitize()
	query := fmt.Sprintf("SELECT * FROM %s LIMIT $1 OFFSET $2", tableName)
	queryLimit := page.Limit + 1
	p.logger.Log(fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, queryLimit, page.Offset))
	rows, err := p.pool.Query(ctx, query, queryLimit, page.Offset)
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

	if len(result.Rows) > page.Limit {
		result.HasMore = true
		result.Rows = result.Rows[:page.Limit]
	}

	return result, nil
}

// Close releases all connections held by the database.
func (p *postgresql) Close() {
	p.pool.Close()
	_ = p.logger.Close()
}
