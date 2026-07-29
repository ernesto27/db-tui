// Package postgres provides a PostgreSQL implementation of db.Database.
package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
	ORDER BY table_name`

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
		if page.Limit < 1 || page.Limit > db.MaxPageSize {
			return db.RowPage{}, fmt.Errorf("page limit must be between 1 and %d", db.MaxPageSize)
		}
	}

	tableName := pgx.Identifier{"public", table.Name}.Sanitize()
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

	return readQueryResult(rows, db.MaxQueryResultRows)
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
