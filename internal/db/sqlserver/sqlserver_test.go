package sqlserver_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ernestoponce27/db-tui/internal/db"
	sqlserver "github.com/ernestoponce27/db-tui/internal/db/sqlserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTables(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)
	assert.Equal(t, "db_tui", database.Name(), "Database.Name()")
	assert.Equal(t, db.EngineSqlServer, database.Engine(), "Database.Engine()")

	tables, err := database.ListTables(ctx, "dbo")
	require.NoError(t, err)
	assert.Equal(t, []db.Table{
		{Schema: "dbo", Name: "cities"},
		{Schema: "dbo", Name: "countries"},
	}, tables)

}

func TestListColumns(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	columns, err := database.ListColumns(ctx, db.Table{Schema: "dbo", Name: "cities"})
	require.NoError(t, err)

	assert.Equal(t, []db.Column{
		{Name: "city_id", OrdinalPosition: 1, DataType: "int", NotNull: true, IsPrimaryKey: true},
		{Name: "country_code", OrdinalPosition: 2, DataType: "char(2)", Collation: "SQL_Latin1_General_CP1_CI_AS", NotNull: true},
		{Name: "name", OrdinalPosition: 3, DataType: "nvarchar(100)", Collation: "SQL_Latin1_General_CP1_CI_AS", NotNull: true},
		{Name: "population", OrdinalPosition: 4, DataType: "int"},
	}, columns)
}

func TestListIndexes(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	indexes, err := database.ListIndexes(ctx, db.Table{Schema: "dbo", Name: "cities"})
	if !assert.NoError(t, err, "list cities indexes") {
		return
	}
	if !assert.Len(t, indexes, 1) {
		return
	}

	assert.Regexp(t, `^PK__cities__`, indexes[0].Name)
	assert.Equal(t, "city_id", indexes[0].Column)
	assert.Equal(t, "cities", indexes[0].Table)
	assert.Equal(t, "CLUSTERED", indexes[0].AccessMethod)
}

func TestListViews(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	views, err := database.ListViews(ctx, "dbo")
	if !assert.NoError(t, err, "list views") {
		return
	}

	assert.Equal(t, []db.View{
		{Name: "city_directory"},
		{Name: "country_city_counts"},
	}, views)
}

func TestListFunctions(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	functions, err := database.ListFunctions(ctx, "dbo")
	if !assert.NoError(t, err, "list functions") {
		return
	}

	assert.Equal(t, []string{
		"cities_by_country",
		"city_count",
	}, sqlServerFunctionNames(functions), "ListFunctions() names")

	expectedMetadata := map[string]struct {
		arguments  string
		returnType string
	}{
		"cities_by_country": {arguments: "@country_code char(2)", returnType: "TABLE"},
		"city_count":        {arguments: "", returnType: "int"},
	}
	for _, function := range functions {
		expected, ok := expectedMetadata[function.Name]
		if !assert.True(t, ok, "unexpected function %q", function.Name) {
			continue
		}
		assert.Equal(t, expected.arguments, function.Arguments, "%s arguments", function.Name)
		assert.Equal(t, expected.returnType, function.ReturnType, "%s return type", function.Name)
		assert.Equal(t, "SQL", function.Language, "%s language", function.Name)
		assert.Contains(t, function.Definition, "FUNCTION dbo."+function.Name, "%s definition", function.Name)
	}
}

func sqlServerFunctionNames(functions []db.FunctionColumns) []string {
	names := make([]string, len(functions))
	for index, function := range functions {
		names[index] = function.Name
	}
	return names
}

func TestListSchemaObjectGroups(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	groups, err := database.ListSchemaObjectGroups(ctx)
	require.NoError(t, err)
	assert.Contains(t, groups, db.SchemaObjectGroup{Schema: "dbo", Type: db.SchemaObjectTables})
	assert.Contains(t, groups, db.SchemaObjectGroup{Schema: "dbo", Type: db.SchemaObjectViews})
	assert.Contains(t, groups, db.SchemaObjectGroup{Schema: "dbo", Type: db.SchemaObjectFunctions})
	for _, group := range groups {
		assert.NotEqual(t, "sys", group.Schema)
		assert.NotEqual(t, "INFORMATION_SCHEMA", group.Schema)
	}
}

