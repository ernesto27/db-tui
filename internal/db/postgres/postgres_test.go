package postgres_test

import (
	"context"
	"encoding/json"
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
	assert.Equal(t, db.EnginePostgreSQL, database.Engine(), "Database.Engine()")
	assert.Equal(t, "127.0.0.1", database.Host(), "Database.Host()")

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
		{Name: "PostgresIndexExample"},
		{Name: "Track"},
	}
	assert.Equal(t, want, tables, "ListTables()")
}

func TestListViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	views, err := database.ListViews(ctx)

	if !assert.NoError(t, err, "list views") {
		return
	}

	assert.Equal(t, []db.View{
		{Name: "AlbumSalesSummary"},
		{Name: "AlbumTrackSummary"},
		{Name: "ArtistAlbumSummary"},
		{Name: "CustomerCountrySummary"},
		{Name: "CustomerDirectory"},
		{Name: "CustomerInvoiceDetail"},
		{Name: "CustomerInvoiceSummary"},
		{Name: "EmployeeDirectory"},
		{Name: "GenreTrackSummary"},
		{Name: "InvoiceLineDetail"},
		{Name: "InvoiceMonthlySales"},
		{Name: "InvoiceSummary"},
		{Name: "MediaTypeTrackSummary"},
		{Name: "PlaylistSummary"},
		{Name: "PlaylistTrackDetail"},
		{Name: "PlaylistTrackSummary"},
		{Name: "SalesByCity"},
		{Name: "SupportRepCustomerSummary"},
		{Name: "TrackCatalog"},
		{Name: "TrackSalesSummary"},
	}, views, "ListViews()")
}

func TestListColumns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	columns, err := database.ListColumns(ctx, db.Table{Name: "Album"})
	if !assert.NoError(t, err, "list Album columns") {
		return
	}

	assert.Equal(t, []db.Column{
		{Name: "AlbumId", OrdinalPosition: 1, DataType: "int4", NotNull: true},
		{Name: "Title", OrdinalPosition: 2, DataType: "varchar(160)", Collation: "default", NotNull: true},
		{Name: "ArtistId", OrdinalPosition: 3, DataType: "int4", NotNull: true},
	}, columns)
}

func TestListIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	indexes, err := database.ListIndexes(ctx, db.Table{Name: "PostgresIndexExample"})
	if !assert.NoError(t, err, "list PostgresIndexExample indexes") {
		return
	}

	assert.Equal(t, []db.IndexColumns{
		{Name: "IX_PostgresIndexExample_Category", Column: "\"Category\"", Table: "PostgresIndexExample", AccessMethod: "btree"},
		{Name: "IX_PostgresIndexExample_CreatedAt", Column: "\"CreatedAt\"", Table: "PostgresIndexExample", AccessMethod: "btree"},
		{Name: "IX_PostgresIndexExample_SearchTerm", Column: "\"SearchTerm\"", Table: "PostgresIndexExample", AccessMethod: "btree"},
		{Name: "PostgresIndexExample_pkey", Column: "\"Id\"", Table: "PostgresIndexExample", AccessMethod: "btree"},
	}, indexes)
}

func TestListColumnsCompactsDroppedColumnPositions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	_, err = database.Execute(ctx, "CREATE TABLE public.list_columns_gap_demo (first int, middle int, last int)")
	if !assert.NoError(t, err, "create ListColumns test table") {
		return
	}
	t.Cleanup(func() {
		_, _ = database.Execute(context.Background(), "DROP TABLE IF EXISTS public.list_columns_gap_demo")
	})
	_, err = database.Execute(ctx, "ALTER TABLE public.list_columns_gap_demo DROP COLUMN middle")
	if !assert.NoError(t, err, "drop middle column") {
		return
	}

	columns, err := database.ListColumns(ctx, db.Table{Name: "list_columns_gap_demo"})
	if !assert.NoError(t, err, "list columns after dropping a column") {
		return
	}

	assert.Equal(t, []db.Column{
		{Name: "first", OrdinalPosition: 1, DataType: "int4"},
		{Name: "last", OrdinalPosition: 2, DataType: "int4"},
	}, columns)
}

func TestListColumnsNormalizesArrayTypeNames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	_, err = database.Execute(ctx, "CREATE TABLE public.list_columns_array_demo (tags varchar(10)[], codes char(3)[])")
	if !assert.NoError(t, err, "create ListColumns array test table") {
		return
	}
	t.Cleanup(func() {
		_, _ = database.Execute(context.Background(), "DROP TABLE IF EXISTS public.list_columns_array_demo")
	})

	columns, err := database.ListColumns(ctx, db.Table{Name: "list_columns_array_demo"})
	if !assert.NoError(t, err, "list array columns") {
		return
	}

	assert.Equal(t, []db.Column{
		{Name: "tags", OrdinalPosition: 1, DataType: "varchar(10)[]", Collation: "default"},
		{Name: "codes", OrdinalPosition: 2, DataType: "char(3)[]", Collation: "default"},
	}, columns)
}

func TestTableDDL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	ddl, err := database.TableDDL(ctx, db.Table{Name: "Album"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Contains(t, ddl, "CREATE TABLE public.\"Album\" (")
	assert.Contains(t, ddl, "\"AlbumId\" int4 NOT NULL")
	assert.Contains(t, ddl, `CONSTRAINT "PK_Album" PRIMARY KEY ("AlbumId")`)
	assert.Contains(t, ddl, `CONSTRAINT "FK_AlbumArtistId" FOREIGN KEY ("ArtistId") REFERENCES public."Artist"("ArtistId")`)
	assert.Contains(t, ddl, `CREATE INDEX "IFK_AlbumArtistId" ON public."Album" USING btree ("ArtistId");`)
	assert.NotContains(t, ddl, "ALTER TABLE")
}

func TestTableDDLIncludesColumnClauses(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)
	_, err = database.Execute(ctx, `CREATE TABLE public.ddl_review_demo (
		id serial PRIMARY KEY,
		created_at timestamptz NOT NULL DEFAULT now(),
		label varchar(20) COLLATE "C" DEFAULT 'x',
		total int GENERATED ALWAYS AS (id * 2) STORED,
		external_id int GENERATED ALWAYS AS IDENTITY
	)`)
	if !assert.NoError(t, err, "create DDL regression table") {
		return
	}
	t.Cleanup(func() {
		_, _ = database.Execute(context.Background(), "DROP TABLE IF EXISTS public.ddl_review_demo")
	})

	ddl, err := database.TableDDL(ctx, db.Table{Name: "ddl_review_demo"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Contains(t, ddl, `"id" int4 DEFAULT nextval(`)
	assert.Contains(t, ddl, `"created_at" timestamp with time zone DEFAULT now() NOT NULL`)
	assert.Contains(t, ddl, `"label" varchar(20) COLLATE "C" DEFAULT 'x'::character varying`)
	assert.Contains(t, ddl, `"total" int4 GENERATED ALWAYS AS ((id * 2)) STORED`)
	assert.Contains(t, ddl, `"external_id" int4 GENERATED ALWAYS AS IDENTITY`)
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

	assert.NoError(t, database.Export(ctx, db.Table{Name: "Album"}, db.ExportTypeCSV))
}

func TestExportJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Chdir(t.TempDir())

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	assert.NoError(t, database.Export(ctx, db.Table{Name: "Album"}, db.ExportTypeJSON))

	exportFiles, err := filepath.Glob("Album_*.json")
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
	assert.NotEmpty(t, document["Album"])
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
