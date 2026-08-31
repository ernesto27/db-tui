package mysql

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestEnsureTrailingSemicolon(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "adds semicolon", input: "CREATE TABLE `city` (id int)", want: "CREATE TABLE `city` (id int);"},
		{name: "keeps semicolon", input: "CREATE TABLE `city` (id int);", want: "CREATE TABLE `city` (id int);"},
		{name: "preserves formatting", input: "CREATE TABLE `city` (\n  id int\n)\n", want: "CREATE TABLE `city` (\n  id int\n);"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, ensureTrailingSemicolon(test.input))
		})
	}

}

func TestListTables(t *testing.T) {
	database := connectWorld(t)

	tables, err := database.ListTables(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, []db.Table{
		{Name: "city"},
		{Name: "country"},
		{Name: "countrylanguage"},
		{Name: "without_primary_key"},
	}, tables)
}

func TestListViews(t *testing.T) {
	database := connectWorld(t)

	views, err := database.ListViews(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, []db.View{
		{Name: "CityCountryDirectory"},
		{Name: "CityDirectory"},
		{Name: "CityDistrictSummary"},
		{Name: "CityPopulationSummary"},
		{Name: "CountryCapitalDirectory"},
		{Name: "CountryDirectory"},
		{Name: "CountryLanguageDetail"},
		{Name: "CountryLanguageSummary"},
		{Name: "CountryPopulationByContinent"},
		{Name: "CountryPopulationByRegion"},
		{Name: "CountryPopulationDensity"},
		{Name: "CountrySurfaceAreaByContinent"},
		{Name: "GNPByContinent"},
		{Name: "GovernmentFormSummary"},
		{Name: "IndependentCountrySummary"},
		{Name: "LargestCities"},
		{Name: "LargestCountries"},
		{Name: "LifeExpectancySummary"},
		{Name: "OfficialLanguageDetail"},
		{Name: "OfficialLanguageSummary"},
	}, views)
}

func TestListFunctions(t *testing.T) {
	database := connectWorld(t)

	functions, err := database.ListFunctions(context.Background(), "world")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"country_capital_name",
		"country_city_count",
		"country_official_language_count",
		"country_population",
		"country_population_density",
	}, mysqlFunctionNames(functions))

	expectedMetadata := map[string]struct {
		arguments  string
		returnType string
	}{
		"country_capital_name":            {arguments: "IN country_code char(3)", returnType: "varchar(35)"},
		"country_city_count":              {arguments: "IN country_code char(3)", returnType: "int"},
		"country_official_language_count": {arguments: "IN country_code char(3)", returnType: "int"},
		"country_population":              {arguments: "IN country_code char(3)", returnType: "int"},
		"country_population_density":      {arguments: "IN country_code char(3)", returnType: "decimal(14,2)"},
	}
	for _, function := range functions {
		expected, ok := expectedMetadata[function.Name]
		if !assert.True(t, ok, "unexpected function %q", function.Name) {
			continue
		}
		assert.Equal(t, expected.arguments, function.Arguments, "%s arguments", function.Name)
		assert.Equal(t, expected.returnType, function.ReturnType, "%s return type", function.Name)
		assert.Equal(t, "SQL", function.Language, "%s language", function.Name)
	}
}

func mysqlFunctionNames(functions []db.FunctionColumns) []string {
	names := make([]string, len(functions))
	for index, function := range functions {
		names[index] = function.Name
	}
	return names
}

func TestListColumns(t *testing.T) {
	database := connectWorld(t)

	columns, err := database.ListColumns(context.Background(), db.Table{Name: "city"})

	require.NoError(t, err)
	assert.Equal(t, []db.Column{
		{Name: "ID", OrdinalPosition: 1, DataType: "int", Identity: "AUTO_INCREMENT", NotNull: true, IsPrimaryKey: true},
		{Name: "Name", OrdinalPosition: 2, DataType: "char(35)", Collation: "default", NotNull: true},
		{Name: "CountryCode", OrdinalPosition: 3, DataType: "char(3)", Collation: "default", NotNull: true},
		{Name: "District", OrdinalPosition: 4, DataType: "char(20)", Collation: "default", NotNull: true},
		{Name: "Population", OrdinalPosition: 5, DataType: "int", NotNull: true, Default: "0"},
	}, columns)
}

func TestListIndexes(t *testing.T) {
	database := connectWorld(t)

	indexes, err := database.ListIndexes(context.Background(), db.Table{Name: "city"})

	require.NoError(t, err)
	assert.Equal(t, []db.IndexColumns{
		{Name: "CountryCode", Column: "CountryCode", Table: "city", AccessMethod: "BTREE"},
		{Name: "PRIMARY", Column: "ID", Table: "city", AccessMethod: "BTREE"},
	}, indexes)
}

func TestTableDDL(t *testing.T) {
	database := connectWorld(t)

	ddl, err := database.TableDDL(context.Background(), db.Table{Name: "city"})

	require.NoError(t, err)
	assert.Contains(t, ddl, "CREATE TABLE `city`")
	assert.True(t, strings.HasSuffix(ddl, ";"))
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
		{name: "zero limit", table: db.Table{Name: "city"}, error: "page limit must be positive"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := database.GetRows(context.Background(), test.table, test.page)
			assert.EqualError(t, err, test.error)
		})
	}

	_, err := database.GetRows(context.Background(), db.Table{Name: "city"}, db.PageRequest{Limit: db.MaxPageSize + 1})
	assert.NoError(t, err)
}