func TestGetRows(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	tests := []struct {
		name         string
		table        db.Table
		page         db.PageRequest
		wantColumns  []string
		wantRowCount int
		wantHasMore  bool
		wantErr      bool
	}{
		{
			name:         "first page",
			table:        db.Table{Schema: "dbo", Name: "cities"},
			page:         db.PageRequest{Limit: 2},
			wantColumns:  []string{"city_id", "country_code", "name", "population"},
			wantRowCount: 2,
			wantHasMore:  true,
		},
		{
			name:         "page past end",
			table:        db.Table{Schema: "dbo", Name: "cities"},
			page:         db.PageRequest{Offset: 10000, Limit: 2},
			wantColumns:  []string{"city_id", "country_code", "name", "population"},
			wantRowCount: 0,
		},
		{
			name:    "empty table name",
			table:   db.Table{},
			page:    db.PageRequest{Limit: 1},
			wantErr: true,
		},
		{
			name:    "negative offset",
			table:   db.Table{Schema: "dbo", Name: "cities"},
			page:    db.PageRequest{Offset: -1, Limit: 1},
			wantErr: true,
		},
		{
			name:    "zero limit",
			table:   db.Table{Schema: "dbo", Name: "cities"},
			page:    db.PageRequest{},
			wantErr: true,
		},
		{
			name:    "malicious table name is quoted",
			table:   db.Table{Schema: "dbo", Name: "cities]; DROP TABLE dbo.countries; --"},
			page:    db.PageRequest{Limit: 1},
			wantErr: true,
		},
	}

	for _, test := range tests {
		page, err := database.GetRows(ctx, test.table, test.page)
		if test.wantErr {
			assert.Error(t, err, test.name)
			continue
		}
		if assert.NoError(t, err, test.name) {
			assert.Equal(t, test.wantColumns, page.Columns, test.name)
			assert.Len(t, page.Rows, test.wantRowCount, test.name)
			assert.Equal(t, test.wantHasMore, page.HasMore, test.name)
			if len(page.Rows) > 0 {
				assert.Len(t, page.Rows[0], len(page.Columns), test.name)
			}
		}
	}

	_, err := database.GetRows(ctx, db.Table{Schema: "dbo", Name: "cities"}, db.PageRequest{Limit: db.MaxPageSize + 1})
	assert.NoError(t, err)
}

func TestTableDDL(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	ddl, err := database.TableDDL(ctx, db.Table{Name: "countries"})
	require.NoError(t, err)
	assert.Contains(t, ddl, `CREATE TABLE [dbo].[countries] (
    [country_code] char(2) NOT NULL,
    [name] nvarchar(100) NOT NULL,`)
	assert.Regexp(t, `CONSTRAINT \[PK__countrie__[A-F0-9]+\] PRIMARY KEY \(\[country_code\]\)`, ddl)
}

