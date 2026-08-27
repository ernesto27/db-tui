package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	tables, err := database.ListTables(ctx, "public")
	if !assert.NoError(t, err, "list tables") {
		return
	}

	want := []db.Table{
		{Schema: "public", Name: "Album"},
		{Schema: "public", Name: "Artist"},
		{Schema: "public", Name: "Customer"},
		{Schema: "public", Name: "Employee"},
		{Schema: "public", Name: "Genre"},
		{Schema: "public", Name: "Invoice"},
		{Schema: "public", Name: "InvoiceLine"},
		{Schema: "public", Name: "MediaType"},
		{Schema: "public", Name: "Playlist"},
		{Schema: "public", Name: "PlaylistTrack"},
		{Schema: "public", Name: "PostgresIndexExample"},
		{Schema: "public", Name: "Track"},
	}
	assert.Equal(t, want, tables, "ListTables()")
}

func TestListSchemaObjectGroups(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	groups, err := database.ListSchemaObjectGroups(ctx)
	if !assert.NoError(t, err, "list schema object groups") {
		return
	}

	assert.Contains(t, groups, db.SchemaObjectGroup{Schema: "public", Type: db.SchemaObjectTables})
	assert.Contains(t, groups, db.SchemaObjectGroup{Schema: "public", Type: db.SchemaObjectViews})
	assert.Contains(t, groups, db.SchemaObjectGroup{Schema: "public", Type: db.SchemaObjectMaterializedViews})
	assert.Contains(t, groups, db.SchemaObjectGroup{Schema: "public", Type: db.SchemaObjectFunctions})
	for _, group := range groups {
		assert.NotEqual(t, "information_schema", group.Schema)
		assert.NotEqual(t, "pg_catalog", group.Schema)
		assert.False(t, strings.HasPrefix(group.Schema, "pg_"))
	}
}

func TestListViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	views, err := database.ListViews(ctx, "public")

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

func TestListMaterializedViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	materializedViews, err := database.ListMaterializedViews(ctx, "public")
	if !assert.NoError(t, err, "list materialized views") {
		return
	}

	assert.Equal(t, []db.MaterializedView{
		{Name: "AlbumSalesMetrics"},
		{Name: "AlbumTrackMetrics"},
		{Name: "ArtistCatalogMetrics"},
		{Name: "ArtistSalesMetrics"},
		{Name: "CitySalesMetrics"},
		{Name: "CustomerGenrePurchaseMetrics"},
		{Name: "CustomerInvoiceTimeline"},
		{Name: "CustomerLocationMetrics"},
		{Name: "CustomerPurchaseMetrics"},
		{Name: "EmployeeSupportMetrics"},
		{Name: "GenreCatalogMetrics"},
		{Name: "GenreSalesMetrics"},
		{Name: "InvoiceDailyRevenueSnapshot"},
		{Name: "InvoiceLineRevenueSnapshot"},
		{Name: "InvoiceMonthlyRevenueSnapshot"},
		{Name: "MediaTypeCatalogMetrics"},
		{Name: "PlaylistCatalogMetrics"},
		{Name: "PlaylistGenreMetrics"},
		{Name: "SalesByCountry"},
		{Name: "TrackSalesMetrics"},
		{Name: "YearlyRevenueSnapshot"},
	}, materializedViews, "ListMaterializedViews()")
}

func TestListFunctions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	functions, err := database.ListFunctions(ctx, "public")
	if !assert.NoError(t, err, "list functions") {
		return
	}

	assert.Equal(t, []string{
		"album_track_count",
		"customer_full_name",
		"customer_lifetime_spend",
		"search_tracks",
		"track_duration_seconds",
	}, functionNames(functions), "ListFunctions() names")

	expectedMetadata := map[string]struct {
		arguments  string
		returnType string
	}{
		"album_track_count":       {arguments: "album_id integer", returnType: "bigint"},
		"customer_full_name":      {arguments: "customer_id integer", returnType: "text"},
		"customer_lifetime_spend": {arguments: "customer_id integer", returnType: "numeric"},
		"search_tracks":           {arguments: "search_text text"},
		"track_duration_seconds":  {arguments: "track_id integer", returnType: "numeric"},
	}
	for _, function := range functions {
		expected, ok := expectedMetadata[function.Name]
		if !assert.True(t, ok, "unexpected function %q", function.Name) {
			continue
		}
		assert.Equal(t, expected.arguments, function.Arguments, "%s arguments", function.Name)
		if expected.returnType != "" {
			assert.Equal(t, expected.returnType, function.ReturnType, "%s return type", function.Name)
		}
		assert.Equal(t, "sql", function.Language, "%s language", function.Name)
		assert.Contains(t, function.Definition, "CREATE OR REPLACE FUNCTION public."+function.Name, "%s definition", function.Name)
	}
}

