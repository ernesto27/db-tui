package sqlite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantName string
		wantErr  string
	}{
		{
			name:     "Employee fixture",
			path:     employeePath(t),
			wantName: "employee.db",
		},
		{
			name:    "missing database file",
			path:    filepath.Join(t.TempDir(), "missing.db"),
			wantErr: "open SQLite database file",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, err := Connect(context.Background(), test.path)
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				assert.Nil(t, database)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantName, database.Name())
			assert.Equal(t, db.EngineSQLite, database.Engine())
			assert.Empty(t, database.Host())
			database.Close()
		})
	}
}

func TestListTables(t *testing.T) {
	database := connectEmployee(t)

	tables, err := database.ListTables(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []db.Table{
		{Name: "department"}, {Name: "dept_emp"}, {Name: "dept_manager"},
		{Name: "employee"}, {Name: "expected_value"}, {Name: "found_value"},
		{Name: "salary"}, {Name: "tchecksum"}, {Name: "title"},
	}, tables)
}

func TestListIndexes(t *testing.T) {
	database := connectEmployee(t)

	indexes, err := database.ListIndexes(context.Background(), db.Table{Name: "department"})

	require.NoError(t, err)
	assert.Equal(t, []db.IndexColumns{
		{Name: "sqlite_autoindex_department_1", Column: "dept_no", Table: "department", AccessMethod: "BTREE"},
		{Name: "sqlite_autoindex_department_2", Column: "dept_name", Table: "department", AccessMethod: "BTREE"},
	}, indexes)
}

func TestGetRows(t *testing.T) {
	database := connectEmployee(t)

	t.Run("returns bounded page", func(t *testing.T) {
		page, err := database.GetRows(context.Background(), db.Table{Name: "employee"}, db.PageRequest{Limit: 2})

		require.NoError(t, err)
		assert.Equal(t, []string{"emp_no", "birth_date", "first_name", "last_name", "gender", "hire_date"}, page.Columns)
		assert.Equal(t, int64(10001), page.Rows[0][0])
		assert.Equal(t, "Georgi", page.Rows[0][2])
		assert.Len(t, page.Rows, 2)
		assert.True(t, page.HasMore)
	})

	t.Run("honors offset", func(t *testing.T) {
		page, err := database.GetRows(context.Background(), db.Table{Name: "employee"}, db.PageRequest{Offset: 1, Limit: 1})

		require.NoError(t, err)
		assert.Equal(t, int64(10002), page.Rows[0][0])
		assert.True(t, page.HasMore)
	})

	t.Run("preserves SQL NULL", func(t *testing.T) {
		isolated := connectSQLiteFile(t, newSQLiteFile(t))
		_, err := isolated.Execute(context.Background(), "CREATE TABLE nullable (value TEXT)")
		require.NoError(t, err)
		_, err = isolated.Execute(context.Background(), "INSERT INTO nullable VALUES (NULL)")
		require.NoError(t, err)
		page, err := isolated.GetRows(context.Background(), db.Table{Name: "nullable"}, db.PageRequest{Limit: 1})

		require.NoError(t, err)
		assert.Nil(t, page.Rows[0][0])
	})

	t.Run("quotes table name", func(t *testing.T) {
		_, err := database.GetRows(context.Background(), db.Table{Name: `employee"; DROP TABLE department; --`}, db.PageRequest{Limit: 1})

		assert.ErrorContains(t, err, "query SQLite rows")
	})

	for _, test := range []struct {
		name  string
		table db.Table
		page  db.PageRequest
		want  string
	}{
		{name: "empty table", page: db.PageRequest{Limit: 1}, want: "table name is required"},
		{name: "negative offset", table: db.Table{Name: "employee"}, page: db.PageRequest{Offset: -1, Limit: 1}, want: "page offset cannot be negative"},
		{name: "zero limit", table: db.Table{Name: "employee"}, want: "page limit must be between 1 and 100"},
		{name: "large limit", table: db.Table{Name: "employee"}, page: db.PageRequest{Limit: db.MaxPageSize + 1}, want: "page limit must be between 1 and 100"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := database.GetRows(context.Background(), test.table, test.page)
			assert.EqualError(t, err, test.want)
		})
	}
}

