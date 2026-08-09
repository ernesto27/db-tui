package oracle_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/oracle"
	"github.com/stretchr/testify/assert"
)

const freePDBDSN = "oracle://db_tui:db_tui@127.0.0.1:1522/FREEPDB1"

func TestListTables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)
	assert.Equal(t, "FREEPDB1", database.Name(), "Database.Name()")
	assert.Equal(t, db.EngineOracle, database.Engine(), "Database.Engine()")
	assert.Equal(t, "127.0.0.1", database.Host(), "Database.Host()")

	tables, err := database.ListTables(ctx)
	if !assert.NoError(t, err, "list tables") {
		return
	}

	assert.Equal(t, []db.Table{
		{Name: "CITIES"},
		{Name: "COUNTRIES"},
		{Name: "CURRENCIES"},
		{Name: "CURRENCIES_COUNTRIES"},
		{Name: "REGIONS"},
		{Name: "WITHOUT_PRIMARY_KEY"},
	}, tables, "ListTables()")
}

func TestListViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	views, err := database.ListViews(ctx)
	if !assert.NoError(t, err, "list views") {
		return
	}
	assert.Equal(t, []db.View{
		{Name: "AFRICAN_COUNTRIES"},
		{Name: "AMERICAN_COUNTRIES"},
		{Name: "ASIAN_COUNTRIES"},
		{Name: "CAPITAL_CITIES"},
		{Name: "CAPITAL_DIRECTORY"},
		{Name: "CITY_DIRECTORY"},
		{Name: "CITY_POPULATION"},
		{Name: "COUNTRY_CITY_COUNTS"},
		{Name: "COUNTRY_CURRENCY"},
		{Name: "COUNTRY_DIRECTORY"},
		{Name: "COUNTRY_GEOGRAPHY"},
		{Name: "COUNTRY_POPULATION"},
		{Name: "COUNTRY_TIMEZONES"},
		{Name: "CURRENCY_DIRECTORY"},
		{Name: "EUROPEAN_COUNTRIES"},
		{Name: "LARGE_CITIES"},
		{Name: "OCEANIA_COUNTRIES"},
		{Name: "REGION_COUNTRY_COUNTS"},
		{Name: "REGION_DIRECTORY"},
		{Name: "REGION_POPULATION"},
	}, views, "ListViews()")
}

func TestListMaterializedViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	materializedViews, err := database.ListMaterializedViews(ctx)
	if !assert.NoError(t, err, "list materialized views") {
		return
	}
	assert.Equal(t, []db.MaterializedView{
		{Name: "AFRICA_COUNTRY_MV"},
		{Name: "AMERICAS_COUNTRY_MV"},
		{Name: "ASIA_COUNTRY_MV"},
		{Name: "CAPITAL_CITY_MV"},
		{Name: "CAPITAL_POPULATION_MV"},
		{Name: "CITY_COORDINATES_MV"},
		{Name: "CITY_POPULATION_MV"},
		{Name: "COUNTRY_AREA_MV"},
		{Name: "COUNTRY_CITY_COUNT_MV"},
		{Name: "COUNTRY_COORDINATES_MV"},
		{Name: "COUNTRY_CURRENCY_MV"},
		{Name: "COUNTRY_POPULATION_MV"},
		{Name: "CURRENCY_COUNTRY_COUNT_MV"},
		{Name: "CURRENCY_SYMBOL_MV"},
		{Name: "EUROPE_COUNTRY_MV"},
		{Name: "OCEANIA_COUNTRY_MV"},
		{Name: "REGION_AREA_MV"},
		{Name: "REGION_COUNTRY_COUNT_MV"},
		{Name: "REGION_POPULATION_MV"},
		{Name: "TIMEZONE_COUNTRY_MV"},
	}, materializedViews, "ListMaterializedViews()")
}