func TestExport(t *testing.T) {
	database := connectWorld(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.Export(context.Background(), db.Table{Name: "city"}, db.ExportTypeCSV))

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

func TestExportJSON(t *testing.T) {
	database := connectWorld(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.Export(context.Background(), db.Table{Name: "city"}, db.ExportTypeJSON))

	exportFiles, err := filepath.Glob("city_*.json")
	if !assert.NoError(t, err, "find generated JSON export") || !assert.Len(t, exportFiles, 1) {
		return
	}

	contents, err := os.ReadFile(exportFiles[0])
	if !assert.NoError(t, err, "read generated JSON export") {
		return
	}
	var document map[string][]map[string]any
	require.NoError(t, json.Unmarshal(contents, &document))
	assert.NotEmpty(t, document["city"])
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
		assert.Len(t, result.Rows, db.MaxPageSize)
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

	t.Run("cancels a running query", func(t *testing.T) {
		queryCtx, cancelQuery := context.WithCancel(context.Background())
		defer cancelQuery()
		cancelTimer := time.AfterFunc(250*time.Millisecond, cancelQuery)
		defer cancelTimer.Stop()

		started := time.Now()
		_, err := database.Execute(queryCtx, "SELECT SLEEP(10)")

		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, time.Since(started), 2*time.Second, "canceled query should not wait for SLEEP")
	})
}

func TestUpdateRow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database := connectWorld(t)

	page, err := database.GetRows(ctx, db.Table{Name: "city"}, db.PageRequest{Limit: 1})
	if !assert.NoError(t, err) || !assert.NotEmpty(t, page.Rows) {
		return
	}
	row := page.Rows[0]
	originalName, ok := row[1].(string)
	if !assert.True(t, ok) {
		return
	}
	cityID := row[0]

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
			whereColumns: map[string]any{"ID": 1},
			wantErr:      true,
		},
		{
			name:         "empty setColumns",
			table:        db.Table{Name: "city"},
			setColumns:   map[string]any{},
			whereColumns: map[string]any{"ID": 1},
			wantErr:      true,
		},
		{
			name:       "empty whereColumns",
			table:      db.Table{Name: "city"},
			setColumns: map[string]any{"Name": "x"},
			wantErr:    true,
		},
		{
			name:         "table without a primary key",
			table:        db.Table{Name: "without_primary_key"},
			setColumns:   map[string]any{"name": "changed"},
			whereColumns: map[string]any{"id": 1},
			wantErr:      true,
			errContains:  "table has no primary key",
		},
		{
			name:         "non-primary-key WHERE",
			table:        db.Table{Name: "city"},
			setColumns:   map[string]any{"Name": "x"},
			whereColumns: map[string]any{"Name": originalName},
			wantErr:      true,
			errContains:  "complete primary key",
		},
		{
			name:         "non-matching WHERE",
			table:        db.Table{Name: "city"},
			setColumns:   map[string]any{"Name": "x"},
			whereColumns: map[string]any{"ID": -99999},
			wantErr:      true,
			errContains:  "no row matched",
		},
		{
			name:         "successful update",
			table:        db.Table{Name: "city"},
			setColumns:   map[string]any{"Name": "success_test"},
			whereColumns: map[string]any{"ID": cityID},
		},
	}

	updated := false
	t.Cleanup(func() {
		if !updated {
			return
		}
		err := database.UpdateRow(context.Background(), db.Table{Name: "city"}, map[string]any{"Name": originalName}, map[string]any{"ID": cityID})
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

	keylessPage, err := database.GetRows(ctx, db.Table{Name: "without_primary_key"}, db.PageRequest{Limit: 1})
	require.NoError(t, err)
	require.Len(t, keylessPage.Rows, 1)
	assert.Equal(t, "unchanged", keylessPage.Rows[0][1])
}

func TestDeleteRow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database := connectWorld(t)
	mysqlDatabase, ok := database.(*mysqlDatabase)
	require.True(t, ok)

	result, err := mysqlDatabase.database.ExecContext(ctx,
		"INSERT INTO city (Name, CountryCode, District, Population) VALUES (?, ?, ?, ?)",
		"delete_row_test", "ARG", "test", 1,
	)
	require.NoError(t, err)
	cityID, err := result.LastInsertId()
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := mysqlDatabase.database.ExecContext(context.Background(), "DELETE FROM city WHERE ID = ?", cityID)
		assert.NoError(t, err)
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
			whereColumns: map[string]any{"ID": cityID},
			wantErr:      true,
		},
		{
			name:         "empty whereColumns",
			table:        db.Table{Name: "city"},
			whereColumns: map[string]any{},
			wantErr:      true,
		},
		{
			name:         "non-primary-key WHERE",
			table:        db.Table{Name: "city"},
			whereColumns: map[string]any{"Name": "delete_row_test"},
			wantErr:      true,
			errContains:  "complete primary key",
		},
		{
			name:         "non-matching WHERE",
			table:        db.Table{Name: "city"},
			whereColumns: map[string]any{"ID": -99999},
			wantErr:      true,
			errContains:  "no row matched",
		},
		{
			name:         "successful delete",
			table:        db.Table{Name: "city"},
			whereColumns: map[string]any{"ID": cityID},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := database.DeleteRow(ctx, test.table, test.whereColumns)
			if test.wantErr {
				assert.Error(t, err)
				if test.errContains != "" {
					assert.ErrorContains(t, err, test.errContains)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
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
	assert.Equal(t, db.EngineMySQL, database.Engine())
	assert.Equal(t, "127.0.0.1", database.Host())
	t.Cleanup(func() {
		database.Close()
	})
	return database
}
