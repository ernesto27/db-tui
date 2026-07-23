package mysql

import (
	"context"
	"errors"
	"io"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/logger"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	database, mock := newMockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta(listTablesSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}).
			AddRow("Album").
			AddRow("Artist"))

	tables, err := database.ListTables(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []db.Table{{Name: "Album"}, {Name: "Artist"}}, tables)
}

func TestGetRows(t *testing.T) {
	database, mock := newMockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `Album` LIMIT ? OFFSET ?")).
		WithArgs(3, 4).
		WillReturnRows(sqlmock.NewRows([]string{"AlbumId", "Title", "Notes"}).
			AddRow(int64(5), []byte("title 5"), nil).
			AddRow(int64(6), []byte("title 6"), nil).
			AddRow(int64(7), []byte("title 7"), nil))

	page, err := database.GetRows(
		context.Background(),
		db.Table{Name: "Album"},
		db.PageRequest{Offset: 4, Limit: 2},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"AlbumId", "Title", "Notes"}, page.Columns)
	assert.Equal(t, [][]any{
		{int64(5), "title 5", nil},
		{int64(6), "title 6", nil},
	}, page.Rows)
	assert.True(t, page.HasMore)
}

func TestGetRowsQuotesTableName(t *testing.T) {
	database, mock := newMockDatabase(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `Album``; DROP TABLE Artist; --` LIMIT ? OFFSET ?")).
		WithArgs(2, 0).
		WillReturnError(errors.New("table does not exist"))

	_, err := database.GetRows(
		context.Background(),
		db.Table{Name: "Album`; DROP TABLE Artist; --"},
		db.PageRequest{Limit: 1},
	)

	assert.ErrorContains(t, err, "query MySQL rows")
}

func TestGetRowsValidatesPage(t *testing.T) {
	database, _ := newMockDatabase(t)
	tests := []struct {
		name  string
		table db.Table
		page  db.PageRequest
		error string
	}{
		{name: "empty table", page: db.PageRequest{Limit: 1}, error: "table name is required"},
		{name: "negative offset", table: db.Table{Name: "Album"}, page: db.PageRequest{Offset: -1, Limit: 1}, error: "page offset cannot be negative"},
		{name: "zero limit", table: db.Table{Name: "Album"}, error: "page limit must be between 1 and 100"},
		{name: "large limit", table: db.Table{Name: "Album"}, page: db.PageRequest{Limit: db.MaxPageSize + 1}, error: "page limit must be between 1 and 100"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := database.GetRows(context.Background(), test.table, test.page)
			assert.EqualError(t, err, test.error)
		})
	}
}

func TestExecute(t *testing.T) {
	t.Run("returns bounded rows", func(t *testing.T) {
		database, mock := newMockDatabase(t)
		rows := sqlmock.NewRows([]string{"number"})
		for number := 1; number <= db.MaxQueryResultRows+1; number++ {
			rows.AddRow(int64(number))
		}
		mock.ExpectQuery(regexp.QuoteMeta("SELECT number FROM numbers")).WillReturnRows(rows)

		result, err := database.Execute(context.Background(), "SELECT number FROM numbers")

		require.NoError(t, err)
		assert.Equal(t, []string{"number"}, result.Columns)
		assert.Len(t, result.Rows, db.MaxQueryResultRows)
		assert.Equal(t, int64(1), result.Rows[0][0])
		assert.Equal(t, int64(100), result.Rows[len(result.Rows)-1][0])
		assert.Equal(t, "SELECT", result.CommandTag)
	})

	t.Run("returns command tag", func(t *testing.T) {
		database, mock := newMockDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta("CREATE TEMPORARY TABLE example (id integer)")).
			WillReturnRows(sqlmock.NewRows([]string{}))

		result, err := database.Execute(context.Background(), "CREATE TEMPORARY TABLE example (id integer)")

		require.NoError(t, err)
		assert.Empty(t, result.Columns)
		assert.Empty(t, result.Rows)
		assert.Equal(t, "CREATE TABLE", result.CommandTag)
	})

	t.Run("wraps query errors", func(t *testing.T) {
		database, mock := newMockDatabase(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM missing")).
			WillReturnError(errors.New("missing table"))

		_, err := database.Execute(context.Background(), "SELECT * FROM missing")

		assert.ErrorContains(t, err, "execute MySQL query")
	})
}

func TestSafeFilename(t *testing.T) {
	assert.Equal(t, "chinook", safeFilename("chinook"))
	assert.Equal(t, "mysql", safeFilename(".."))
	assert.Equal(t, "database_name", safeFilename("database name"))
	assert.Equal(t, "passwd", safeFilename("../../passwd"))
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

func newMockDatabase(t *testing.T) (*mysqlDatabase, sqlmock.Sqlmock) {
	t.Helper()
	sqlDatabase, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		_ = sqlDatabase.Close()
		require.NoError(t, mock.ExpectationsWereMet())
	})
	return &mysqlDatabase{
		database: sqlDatabase,
		logger:   logger.New(io.Discard),
	}, mock
}
