// Package mysql provides a MySQL implementation of db.Database.
package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/csvexport"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/jsonexport"
	"github.com/ernestoponce27/db-tui/internal/logger"
	mysqldriver "github.com/go-sql-driver/mysql"
)

const listTablesSQL = `
	SELECT table_name
	FROM information_schema.tables
	WHERE table_schema = DATABASE()
		AND table_type = 'BASE TABLE'
	ORDER BY table_name`

const listViewsSQL = `
	SELECT table_name
	FROM information_schema.views
	WHERE table_schema = DATABASE()
	ORDER BY table_name`

const listColumnsSQL = `
	SELECT
		column_info.COLUMN_NAME AS column_name,
		column_info.ORDINAL_POSITION AS ordinal_position,
		column_info.COLUMN_TYPE AS data_type,
		CASE
			WHEN column_info.EXTRA LIKE '%auto_increment%' THEN 'AUTO_INCREMENT'
			ELSE NULL
		END AS identity,
		CASE
			WHEN column_info.COLLATION_NAME IS NULL THEN NULL
			WHEN column_info.COLLATION_NAME = table_info.TABLE_COLLATION THEN 'default'
			ELSE column_info.COLLATION_NAME
		END AS collation_name,
		CASE column_info.IS_NULLABLE
			WHEN 'NO' THEN TRUE
			ELSE FALSE
		END AS not_null,
		column_info.COLUMN_DEFAULT AS default_expression,
		column_info.COLUMN_COMMENT AS comment,
		EXISTS (
			SELECT 1
			FROM information_schema.key_column_usage AS key_info
			WHERE key_info.CONSTRAINT_SCHEMA = column_info.TABLE_SCHEMA
				AND key_info.TABLE_NAME = column_info.TABLE_NAME
				AND key_info.COLUMN_NAME = column_info.COLUMN_NAME
				AND key_info.CONSTRAINT_NAME = 'PRIMARY'
		) AS is_primary_key
	FROM information_schema.columns AS column_info
	JOIN information_schema.tables AS table_info
		ON table_info.TABLE_SCHEMA = column_info.TABLE_SCHEMA
		AND table_info.TABLE_NAME = column_info.TABLE_NAME
	WHERE column_info.TABLE_SCHEMA = DATABASE()
		AND column_info.TABLE_NAME = ?
	ORDER BY column_info.ORDINAL_POSITION`

const listIndexesSQL = `
	SELECT
		index_info.INDEX_NAME AS index_name,
		COALESCE(index_info.COLUMN_NAME, '') AS column_name,
		index_info.TABLE_NAME AS table_name,
		index_info.INDEX_TYPE AS access_method
	FROM information_schema.statistics AS index_info
	WHERE index_info.TABLE_SCHEMA = DATABASE()
		AND index_info.TABLE_NAME = ?
	ORDER BY index_info.INDEX_NAME, index_info.SEQ_IN_INDEX`

const listPrimaryKeyColumnsSQL = `
	SELECT column_name
	FROM information_schema.key_column_usage
	WHERE constraint_schema = DATABASE()
		AND table_name = ?
		AND constraint_name = 'PRIMARY'
	ORDER BY ordinal_position`

const listFunctionsSQL = `
	SELECT
		r.ROUTINE_NAME AS function_name,
		COALESCE(
			GROUP_CONCAT(
				CONCAT(p.PARAMETER_MODE, ' ', p.PARAMETER_NAME, ' ', p.DTD_IDENTIFIER)
				ORDER BY p.ORDINAL_POSITION
				SEPARATOR ', '
			),
			''
		) AS arguments,
		r.DTD_IDENTIFIER AS returns,
		r.ROUTINE_BODY AS language,
		COALESCE(r.ROUTINE_DEFINITION, '') AS definition
	FROM INFORMATION_SCHEMA.ROUTINES AS r
	LEFT JOIN INFORMATION_SCHEMA.PARAMETERS AS p
		ON p.SPECIFIC_SCHEMA = r.ROUTINE_SCHEMA
		AND p.SPECIFIC_NAME = r.SPECIFIC_NAME
		AND p.ORDINAL_POSITION > 0
	WHERE r.ROUTINE_TYPE = 'FUNCTION'
		AND r.ROUTINE_SCHEMA = ?
	GROUP BY
		r.ROUTINE_SCHEMA,
		r.ROUTINE_NAME,
		r.SPECIFIC_NAME,
		r.DTD_IDENTIFIER,
		r.ROUTINE_BODY,
		r.ROUTINE_DEFINITION
	ORDER BY r.ROUTINE_NAME;`