func TestExecute(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	tests := []struct {
		name          string
		statement     string
		wantColumns   []string
		wantRowCount  int
		wantValueType any
		wantCommand   string
		wantErr       string
		cancel        bool
	}{
		{
			name: "returns bounded rows",
			statement: `
				SELECT TOP (101)
					ROW_NUMBER() OVER (ORDER BY object_id) AS id
				FROM sys.all_objects`,
			wantColumns:   []string{"id"},
			wantRowCount:  db.MaxPageSize,
			wantValueType: int64(0),
			wantCommand:   "SELECT",
		},
		{
			name:        "returns command tag",
			statement:   "CREATE TABLE #execute_integration_example (id INT)",
			wantColumns: []string{},
			wantCommand: "CREATE TABLE",
		},
		{
			name:      "wraps query errors",
			statement: "SELECT * FROM dbo.missing",
			wantErr:   "execute SQL Server query",
		},
		{
			name:      "cancels a running query",
			statement: "WAITFOR DELAY '00:00:10'",
			cancel:    true,
		},
	}

	for _, test := range tests {
		queryCtx := ctx
		cancelQuery := func() {}
		cancelTimer := (*time.Timer)(nil)
		started := time.Now()
		if test.cancel {
			queryCtx, cancelQuery = context.WithCancel(ctx)
			cancelTimer = time.AfterFunc(250*time.Millisecond, cancelQuery)
		}

		result, err := database.Execute(queryCtx, test.statement)
		if cancelTimer != nil {
			cancelTimer.Stop()
		}
		cancelQuery()

		if test.cancel {
			assert.ErrorIs(t, err, context.Canceled, test.name)
			assert.Less(t, time.Since(started), 2*time.Second, test.name)
			continue
		}
		if test.wantErr != "" {
			assert.ErrorContains(t, err, test.wantErr, test.name)
			continue
		}

		if !assert.NoError(t, err, test.name) {
			continue
		}
		assert.Equal(t, test.wantCommand, result.CommandTag, test.name)
		assert.Equal(t, test.wantColumns, result.Columns, test.name)
		assert.Len(t, result.Rows, test.wantRowCount, test.name)
		if test.wantValueType != nil && len(result.Rows) > 0 {
			assert.IsType(t, test.wantValueType, result.Rows[0][0], test.name)
		}
	}
}

func TestUpdateRow(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)
	table := db.Table{Schema: "dbo", Name: "cities"}

	result, err := database.Execute(ctx, "SELECT name FROM dbo.cities WHERE city_id = 1")
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	originalName, ok := result.Rows[0][0].(string)
	require.True(t, ok)

	updated := false
	t.Cleanup(func() {
		if !updated {
			return
		}
		assert.NoError(t, database.UpdateRow(context.Background(), table, map[string]any{"name": originalName}, map[string]any{"city_id": 1}))
	})

	tests := []struct {
		name         string
		table        db.Table
		setColumns   map[string]any
		whereColumns map[string]any
		wantErr      string
	}{
		{
			name:         "empty table name",
			table:        db.Table{},
			setColumns:   map[string]any{"name": "x"},
			whereColumns: map[string]any{"city_id": 1},
			wantErr:      "table has no primary key",
		},
		{
			name:         "empty set columns",
			table:        table,
			setColumns:   map[string]any{},
			whereColumns: map[string]any{"city_id": 1},
			wantErr:      "requires at least one column",
		},
		{
			name:       "empty where columns",
			table:      table,
			setColumns: map[string]any{"name": "x"},
			wantErr:    "complete primary key",
		},
		{
			name:         "non-primary-key where column",
			table:        table,
			setColumns:   map[string]any{"name": "x"},
			whereColumns: map[string]any{"name": originalName},
			wantErr:      "complete primary key",
		},
		{
			name:         "missing row",
			table:        table,
			setColumns:   map[string]any{"name": "x"},
			whereColumns: map[string]any{"city_id": -1},
			wantErr:      "no row matched",
		},
		{
			name:         "successful update",
			table:        table,
			setColumns:   map[string]any{"name": "update_test"},
			whereColumns: map[string]any{"city_id": 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := database.UpdateRow(ctx, test.table, test.setColumns, test.whereColumns)
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				return
			}
			if assert.NoError(t, err) {
				updated = true
			}
		})
	}

	result, err = database.Execute(ctx, "SELECT name FROM dbo.cities WHERE city_id = 1")
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, "update_test", result.Rows[0][0])
}

