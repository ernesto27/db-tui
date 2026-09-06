package sqlserver_test

import (
	"context"
	"net/url"
	"os"
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