func functionNames(functions []db.FunctionColumns) []string {
	names := make([]string, len(functions))
	for index, function := range functions {
		names[index] = function.Name
	}
	return names
}

func TestListColumns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	columns, err := database.ListColumns(ctx, db.Table{Schema: "public", Name: "Album"})
	if !assert.NoError(t, err, "list Album columns") {
		return
	}

	assert.Equal(t, []db.Column{
		{Name: "AlbumId", OrdinalPosition: 1, DataType: "int4", NotNull: true, IsPrimaryKey: true},
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

	indexes, err := database.ListIndexes(ctx, db.Table{Schema: "public", Name: "PostgresIndexExample"})
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

func TestTableOperationsUseSelectedSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	const schema = "analytics"
	const tableName = "DailyRevenue"
	t.Cleanup(func() {
		_, _ = database.Execute(context.Background(), `
			INSERT INTO analytics."DailyRevenue" ("RevenueDate", "InvoiceCount", "Revenue")
			VALUES ('2024-01-01', 12, 34.87)
			ON CONFLICT ("RevenueDate") DO UPDATE
			SET "InvoiceCount" = EXCLUDED."InvoiceCount", "Revenue" = EXCLUDED."Revenue"`)
	})

	table := db.Table{Schema: schema, Name: tableName}
	columns, err := database.ListColumns(ctx, table)
	if !assert.NoError(t, err, "list selected-schema columns") {
		return
	}
	assert.Len(t, columns, 3)
	assert.Equal(t, "RevenueDate", columns[0].Name)
	assert.Equal(t, "InvoiceCount", columns[1].Name)
	assert.Equal(t, "Revenue", columns[2].Name)

	indexes, err := database.ListIndexes(ctx, table)
	if !assert.NoError(t, err, "list selected-schema indexes") {
		return
	}
	assert.Contains(t, indexes, db.IndexColumns{
		Name:         "DailyRevenue_pkey",
		Column:       `"RevenueDate"`,
		Table:        tableName,
		AccessMethod: "btree",
	})

	err = database.UpdateRow(ctx, table, map[string]any{"InvoiceCount": 13}, map[string]any{"RevenueDate": "2024-01-01"})
	if !assert.NoError(t, err, "update selected-schema row") {
		return
	}
	updatedRow, err := database.Execute(ctx, `SELECT "InvoiceCount" FROM analytics."DailyRevenue" WHERE "RevenueDate" = '2024-01-01'`)
	if !assert.NoError(t, err, "read selected-schema row after update") {
		return
	}
	assert.EqualValues(t, 13, updatedRow.Rows[0][0])

	err = database.DeleteRow(ctx, table, map[string]any{"RevenueDate": "2024-01-01"})
	if !assert.NoError(t, err, "delete selected-schema row") {
		return
	}
	remainingRows, err := database.Execute(ctx, `SELECT COUNT(*) FROM analytics."DailyRevenue" WHERE "RevenueDate" = '2024-01-01'`)
	if assert.NoError(t, err, "read selected-schema row after delete") {
		assert.EqualValues(t, 0, remainingRows.Rows[0][0])
	}
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

	columns, err := database.ListColumns(ctx, db.Table{Schema: "public", Name: "list_columns_gap_demo"})
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

	columns, err := database.ListColumns(ctx, db.Table{Schema: "public", Name: "list_columns_array_demo"})
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

	ddl, err := database.TableDDL(ctx, db.Table{Schema: "public", Name: "Album"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Contains(t, ddl, "CREATE TABLE \"public\".\"Album\" (")
	assert.Contains(t, ddl, "\"AlbumId\" int4 NOT NULL")
	assert.Contains(t, ddl, `CONSTRAINT "PK_Album" PRIMARY KEY ("AlbumId")`)
	assert.Contains(t, ddl, `CONSTRAINT "FK_AlbumArtistId" FOREIGN KEY ("ArtistId") REFERENCES public."Artist"("ArtistId")`)
	assert.Contains(t, ddl, `CREATE INDEX "IFK_AlbumArtistId" ON public."Album" USING btree ("ArtistId");`)
	assert.NotContains(t, ddl, "ALTER TABLE")
}

func TestTableDDLUsesTableSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	ddl, err := database.TableDDL(ctx, db.Table{Schema: "analytics", Name: "DailyRevenue"})
	if !assert.NoError(t, err, "load DDL from the selected schema") {
		return
	}

	assert.Contains(t, ddl, `CREATE TABLE "analytics"."DailyRevenue" (`)
	assert.Contains(t, ddl, `"RevenueDate" date NOT NULL`)
	assert.Contains(t, ddl, `"InvoiceCount" int4 NOT NULL`)
	assert.Contains(t, ddl, `"Revenue" numeric(10,2) NOT NULL`)
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

	ddl, err := database.TableDDL(ctx, db.Table{Schema: "public", Name: "ddl_review_demo"})
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

	assert.NoError(t, database.Export(ctx, db.Table{Schema: "public", Name: "Album"}, db.ExportTypeCSV))
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

	assert.NoError(t, database.Export(ctx, db.Table{Schema: "public", Name: "Album"}, db.ExportTypeJSON))

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

func TestUpdateRow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	// Snapshot a row for mutation tests
	page, err := database.GetRows(ctx, db.Table{Schema: "public", Name: "Artist"}, db.PageRequest{Offset: 0, Limit: 1})
	if !assert.NoError(t, err) || !assert.NotEmpty(t, page.Rows) {
		return
	}
	row := page.Rows[0]
	originalName, ok := row[1].(string)
	if !assert.True(t, ok) {
		return
	}
	artistID := row[0]

	tests := []struct {
		name         string
		table        db.Table
		setColumns   map[string]any
		whereColumns map[string]any
		wantErr      bool
		errContains  string
	}{
		{
			name:         "empty table name",
			table:        db.Table{},
			setColumns:   map[string]any{"Name": "x"},
			whereColumns: map[string]any{"ArtistId": 1},
			wantErr:      true,
		},
		{
			name:         "empty setColumns",
			table:        db.Table{Schema: "public", Name: "Artist"},
			setColumns:   map[string]any{},
			whereColumns: map[string]any{"ArtistId": 1},
			wantErr:      true,
		},
		{
			name:         "empty whereColumns",
			table:        db.Table{Schema: "public", Name: "Artist"},
			setColumns:   map[string]any{"Name": "x"},
			whereColumns: map[string]any{},
			wantErr:      true,
		},
		{
			name:         "non-matching WHERE",
			table:        db.Table{Schema: "public", Name: "Artist"},
			setColumns:   map[string]any{"Name": "x"},
			whereColumns: map[string]any{"ArtistId": -99999},
			wantErr:      true,
			errContains:  "no row matched",
		},
		{
			name:         "successful update",
			table:        db.Table{Schema: "public", Name: "Artist"},
			setColumns:   map[string]any{"Name": "success_test"},
			whereColumns: map[string]any{"ArtistId": artistID},
			wantErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := database.UpdateRow(ctx, test.table, test.setColumns, test.whereColumns)
			if test.wantErr {
				assert.Error(t, err)
				if test.errContains != "" {
					assert.Contains(t, err.Error(), test.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}

	// Restore original value after success case mutated it
	err = database.UpdateRow(ctx, db.Table{Schema: "public", Name: "Artist"},
		map[string]any{"Name": originalName},
		map[string]any{"ArtistId": artistID},
	)
	assert.NoError(t, err)
}

func TestDeleteRow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	result, err := database.Execute(ctx, `INSERT INTO "Artist" ("ArtistId", "Name")
		SELECT COALESCE(MAX("ArtistId"), 0) + 1, 'delete_row_test' FROM "Artist"
		RETURNING "ArtistId"`)
	if !assert.NoError(t, err, "insert row to delete") || !assert.Len(t, result.Rows, 1) {
		return
	}
	artistID := result.Rows[0][0]
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = database.Execute(cleanupCtx, fmt.Sprintf(`DELETE FROM "Artist" WHERE "ArtistId" = %v`, artistID))
	})

	tests := []struct {
		name         string
		table        db.Table
		whereColumns map[string]any
		wantErr      bool
		errContains  string
	}{
		{
			name:         "empty table name",
			table:        db.Table{},
			whereColumns: map[string]any{"ArtistId": artistID},
			wantErr:      true,
		},
		{
			name:         "empty whereColumns",
			table:        db.Table{Schema: "public", Name: "Artist"},
			whereColumns: map[string]any{},
			wantErr:      true,
		},
		{
			name:         "non-matching WHERE",
			table:        db.Table{Schema: "public", Name: "Artist"},
			whereColumns: map[string]any{"ArtistId": -99999},
			wantErr:      true,
			errContains:  "no row matched",
		},
		{
			name:         "successful delete",
			table:        db.Table{Schema: "public", Name: "Artist"},
			whereColumns: map[string]any{"ArtistId": artistID},
			wantErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := database.DeleteRow(ctx, test.table, test.whereColumns)
			if test.wantErr {
				assert.Error(t, err)
				if test.errContains != "" {
					assert.Contains(t, err.Error(), test.errContains)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
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
			table:        db.Table{Schema: "public", Name: "Album"},
			page:         db.PageRequest{Limit: 2},
			wantColumns:  []string{"AlbumId", "Title", "ArtistId"},
			wantRowCount: 2,
			wantHasMore:  true,
		},
		{
			name:         "page past end",
			table:        db.Table{Schema: "public", Name: "Album"},
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
			table:   db.Table{Schema: "public", Name: "Album"},
			page:    db.PageRequest{Offset: -1, Limit: 1},
			wantErr: true,
		},
		{
			name:    "zero limit",
			table:   db.Table{Schema: "public", Name: "Album"},
			page:    db.PageRequest{},
			wantErr: true,
		},
		{
			name:    "malicious table name is quoted",
			table:   db.Table{Schema: "public", Name: "Album\"; DROP TABLE public.\"Artist\"; --"},
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

	_, err = database.GetRows(ctx, db.Table{Schema: "public", Name: "Album"}, db.PageRequest{Limit: db.MaxPageSize + 1})
	assert.NoError(t, err)
}

func TestGetRowsUsesTableSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Connect(ctx, chinookDSN)
	if !assert.NoError(t, err, "connect to local Compose PostgreSQL") {
		return
	}
	t.Cleanup(database.Close)

	const schema = "schema_object_picker_test"
	_, err = database.Execute(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
	if !assert.NoError(t, err, "remove prior test schema") {
		return
	}
	t.Cleanup(func() {
		_, _ = database.Execute(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
	})
	_, err = database.Execute(ctx, "CREATE SCHEMA "+schema)
	if !assert.NoError(t, err, "create test schema") {
		return
	}
	_, err = database.Execute(ctx, "CREATE TABLE "+schema+".event (id integer)")
	if !assert.NoError(t, err, "create test table") {
		return
	}
	_, err = database.Execute(ctx, "INSERT INTO "+schema+".event (id) VALUES (7)")
	if !assert.NoError(t, err, "insert test row") {
		return
	}

	page, err := database.GetRows(ctx, db.Table{Schema: schema, Name: "event"}, db.PageRequest{Limit: 1})

	if assert.NoError(t, err, "read rows from selected schema") {
		assert.Equal(t, []string{"id"}, page.Columns)
		assert.Len(t, page.Rows, 1)
	}
}