type mysqlDatabase struct {
	database *sql.DB
	logger   *logger.Logger
	config   *mysqldriver.Config
}

// ListColumns returns the columns defined by a table in the active MySQL database.
func (m *mysqlDatabase) ListColumns(ctx context.Context, table db.Table) ([]db.Column, error) {
	m.logger.Log(listColumnsSQL)
	rows, err := m.database.QueryContext(ctx, listColumnsSQL, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query MySQL columns: %w", err)
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
			&column.IsPrimaryKey,
		); err != nil {
			return nil, fmt.Errorf("scan MySQL column: %w", err)
		}

		column.Identity = identity.String
		column.Collation = collation.String
		column.Default = defaultValue.String
		column.Comment = comment.String
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL columns: %w", err)
	}

	return columns, nil
}

// ListIndexes returns every indexed column of a table in the active MySQL database.
func (m *mysqlDatabase) ListIndexes(ctx context.Context, table db.Table) ([]db.IndexColumns, error) {
	m.logger.Log(listIndexesSQL)
	rows, err := m.database.QueryContext(ctx, listIndexesSQL, table.Name)
	if err != nil {
		return nil, fmt.Errorf("query MySQL indexes: %w", err)
	}
	defer rows.Close()

	indexes := make([]db.IndexColumns, 0)
	for rows.Next() {
		var index db.IndexColumns
		if err := rows.Scan(&index.Name, &index.Column, &index.Table, &index.AccessMethod); err != nil {
			return nil, fmt.Errorf("scan MySQL index: %w", err)
		}
		indexes = append(indexes, index)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL indexes: %w", err)
	}

	return indexes, nil
}

// Connect opens a MySQL database using dsn and verifies that it is reachable.
//
// Both go-sql-driver/mysql DSNs and mysql:// URLs are accepted. URL-form DSNs
// are useful for connection settings generated by the TUI.
func Connect(ctx context.Context, dsn string) (db.Database, error) {
	driverConfig, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse MySQL DSN: %w", err)
	}

	queryLogger, err := logger.Open()
	if err != nil {
		return nil, fmt.Errorf("open query log: %w", err)
	}

	connector, err := mysqldriver.NewConnector(driverConfig)
	if err != nil {
		_ = queryLogger.Close()
		return nil, fmt.Errorf("create MySQL connector: %w", err)
	}
	database := sql.OpenDB(connector)
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		_ = queryLogger.Close()
		return nil, fmt.Errorf("connect to MySQL: %w", err)
	}

	return &mysqlDatabase{database: database, logger: queryLogger, config: driverConfig}, nil
}

