package mysql

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ernestoponce27/db-tui/internal/db"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const worldDSN = "db_tui:db_tui@tcp(127.0.0.1:3307)/world?parseTime=true"

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name       string
		dsn        string
		wantUser   string
		wantPass   string
		wantAddr   string
		wantDBName string
		wantParse  bool
		wantErr    string
	}{
		{
			name:       "driver DSN",
			dsn:        "db_tui:secret@tcp(127.0.0.1:3307)/chinook?parseTime=true",
			wantUser:   "db_tui",
			wantPass:   "secret",
			wantAddr:   "127.0.0.1:3307",
			wantDBName: "chinook",
			wantParse:  true,
		},
		{
			name:       "URL DSN with defaults and escaping",
			dsn:        "mysql://db%20user:p%40ss@localhost/sales%20data?parseTime=true",
			wantUser:   "db user",
			wantPass:   "p@ss",
			wantAddr:   "localhost:3306",
			wantDBName: "sales data",
			wantParse:  true,
		},
		{
			name:    "driver DSN requires database",
			dsn:     "db_tui@tcp(127.0.0.1:3306)/",
			wantErr: "database name is required",
		},
		{
			name:    "URL requires username",
			dsn:     "mysql://localhost:3306/chinook",
			wantErr: "username is required",
		},
		{
			name:    "URL requires host",
			dsn:     "mysql://db_tui@/chinook",
			wantErr: "host is required",
		},
		{
			name:    "URL requires database",
			dsn:     "mysql://db_tui@localhost:3306",
			wantErr: "database name is required",
		},
		{
			name:    "URL validates port",
			dsn:     "mysql://db_tui@localhost:70000/chinook",
			wantErr: "port must be between 1 and 65535",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := parseDSN(test.dsn)
			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantUser, config.User)
			assert.Equal(t, test.wantPass, config.Passwd)
			assert.Equal(t, test.wantAddr, config.Addr)
			assert.Equal(t, test.wantDBName, config.DBName)
			assert.Equal(t, test.wantParse, config.ParseTime)
		})
	}
}

func TestListTables(t *testing.T) {
	database := connectWorld(t)

	tables, err := database.ListTables(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []db.Table{{Name: "city"}, {Name: "country"}, {Name: "countrylanguage"}}, tables)
}

func TestGetRows(t *testing.T) {
	database := connectWorld(t)

	page, err := database.GetRows(
		context.Background(),
		db.Table{Name: "city"},
		db.PageRequest{Limit: 2},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"ID", "Name", "CountryCode", "District", "Population"}, page.Columns)
	assert.Len(t, page.Rows, 2)
	assert.Len(t, page.Rows[0], len(page.Columns))
	assert.True(t, page.HasMore)
}

func TestGetRowsQuotesTableName(t *testing.T) {
	database := connectWorld(t)

	_, err := database.GetRows(
		context.Background(),
		db.Table{Name: "city`; DROP TABLE country; --"},
		db.PageRequest{Limit: 1},
	)

	assert.ErrorContains(t, err, "query MySQL rows")
}

func TestGetRowsValidatesPage(t *testing.T) {
	database := connectWorld(t)
	tests := []struct {
		name  string
		table db.Table
		page  db.PageRequest
		error string
	}{
		{name: "empty table", page: db.PageRequest{Limit: 1}, error: "table name is required"},
		{name: "negative offset", table: db.Table{Name: "city"}, page: db.PageRequest{Offset: -1, Limit: 1}, error: "page offset cannot be negative"},
		{name: "zero limit", table: db.Table{Name: "city"}, error: "page limit must be between 1 and 100"},
		{name: "large limit", table: db.Table{Name: "city"}, page: db.PageRequest{Limit: db.MaxPageSize + 1}, error: "page limit must be between 1 and 100"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := database.GetRows(context.Background(), test.table, test.page)
			assert.EqualError(t, err, test.error)
		})
	}
}

func TestExport(t *testing.T) {
	database := connectWorld(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.Export(context.Background(), db.Table{Name: "city"}))

	exportFiles, err := filepath.Glob("city_*.csv")
	if !assert.NoError(t, err, "find generated CSV export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated CSV export") {
		return
	}
	assert.Contains(t, string(contents), "ID,Name,CountryCode,District,Population\n")
}

func TestExportQuery(t *testing.T) {
	database := connectWorld(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.ExportQuery(context.Background(), "SELECT ID FROM city"))

	exportFiles, err := filepath.Glob("query_*.csv")
	if !assert.NoError(t, err, "find generated CSV query export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated CSV query export") {
		return
	}
	assert.Contains(t, string(contents), "ID\n")
}

func TestExecute(t *testing.T) {
	database := connectWorld(t)

	t.Run("returns bounded rows", func(t *testing.T) {
		result, err := database.Execute(context.Background(), "SELECT ID FROM city")

		require.NoError(t, err)
		assert.Equal(t, []string{"ID"}, result.Columns)
		assert.Len(t, result.Rows, db.MaxQueryResultRows)
		assert.IsType(t, int64(0), result.Rows[0][0])
		assert.Equal(t, "SELECT", result.CommandTag)
	})

	t.Run("returns command tag", func(t *testing.T) {
		result, err := database.Execute(context.Background(), "CREATE TEMPORARY TABLE integration_example (id integer)")

		require.NoError(t, err)
		assert.Empty(t, result.Columns)
		assert.Empty(t, result.Rows)
		assert.Equal(t, "CREATE TABLE", result.CommandTag)
	})

	t.Run("wraps query errors", func(t *testing.T) {
		_, err := database.Execute(context.Background(), "SELECT * FROM missing")

		assert.ErrorContains(t, err, "execute MySQL query")
	})
}

func TestDumpRejectsUnsupportedNetwork(t *testing.T) {
	database := &mysqlDatabase{
		config: &mysqldriver.Config{
			Net:    "udp",
			DBName: "chinook",
		},
	}

	err := database.Dump(context.Background())

	assert.EqualError(t, err, `unsupported MySQL network "udp" for dump`)
}

func connectWorld(t *testing.T) db.Database {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := Connect(ctx, worldDSN)
	require.NoError(t, err)
	t.Cleanup(func() {
		database.Close()
	})
	return database
}