func TestTableDDL(t *testing.T) {
	database := connectEmployee(t)

	ddl, err := database.TableDDL(context.Background(), db.Table{Name: "employee"})

	require.NoError(t, err)
	assert.Contains(t, ddl, "CREATE TABLE employee")
	assert.True(t, strings.HasSuffix(ddl, ";"))

	_, err = database.TableDDL(context.Background(), db.Table{Name: "missing"})
	assert.ErrorContains(t, err, "lookup SQLite table DDL")
}

func TestExecute(t *testing.T) {
	database := connectEmployee(t)

	result, err := database.Execute(context.Background(), "SELECT emp_no FROM employee")

	require.NoError(t, err)
	assert.Equal(t, []string{"emp_no"}, result.Columns)
	assert.Len(t, result.Rows, db.MaxQueryResultRows)
	assert.Equal(t, "SELECT", result.CommandTag)

	isolated := connectSQLiteFile(t, newSQLiteFile(t))
	result, err = isolated.Execute(context.Background(), "CREATE TABLE example (id INTEGER)")
	require.NoError(t, err)
	assert.Empty(t, result.Columns)
	assert.Empty(t, result.Rows)
	assert.Equal(t, "CREATE TABLE", result.CommandTag)
}

func TestExport(t *testing.T) {
	database := connectEmployee(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.Export(context.Background(), db.Table{Name: "employee"}, db.ExportTypeCSV))
	csvFiles, err := filepath.Glob("employee_*.csv")
	require.NoError(t, err)
	require.Len(t, csvFiles, 1)
	contents, err := os.ReadFile(csvFiles[0])
	require.NoError(t, err)
	assert.Contains(t, string(contents), "emp_no,birth_date,first_name,last_name,gender,hire_date")

	require.NoError(t, database.Export(context.Background(), db.Table{Name: "employee"}, db.ExportTypeJSON))
	jsonFiles, err := filepath.Glob("employee_*.json")
	require.NoError(t, err)
	require.Len(t, jsonFiles, 1)
	var document map[string][]map[string]any
	contents, err = os.ReadFile(jsonFiles[0])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(contents, &document))
	assert.Equal(t, "Georgi", document["employee"][0]["first_name"])
}

func TestExportQuery(t *testing.T) {
	database := connectEmployee(t)
	t.Chdir(t.TempDir())

	require.NoError(t, database.ExportQuery(context.Background(), "SELECT emp_no FROM employee"))
	files, err := filepath.Glob("query_*.csv")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.EqualError(t, database.ExportQuery(context.Background(), "UPDATE employee SET first_name = 'x'"), "only SELECT queries can be exported")
}

func TestDump(t *testing.T) {
	database := connectEmployee(t)
	t.Chdir(t.TempDir())

	bin := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "args")
	shim := filepath.Join(bin, "sqlite3")
	require.NoError(t, os.WriteFile(shim, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$TEST_SQLITE_ARGS\"\nprintf 'BEGIN TRANSACTION;\\n'\n"), 0o755))
	t.Setenv("PATH", bin)
	t.Setenv("TEST_SQLITE_ARGS", argsPath)

	require.NoError(t, database.Dump(context.Background()))
	args, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, employeePath(t)+"\n.dump\n", string(args))
	files, err := filepath.Glob("employee.db_*.sql")
	require.NoError(t, err)
	require.Len(t, files, 1)
	contents, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, "BEGIN TRANSACTION;\n", string(contents))
}

func connectEmployee(t *testing.T) db.Database {
	t.Helper()
	return connectSQLiteFile(t, employeePath(t))
}

func connectSQLiteFile(t *testing.T, path string) db.Database {
	t.Helper()
	database, err := Connect(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(database.Close)
	return database
}

func newSQLiteFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	return path
}

func employeePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "docker", "sqlite", "employee.db")
}