func TestDeleteRow(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)
	table := db.Table{Schema: "dbo", Name: "cities"}

	_, err := database.Execute(ctx, "DELETE FROM dbo.cities WHERE city_id = 999999")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := database.Execute(context.Background(), "DELETE FROM dbo.cities WHERE city_id = 999999")
		assert.NoError(t, err)
	})

	_, err = database.Execute(ctx, "INSERT INTO dbo.cities (city_id, country_code, name, population) VALUES (999999, 'AR', 'delete_test', 0)")
	require.NoError(t, err)

	tests := []struct {
		name         string
		table        db.Table
		whereColumns map[string]any
		wantErr      string
	}{
		{
			name:         "empty table name",
			table:        db.Table{},
			whereColumns: map[string]any{"city_id": 1},
			wantErr:      "table has no primary key",
		},
		{
			name:    "empty where columns",
			table:   table,
			wantErr: "complete primary key",
		},
		{
			name:         "non-primary-key where column",
			table:        table,
			whereColumns: map[string]any{"name": "delete_test"},
			wantErr:      "complete primary key",
		},
		{
			name:         "missing row",
			table:        table,
			whereColumns: map[string]any{"city_id": -1},
			wantErr:      "no row matched",
		},
		{
			name:         "successful delete",
			table:        table,
			whereColumns: map[string]any{"city_id": 999999},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := database.DeleteRow(ctx, test.table, test.whereColumns)
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}

	result, err := database.Execute(ctx, "SELECT city_id FROM dbo.cities WHERE city_id = 999999")
	require.NoError(t, err)
	assert.Empty(t, result.Rows)
}

func TestDump(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)

	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	temporaryDirectory := t.TempDir()
	require.NoError(t, os.Chdir(temporaryDirectory))
	t.Cleanup(func() {
		_ = os.Chdir(workingDirectory)
	})

	require.NoError(t, database.Dump(ctx))

	entries, err := os.ReadDir(temporaryDirectory)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, strings.HasPrefix(entries[0].Name(), "db_tui_"))
	assert.True(t, strings.HasSuffix(entries[0].Name(), ".bak"))

	info, err := entries[0].Info()
	require.NoError(t, err)
	assert.Positive(t, info.Size())
}

func TestExportCSV(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.Export(ctx, db.Table{Schema: "dbo", Name: "cities"}, db.ExportTypeCSV))

	exportFiles, err := filepath.Glob("cities_*.csv")
	if !assert.NoError(t, err, "find generated CSV export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated CSV export") {
		return
	}
	assert.Contains(t, string(contents), "city_id,country_code,name,population\n")
}

func TestExportJSON(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.Export(ctx, db.Table{Schema: "dbo", Name: "cities"}, db.ExportTypeJSON))

	exportFiles, err := filepath.Glob("cities_*.json")
	if !assert.NoError(t, err, "find generated JSON export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated JSON export") {
		return
	}
	var document map[string][]map[string]any
	if !assert.NoError(t, json.Unmarshal(contents, &document)) {
		return
	}
	assert.NotEmpty(t, document["cities"])
}

func TestExportQuery(t *testing.T) {
	ctx, database := connectLocalSQLServer(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.ExportQuery(ctx, "SELECT city_id FROM dbo.cities"))

	exportFiles, err := filepath.Glob("query_*.csv")
	if !assert.NoError(t, err, "find generated CSV query export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated CSV query export") {
		return
	}
	assert.Contains(t, string(contents), "city_id\n")

	assert.EqualError(t, database.ExportQuery(ctx, "UPDATE dbo.cities SET name = 'x'"), "only SELECT queries can be exported")
}

func connectLocalSQLServer(t *testing.T) (context.Context, db.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	database, err := sqlserver.Connect(ctx, localSQLServerDSN("DbTuiSql2026!"))
	require.NoError(t, err, "connect to local Compose SQL Server")
	t.Cleanup(database.Close)
	return ctx, database
}

func localSQLServerDSN(password string) string {
	values := url.Values{
		"database":               {"db_tui"},
		"encrypt":                {"true"},
		"trustservercertificate": {"true"},
	}
	return (&url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword("sa", password),
		Host:     "localhost:1434",
		RawQuery: values.Encode(),
	}).String()
}
