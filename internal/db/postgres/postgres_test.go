package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
	"github.com/stretchr/testify/assert"
)

const chinookDSN = "postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable"

func TestListTables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)
	assert.Equal(t, "chinook", database.Name(), "Database.Name()")

	tables, err := database.ListTables(ctx)
	if !assert.NoError(t, err, "list tables") {
		return
	}

	want := []db.Table{
		{Name: "Album"},
		{Name: "Artist"},
		{Name: "Customer"},
		{Name: "Employee"},
		{Name: "Genre"},
		{Name: "Invoice"},
		{Name: "InvoiceLine"},
		{Name: "MediaType"},
		{Name: "Playlist"},
		{Name: "PlaylistTrack"},
		{Name: "Track"},
	}
	assert.Equal(t, want, tables, "ListTables()")
}

func TestConnectReturnsErrorForUnreachableDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, "postgres://db_tui@127.0.0.1:1/chinook?sslmode=disable&connect_timeout=1")
	if database != nil {
		database.Close()
	}
	assert.Error(t, err, "Connect() to an unreachable database")
}

func TestDumpCreatesSQLFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)
	t.Chdir(t.TempDir())

	if !assert.NoError(t, database.Dump(ctx), "dump database") {
		return
	}

	dumpFiles, err := filepath.Glob("chinook_*.sql")
	if !assert.NoError(t, err, "find generated SQL dump") || !assert.Len(t, dumpFiles, 1) {
		return
	}

	contents, err := os.ReadFile(dumpFiles[0])
	if !assert.NoError(t, err, "read generated SQL dump") {
		return
	}
	assert.Contains(t, string(contents), `CREATE TABLE public."Album"`)
}

func TestExport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Chdir(t.TempDir())

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	assert.NoError(t, database.Export(ctx, db.Table{Name: "Album"}))
}

func TestExportQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Chdir(t.TempDir())

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	assert.NoError(t, database.ExportQuery(ctx, "SELECT generate_series(1, 101) AS number"))

	exportFiles, err := filepath.Glob("query_*.csv")
	if !assert.NoError(t, err, "find generated CSV query export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated CSV query export") {
		return
	}
	assert.Contains(t, string(contents), "number\n")
	assert.Contains(t, string(contents), "\n101\n")
}

func TestGetRows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

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
			table:        db.Table{Name: "Album"},
			page:         db.PageRequest{Limit: 2},
			wantColumns:  []string{"AlbumId", "Title", "ArtistId"},
			wantRowCount: 2,
			wantHasMore:  true,
		},
		{
			name:         "page past end",
			table:        db.Table{Name: "Album"},
			page:         db.PageRequest{Offset: 10000, Limit: 2},
			wantColumns:  []string{"AlbumId", "Title", "ArtistId"},
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
			table:   db.Table{Name: "Album"},
			page:    db.PageRequest{Offset: -1, Limit: 1},
			wantErr: true,
		},
		{
			name:    "zero limit",
			table:   db.Table{Name: "Album"},
			page:    db.PageRequest{},
			wantErr: true,
		},
		{
			name:    "limit above maximum",
			table:   db.Table{Name: "Album"},
			page:    db.PageRequest{Limit: db.MaxPageSize + 1},
			wantErr: true,
		},
		{
			name:    "malicious table name is quoted",
			table:   db.Table{Name: "Album\"; DROP TABLE public.\"Artist\"; --"},
			page:    db.PageRequest{Limit: 1},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, err := database.GetRows(ctx, test.table, test.page)
			if test.wantErr {
				assert.Error(t, err)
			} else if assert.NoError(t, err) {
				assert.Equal(t, test.wantColumns, page.Columns)
				assert.Len(t, page.Rows, test.wantRowCount)
				assert.Equal(t, test.wantHasMore, page.HasMore)
				if len(page.Rows) > 0 {
					assert.Len(t, page.Rows[0], len(page.Columns))
				}
			}
		})
	}
}