func parseDSN(dsn string) (*mysqldriver.Config, error) {
	dsn = strings.TrimSpace(dsn)
	if !strings.HasPrefix(strings.ToLower(dsn), "mysql://") {
		config, err := mysqldriver.ParseDSN(dsn)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(config.DBName) == "" {
			return nil, errors.New("database name is required")
		}
		return config, nil
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "mysql" {
		return nil, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	if parsed.User == nil || parsed.User.Username() == "" {
		return nil, errors.New("username is required")
	}
	if parsed.Hostname() == "" {
		return nil, errors.New("host is required")
	}

	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if databaseName == "" {
		return nil, errors.New("database name is required")
	}
	if strings.Contains(databaseName, "/") {
		return nil, errors.New("database name cannot contain a slash")
	}

	port := parsed.Port()
	if port == "" {
		port = "3306"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, errors.New("port must be between 1 and 65535")
	}

	password, _ := parsed.User.Password()
	config := mysqldriver.NewConfig()
	config.User = parsed.User.Username()
	config.Passwd = password
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(parsed.Hostname(), port)
	config.DBName = databaseName

	if parsed.RawQuery != "" {
		return mysqldriver.ParseDSN(config.FormatDSN() + "?" + parsed.RawQuery)
	}
	return config, nil
}

// Name returns the configured MySQL database name.
func (m *mysqlDatabase) Name() string {
	return m.config.DBName
}

// Engine returns the MySQL database engine identifier.
func (m *mysqlDatabase) Engine() string {
	return db.EngineMySQL
}

// Host returns the configured MySQL network host.
func (m *mysqlDatabase) Host() string {
	host, _, err := net.SplitHostPort(m.config.Addr)
	if err != nil {
		return ""
	}
	return host
}

// ListTables returns the base tables in the connected MySQL database.
func (m *mysqlDatabase) ListTables(ctx context.Context, _ string) ([]db.Table, error) {
	m.logger.Log(listTablesSQL)
	rows, err := m.database.QueryContext(ctx, listTablesSQL)
	if err != nil {
		return nil, fmt.Errorf("query MySQL tables: %w", err)
	}
	defer rows.Close()

	tables := make([]db.Table, 0)
	for rows.Next() {
		var table db.Table
		if err := rows.Scan(&table.Name); err != nil {
			return nil, fmt.Errorf("scan MySQL table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL tables: %w", err)
	}
	return tables, nil
}

// ListSchemaObjectGroups is a placeholder because schema browsing is PostgreSQL-only.
func (m *mysqlDatabase) ListSchemaObjectGroups(context.Context) ([]db.SchemaObjectGroup, error) {
	return []db.SchemaObjectGroup{}, nil
}

// ListViews returns views in the connected MySQL database in alphabetical order.
func (m *mysqlDatabase) ListViews(ctx context.Context, _ string) ([]db.View, error) {
	m.logger.Log(listViewsSQL)
	rows, err := m.database.QueryContext(ctx, listViewsSQL)
	if err != nil {
		return nil, fmt.Errorf("query MySQL views: %w", err)
	}
	defer rows.Close()

	views := make([]db.View, 0)
	for rows.Next() {
		var view db.View
		if err := rows.Scan(&view.Name); err != nil {
			return nil, fmt.Errorf("scan MySQL view: %w", err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL views: %w", err)
	}

	return views, nil
}

// ListMaterializedViews is a placeholder for MySQL materialized-view discovery.
func (m *mysqlDatabase) ListMaterializedViews(context.Context, string) ([]db.MaterializedView, error) {
	return []db.MaterializedView{}, nil
}

func (m *mysqlDatabase) ListFunctions(ctx context.Context, schema string) ([]db.FunctionColumns, error) {
	rows, err := m.database.QueryContext(ctx, listFunctionsSQL, schema)
	if err != nil {
		return nil, fmt.Errorf("query MySQL functions: %w", err)
	}
	defer rows.Close()

	functionColumns := make([]db.FunctionColumns, 0)
	for rows.Next() {
		var function db.FunctionColumns
		if err := rows.Scan(&function.Name, &function.Arguments, &function.ReturnType, &function.Language, &function.Definition); err != nil {
			return nil, fmt.Errorf("scan MySQL function: %w", err)
		}
		functionColumns = append(functionColumns, function)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate MySQL functions: %w", err)
	}

	return functionColumns, nil
}

// GetRows returns an unordered page of rows from a MySQL table.
func (m *mysqlDatabase) GetRows(ctx context.Context, table db.Table, page db.PageRequest) (db.RowPage, error) {
	return m.getRows(ctx, table, &page)
}

// TableDDL returns MySQL's executable CREATE TABLE statement for table.
func (m *mysqlDatabase) TableDDL(ctx context.Context, table db.Table) (string, error) {
	if table.Name == "" {
		return "", errors.New("table name is required")
	}

	query := "SHOW CREATE TABLE " + quoteIdentifier(table.Name)
	m.logger.Log(query)

	var returnedName, statement string
	if err := m.database.QueryRowContext(ctx, query).Scan(&returnedName, &statement); err != nil {
		return "", fmt.Errorf("show MySQL table DDL: %w", err)
	}
	return ensureTrailingSemicolon(statement), nil
}

func (m *mysqlDatabase) getRows(ctx context.Context, table db.Table, page *db.PageRequest) (db.RowPage, error) {
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
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	args := make([]any, 0, 2)
	if page != nil {
		query += " LIMIT ? OFFSET ?"
		queryLimit := page.Limit + 1
		args = append(args, queryLimit, page.Offset)
		m.logger.Log(fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, queryLimit, page.Offset))
	} else {
		m.logger.Log(query)
	}
	rows, err := m.database.QueryContext(ctx, query, args...)
	if err != nil {
		return db.RowPage{}, fmt.Errorf("query MySQL rows: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return db.RowPage{}, fmt.Errorf("read MySQL columns: %w", err)
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return db.RowPage{}, fmt.Errorf("read MySQL column types: %w", err)
	}
	result := db.RowPage{Columns: columns}
	for rows.Next() {
		values, err := scanRow(rows, columnTypes)
		if err != nil {
			return db.RowPage{}, fmt.Errorf("read MySQL row: %w", err)
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return db.RowPage{}, fmt.Errorf("iterate MySQL rows: %w", err)
	}

	if page != nil && len(result.Rows) > page.Limit {
		result.HasMore = true
		result.Rows = result.Rows[:page.Limit]
	}
	return result, nil
}

// Execute runs arbitrary SQL and returns its first 100 rows and command status.
func (m *mysqlDatabase) Execute(ctx context.Context, statement string) (db.QueryResult, error) {
	m.logger.Log(statement)
	rows, err := m.database.QueryContext(ctx, statement)
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("execute MySQL query: %w", err)
	}

	return readQueryResult(rows, db.MaxPageSize, commandTag(statement))
}

func readQueryResult(rows *sql.Rows, rowLimit int, commandTag string) (db.QueryResult, error) {
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("read MySQL query columns: %w", err)
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("read MySQL query column types: %w", err)
	}
	result := db.QueryResult{Columns: columns, CommandTag: commandTag}
	for rows.Next() {
		if rowLimit > 0 && len(result.Rows) == rowLimit {
			break
		}
		values, err := scanRow(rows, columnTypes)
		if err != nil {
			return db.QueryResult{}, fmt.Errorf("read MySQL query row: %w", err)
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Close(); err != nil {
		return db.QueryResult{}, fmt.Errorf("close MySQL query rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return db.QueryResult{}, fmt.Errorf("iterate MySQL query rows: %w", err)
	}
	return result, nil
}

func scanRow(rows *sql.Rows, columnTypes []*sql.ColumnType) ([]any, error) {
	values := make([]any, len(columnTypes))
	destinations := make([]any, len(values))
	for index := range values {
		destinations[index] = &values[index]
	}
	if err := rows.Scan(destinations...); err != nil {
		return nil, err
	}
	for index, value := range values {
		bytes, ok := value.([]byte)
		if !ok {
			continue
		}
		dataType := strings.ToUpper(columnTypes[index].DatabaseTypeName())
		if strings.Contains(dataType, "BLOB") ||
			strings.Contains(dataType, "BINARY") ||
			dataType == "BIT" ||
			dataType == "GEOMETRY" {
			values[index] = append([]byte(nil), bytes...)
		} else {
			values[index] = string(bytes)
		}
	}
	return values, nil
}

func quoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
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
		index := 1
		if command == "CREATE" && strings.EqualFold(strings.Trim(fields[index], "();"), "TEMPORARY") && len(fields) > 2 {
			index++
		}
		return command + " " + strings.ToUpper(strings.Trim(fields[index], "();`"))
	}
	return command
}

// Dump writes the connected database to a timestamped SQL file using mysqldump.
func (m *mysqlDatabase) Dump(ctx context.Context) error {
	filename := db.TimestampedFilename(db.SafeFilename(m.config.DBName), "sql")
	args := []string{"--user=" + m.config.User, "--result-file=" + filename, "--single-transaction"}
	var port string

	switch m.config.Net {
	case "", "tcp", "tcp4", "tcp6":
		host, parsedPort, err := net.SplitHostPort(m.config.Addr)
		if err != nil {
			return fmt.Errorf("parse MySQL address: %w", err)
		}
		port = parsedPort
		args = append(args, "--protocol=tcp", "--host="+host, "--port="+port)
	case "unix":
		args = append(args, "--socket="+m.config.Addr)
	default:
		return fmt.Errorf("unsupported MySQL network %q for dump", m.config.Net)
	}
	args = append(args, m.config.DBName)

	command := exec.CommandContext(ctx, "mysqldump", args...)
	if m.config.Passwd != "" {
		command.Env = append(os.Environ(), "MYSQL_PWD="+m.config.Passwd)
	}
	output, err := command.CombinedOutput()
	if err != nil {
		_ = os.Remove(filename)
		if errors.Is(err, exec.ErrNotFound) && port != "" {
			return m.dumpFromDocker(ctx, filename, port)
		}
		return fmt.Errorf("mysqldump: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *mysqlDatabase) dumpFromDocker(ctx context.Context, filename, port string) error {
	containerID, err := dockerContainerIDForPort(ctx, port)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create dump file: %w", err)
	}

	var stderr bytes.Buffer
	args := []string{"exec"}
	if m.config.Passwd != "" {
		args = append(args, "-e", "MYSQL_PWD="+m.config.Passwd)
	}
	args = append(args,
		containerID,
		"mysqldump",
		"--user="+m.config.User,
		"--single-transaction",
		m.config.DBName,
	)
	command := exec.CommandContext(ctx, "docker", args...)
	command.Stdout = file
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		_ = file.Close()
		_ = os.Remove(filename)
		return fmt.Errorf("docker mysqldump: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(filename)
		return fmt.Errorf("close dump file: %w", err)
	}

	return nil
}

func (m *mysqlDatabase) Export(ctx context.Context, table db.Table, typeVal string) error {
	data, err := m.getRows(ctx, table, nil)
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

// ExportQuery re-runs a SELECT query in a read-only transaction and writes all rows to CSV.
func (m *mysqlDatabase) ExportQuery(ctx context.Context, statement string) error {
	if err := db.ValidateSelectQuery(statement); err != nil {
		return err
	}

	tx, err := m.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin MySQL export transaction: %w", err)
	}
	defer tx.Rollback()

	m.logger.Log(statement)
	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return fmt.Errorf("query MySQL export rows: %w", err)
	}

	result, err := readQueryResult(rows, 0, "")
	if err != nil {
		return err
	}

	filename := db.TimestampedFilename("query", "csv")
	if err := csvexport.Write(filename, result.Columns, result.Rows); err != nil {
		return fmt.Errorf("write CSV query export: %w", err)
	}
	return nil
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

// Close releases all connections held by the database.
func (m *mysqlDatabase) Close() {
	_ = m.database.Close()
	_ = m.logger.Close()
}

func (m *mysqlDatabase) UpdateRow(ctx context.Context, table db.Table, setColumns map[string]any, whereColumns map[string]any) error {
	m.logger.Log(listPrimaryKeyColumnsSQL)
	rows, err := m.database.QueryContext(ctx, listPrimaryKeyColumnsSQL, table.Name)
	if err != nil {
		return fmt.Errorf("query MySQL primary key columns: %w", err)
	}
	defer rows.Close()

	primaryKeys := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan MySQL primary key column: %w", err)
		}
		primaryKeys[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate MySQL primary key columns: %w", err)
	}
	if len(primaryKeys) == 0 {
		return errors.New("cannot update MySQL row: table has no primary key")
	}
	if len(whereColumns) != len(primaryKeys) {
		return errors.New("cannot update MySQL row without its complete primary key")
	}
	for name := range whereColumns {
		if _, ok := primaryKeys[name]; !ok {
			return errors.New("cannot update MySQL row without its complete primary key")
		}
	}

	setNames := make([]string, 0, len(setColumns))
	for name := range setColumns {
		setNames = append(setNames, name)
	}
	sort.Strings(setNames)

	setClauses := make([]string, 0, len(setNames))
	args := make([]any, 0, len(setColumns)+len(whereColumns))
	for _, name := range setNames {
		setClauses = append(setClauses, quoteIdentifier(name)+" = ?")
		args = append(args, setColumns[name])
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
		whereClauses = append(whereClauses, quoteIdentifier(name)+" = ?")
		args = append(args, whereColumns[name])
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		quoteIdentifier(table.Name),
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)
	m.logger.Log(query)

	result, err := m.database.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update MySQL row: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read MySQL update result: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("no row matched the WHERE clause; the row may have been modified or deleted")
	}
	return nil
}

func (m *mysqlDatabase) DeleteRow(ctx context.Context, table db.Table, whereColumns map[string]any) error {
	columns, err := m.ListColumns(ctx, table)
	if err != nil {
		return fmt.Errorf("list MySQL columns before delete: %w", err)
	}
	if err := db.ValidatePrimaryKeyWhere("delete MySQL row", columns, whereColumns); err != nil {
		return err
	}

	whereNames := make([]string, 0, len(whereColumns))
	for name := range whereColumns {
		whereNames = append(whereNames, name)
	}
	sort.Strings(whereNames)

	whereClauses := make([]string, 0, len(whereNames))
	args := make([]any, 0, len(whereNames))
	for _, name := range whereNames {
		if whereColumns[name] == nil {
			whereClauses = append(whereClauses, quoteIdentifier(name)+" IS NULL")
			continue
		}
		whereClauses = append(whereClauses, quoteIdentifier(name)+" = ?")
		args = append(args, whereColumns[name])
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		quoteIdentifier(table.Name),
		strings.Join(whereClauses, " AND "),
	)
	m.logger.Log(query)

	result, err := m.database.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete MySQL row: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read MySQL delete result: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("no row matched the WHERE clause; the row may have been modified or deleted")
	}
	return nil
}