func TestListColumns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	columns, err := database.ListColumns(ctx, db.Table{Name: "COUNTRIES"})
	if !assert.NoError(t, err, "list COUNTRIES columns") {
		return
	}

	assert.Equal(t, []db.Column{
		{Name: "COUNTRY_ID", OrdinalPosition: 1, DataType: "VARCHAR2", NotNull: true, IsPrimaryKey: true},
		{Name: "COUNTRY_CODE", OrdinalPosition: 2, DataType: "VARCHAR2", NotNull: true},
		{Name: "NAME", OrdinalPosition: 3, DataType: "VARCHAR2", NotNull: true},
		{Name: "OFFICIAL_NAME", OrdinalPosition: 4, DataType: "VARCHAR2"},
		{Name: "POPULATION", OrdinalPosition: 5, DataType: "NUMBER"},
		{Name: "AREA_SQ_KM", OrdinalPosition: 6, DataType: "NUMBER"},
		{Name: "LATITUDE", OrdinalPosition: 7, DataType: "NUMBER"},
		{Name: "LONGITUDE", OrdinalPosition: 8, DataType: "NUMBER"},
		{Name: "TIMEZONE", OrdinalPosition: 9, DataType: "VARCHAR2"},
		{Name: "REGION_ID", OrdinalPosition: 10, DataType: "VARCHAR2", NotNull: true},
	}, columns, "ListColumns()")
}

func TestListIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	indexes, err := database.ListIndexes(ctx, db.Table{Name: "COUNTRIES"})
	if !assert.NoError(t, err, "list COUNTRIES indexes") {
		return
	}

	assert.Equal(t, []db.IndexColumns{
		{Name: "COUNTRIES_PK", Column: "COUNTRY_ID", Table: "COUNTRIES", AccessMethod: "NORMAL"},
		{Name: "COUNTRIES_REGIONS_FK001", Column: "REGION_ID", Table: "COUNTRIES", AccessMethod: "NORMAL"},
	}, indexes, "ListIndexes()")
}

func TestTableDDL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	ddl, err := database.TableDDL(ctx, db.Table{Name: "COUNTRIES"})
	if !assert.NoError(t, err, "load COUNTRIES DDL") {
		return
	}
	assert.Contains(t, ddl, `CREATE TABLE "DB_TUI"."COUNTRIES"`)
	assert.Contains(t, ddl, `"COUNTRY_ID" VARCHAR2(3) NOT NULL`)
	assert.Contains(t, ddl, `CONSTRAINT "COUNTRIES_PK" PRIMARY KEY ("COUNTRY_ID")`)
}

func TestDumpReturnsUnsupportedError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	assert.EqualError(t, database.Dump(ctx), "Oracle dumps are not supported; use Data Pump outside db-tui")
}

func TestExport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Chdir(t.TempDir())

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	assert.NoError(t, database.Export(ctx, db.Table{Name: "COUNTRIES"}, db.ExportTypeCSV))
}

func TestExportJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Chdir(t.TempDir())

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	if !assert.NoError(t, database.Export(ctx, db.Table{Name: "COUNTRIES"}, db.ExportTypeJSON)) {
		return
	}

	exportFiles, err := filepath.Glob("COUNTRIES_*.json")
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
	assert.NotEmpty(t, document["COUNTRIES"])
}

func TestExportQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Chdir(t.TempDir())

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	if !assert.NoError(t, database.ExportQuery(ctx, "SELECT COUNTRY_ID FROM COUNTRIES")) {
		return
	}

	exportFiles, err := filepath.Glob("query_*.csv")
	if !assert.NoError(t, err, "find generated CSV query export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated CSV query export") {
		return
	}
	assert.Contains(t, string(contents), "COUNTRY_ID\n")
}

