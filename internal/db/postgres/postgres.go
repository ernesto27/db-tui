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
	"time"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/logger"
	"github.com/jackc/pgx/v5"
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

// Execute runs arbitrary SQL and returns its first 100 rows and command status.
func (p *postgresql) Execute(ctx context.Context, sql string) (db.QueryResult, error) {
	p.logger.Log(sql)
	rows, err := p.pool.Query(ctx, sql, pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("execute PostgreSQL query: %w", err)
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	result := db.QueryResult{Columns: make([]string, len(fields))}
	for index, field := range fields {
		result.Columns[index] = field.Name
	}
	for rows.Next() {
		if len(result.Rows) == db.MaxQueryResultRows {
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

	timestamp := time.Now().Format("20060102_150405")
	filename := connConfig.Database + "_" + timestamp + ".sql"
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
