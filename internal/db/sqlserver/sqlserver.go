// Package sqlserver provides the Microsoft SQL Server database adapter.
package sqlserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/db"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
)

var errNotImplemented = errors.New("SQL Server support is not implemented")

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

		SELECT schema_info.name AS schema_name, 'views' AS object_type
		FROM sys.views AS view_info
		JOIN sys.schemas AS schema_info
			ON schema_info.schema_id = view_info.schema_id
		WHERE view_info.is_ms_shipped = 0
			AND NOT EXISTS (
				SELECT 1
				FROM sys.indexes AS index_info
				WHERE index_info.object_id = view_info.object_id
					AND index_info.index_id = 1
			)

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
	return db.EngineSqlServer
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
		return "", fmt.Errorf("not found")
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

func (s *sqlserverDatabase) Export(context.Context, db.Table, string) error {
	return errNotImplemented
}

func (s *sqlserverDatabase) ExportQuery(context.Context, string) error {
	return errNotImplemented
}

func (s *sqlserverDatabase) ListColumns(context.Context, db.Table) ([]db.Column, error) {
	return nil, errNotImplemented
}

func (s *sqlserverDatabase) ListIndexes(context.Context, db.Table) ([]db.IndexColumns, error) {
	return nil, errNotImplemented
}

func (s *sqlserverDatabase) ListViews(context.Context, string) ([]db.View, error) {
	return nil, errNotImplemented
}

func (s *sqlserverDatabase) ListMaterializedViews(context.Context, string) ([]db.MaterializedView, error) {
	return nil, errNotImplemented
}

func (s *sqlserverDatabase) UpdateRow(context.Context, db.Table, map[string]any, map[string]any) error {
	return errNotImplemented
}

func (s *sqlserverDatabase) DeleteRow(context.Context, db.Table, map[string]any) error {
	return errNotImplemented
}

func (s *sqlserverDatabase) ListFunctions(context.Context, string) ([]db.FunctionColumns, error) {
	return nil, errNotImplemented
}

func (s *sqlserverDatabase) Close() {}