func TestUpdateRow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
		return
	}
	t.Cleanup(database.Close)

	page, err := database.GetRows(ctx, db.Table{Name: "COUNTRIES"}, db.PageRequest{Limit: 1})
	if !assert.NoError(t, err) || !assert.NotEmpty(t, page.Rows) {
		return
	}
	row := page.Rows[0]
	originalName := row[2]
	countryID := row[0]

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
			setColumns:   map[string]any{"NAME": "x"},
			whereColumns: map[string]any{"COUNTRY_ID": "AND"},
			wantErr:      true,
		},
		{
			name:         "empty setColumns",
			table:        db.Table{Name: "COUNTRIES"},
			setColumns:   map[string]any{},
			whereColumns: map[string]any{"COUNTRY_ID": countryID},
			wantErr:      true,
		},
		{
			name:       "empty whereColumns",
			table:      db.Table{Name: "COUNTRIES"},
			setColumns: map[string]any{"NAME": "x"},
			wantErr:    true,
		},
		{
			name:         "table without a primary key",
			table:        db.Table{Name: "WITHOUT_PRIMARY_KEY"},
			setColumns:   map[string]any{"NAME": "changed"},
			whereColumns: map[string]any{"ID": 1},
			wantErr:      true,
			errContains:  "table has no primary key",
		},
		{
			name:         "non-primary-key WHERE",
			table:        db.Table{Name: "COUNTRIES"},
			setColumns:   map[string]any{"NAME": "x"},
			whereColumns: map[string]any{"NAME": originalName},
			wantErr:      true,
			errContains:  "complete primary key",
		},
		{
			name:         "non-matching WHERE",
			table:        db.Table{Name: "COUNTRIES"},
			setColumns:   map[string]any{"NAME": "x"},
			whereColumns: map[string]any{"COUNTRY_ID": "XXX"},
			wantErr:      true,
			errContains:  "no row matched",
		},
		{
			name:         "successful update",
			table:        db.Table{Name: "COUNTRIES"},
			setColumns:   map[string]any{"NAME": "success_test"},
			whereColumns: map[string]any{"COUNTRY_ID": countryID},
		},
	}

	updated := false
	t.Cleanup(func() {
		if !updated {
			return
		}
		err := database.UpdateRow(context.Background(), db.Table{Name: "COUNTRIES"}, map[string]any{"NAME": originalName}, map[string]any{"COUNTRY_ID": countryID})
		assert.NoError(t, err)
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := database.UpdateRow(ctx, test.table, test.setColumns, test.whereColumns)
			if test.wantErr {
				assert.Error(t, err)
				if test.errContains != "" {
					assert.ErrorContains(t, err, test.errContains)
				}
				return
			}
			assert.NoError(t, err)
			updated = true
		})
	}
}

func TestGetRows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := oracle.Connect(ctx, freePDBDSN)
	if !assert.NoError(t, err, "connect to local Compose Oracle") {
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
			table:        db.Table{Name: "COUNTRIES"},
			page:         db.PageRequest{Limit: 2},
			wantColumns:  []string{"COUNTRY_ID", "COUNTRY_CODE", "NAME", "OFFICIAL_NAME", "POPULATION", "AREA_SQ_KM", "LATITUDE", "LONGITUDE", "TIMEZONE", "REGION_ID"},
			wantRowCount: 2,
			wantHasMore:  true,
		},
		{
			name:        "page past end",
			table:       db.Table{Name: "COUNTRIES"},
			page:        db.PageRequest{Offset: 10000, Limit: 2},
			wantColumns: []string{"COUNTRY_ID", "COUNTRY_CODE", "NAME", "OFFICIAL_NAME", "POPULATION", "AREA_SQ_KM", "LATITUDE", "LONGITUDE", "TIMEZONE", "REGION_ID"},
		},
		{name: "empty table name", page: db.PageRequest{Limit: 1}, wantErr: true},
		{name: "negative offset", table: db.Table{Name: "COUNTRIES"}, page: db.PageRequest{Offset: -1, Limit: 1}, wantErr: true},
		{name: "zero limit", table: db.Table{Name: "COUNTRIES"}, page: db.PageRequest{}, wantErr: true},
		{name: "limit above maximum", table: db.Table{Name: "COUNTRIES"}, page: db.PageRequest{Limit: db.MaxPageSize + 1}, wantErr: true},
		{name: "malicious table name is quoted", table: db.Table{Name: `COUNTRIES"; DROP TABLE "REGIONS"; --`}, page: db.PageRequest{Limit: 1}, wantErr: true},
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
