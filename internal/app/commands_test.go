package app

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestLoadTables(t *testing.T) {
	wantErr := errors.New("list failed")
	database := &fakeDatabase{
		tables:    []db.Table{{Name: "Album"}},
		tablesErr: wantErr,
	}

	message, ok := loadTables(database, 7)().(tablesLoadedMsg)
	require.True(t, ok)

	assert.Equal(t, 1, database.listTablesCalls)
	assert.True(t, database.listTablesDeadline)
	assert.Equal(t, database.tables, message.tables)
	assert.ErrorIs(t, message.err, wantErr)
	assert.Equal(t, uint64(7), message.session)
}

func TestLoadRows(t *testing.T) {
	wantErr := errors.New("rows failed")
	tests := []struct {
		name        string
		database    *fakeDatabase
		table       db.Table
		offset      int
		selectedRow int
		session     uint64
		wantPage    db.RowPage
		wantErr     error
	}{
		{
			name: "success",
			database: &fakeDatabase{page: db.RowPage{
				Columns: []string{"TrackId"},
				Rows:    [][]any{{1}},
				HasMore: true,
			}},
			table:       db.Table{Name: "Track"},
			offset:      200,
			selectedRow: 4,
			session:     9,
			wantPage: db.RowPage{
				Columns: []string{"TrackId"},
				Rows:    [][]any{{1}},
				HasMore: true,
			},
		},
		{
			name:        "error",
			database:    &fakeDatabase{pageErr: wantErr},
			table:       db.Table{Name: "Track"},
			offset:      0,
			selectedRow: 0,
			session:     2,
			wantErr:     wantErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, ok := loadRows(
				test.database,
				test.table,
				test.offset,
				test.selectedRow,
				test.session,
			)().(rowsLoadedMsg)
			require.True(t, ok)

			assert.Equal(t, 1, test.database.getRowsCalls)
			assert.Equal(t, test.table, test.database.getRowsTable)
			assert.Equal(t, db.PageRequest{
				Offset: test.offset,
				Limit:  rowPageSize,
			}, test.database.getRowsRequest)
			assert.True(t, test.database.getRowsDeadline)
			assert.Equal(t, test.wantPage, message.page)
			assert.Equal(t, test.table.Name, message.tableName)
			assert.Equal(t, test.offset, message.offset)
			assert.Equal(t, test.selectedRow, message.selectedRow)
			assert.Equal(t, test.session, message.session)
			assert.ErrorIs(t, message.err, test.wantErr)
		})
	}
}

func TestLoadTableDDL(t *testing.T) {
	wantErr := errors.New("DDL failed")
	tests := []struct {
		name     string
		database *fakeDatabase
		wantSQL  string
		wantErr  error
	}{
		{name: "success", database: &fakeDatabase{ddl: "CREATE TABLE public.\"Album\" ();"}, wantSQL: "CREATE TABLE public.\"Album\" ();"},
		{name: "error", database: &fakeDatabase{ddlErr: wantErr}, wantErr: wantErr},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, ok := loadTableDDL(test.database, db.Table{Name: "Album"}, 7, 3)().(tableDDLLoadedMsg)
			require.True(t, ok)

			assert.Equal(t, 1, test.database.tableDDLCalls)
			assert.Equal(t, db.Table{Name: "Album"}, test.database.tableDDLTable)
			assert.True(t, test.database.tableDDLDeadline)
			assert.Equal(t, "Album", message.tableName)
			assert.Equal(t, test.wantSQL, message.sql)
			assert.Equal(t, uint64(7), message.session)
			assert.Equal(t, uint64(3), message.request)
			assert.ErrorIs(t, message.err, test.wantErr)
		})
	}
}

func TestLoadColumns(t *testing.T) {
	wantColumns := []db.Column{{Name: "AlbumId", OrdinalPosition: 1, DataType: "int4", NotNull: true}}
	database := &fakeDatabase{columns: wantColumns}
	table := db.Table{Name: "Album"}

	message, ok := loadColumns(database, table, 7, 3)().(columnsLoadedMsg)
	require.True(t, ok)

	assert.Equal(t, 1, database.listColumnsCalls)
	assert.Equal(t, table, database.listColumnsTable)
	assert.True(t, database.listColumnsDeadline)
	assert.Equal(t, wantColumns, message.columns)
	assert.Equal(t, "Album", message.tableName)
	assert.Equal(t, uint64(7), message.session)
	assert.Equal(t, uint64(3), message.request)
}

func TestExecuteQuery(t *testing.T) {
	wantErr := errors.New("query failed")
	tests := []struct {
		name       string
		database   *fakeDatabase
		sql        string
		session    uint64
		request    uint64
		wantResult db.QueryResult
		wantErr    error
	}{
		{
			name: "success",
			database: &fakeDatabase{queryResult: db.QueryResult{
				Columns:    []string{"count"},
				Rows:       [][]any{{3}},
				CommandTag: "SELECT 1",
			}},
			sql:     "SELECT count(*)",
			session: 4,
			request: 12,
			wantResult: db.QueryResult{
				Columns:    []string{"count"},
				Rows:       [][]any{{3}},
				CommandTag: "SELECT 1",
			},
		},
		{
			name:     "error",
			database: &fakeDatabase{queryErr: wantErr},
			sql:      "broken",
			session:  4,
			request:  12,
			wantErr:  wantErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, ok := executeQuery(
				test.database,
				test.sql,
				test.session,
				test.request,
			)().(queryFinishedMsg)
			require.True(t, ok)

			assert.Equal(t, 1, test.database.executeCalls)
			assert.Equal(t, test.sql, test.database.executedSQL)
			assert.True(t, test.database.executeDeadline)
			assert.Equal(t, test.wantResult, message.result)
			assert.Equal(t, test.session, message.session)
			assert.Equal(t, test.request, message.request)
			assert.GreaterOrEqual(t, message.elapsed, time.Duration(0))
			assert.ErrorIs(t, message.err, test.wantErr)
		})
	}
}

func TestDumpDatabase(t *testing.T) {
	wantErr := errors.New("dump failed")
	database := &fakeDatabase{dumpErr: wantErr}

	message, ok := dumpDatabase(database, 15)().(dumpFinishedMsg)
	require.True(t, ok)

	assert.Equal(t, 1, database.dumpCalls)
	assert.True(t, database.dumpDeadline)
	assert.Equal(t, uint64(15), message.session)
	assert.ErrorIs(t, message.err, wantErr)
}

func TestExportQuery(t *testing.T) {
	wantErr := errors.New("export failed")
	database := &fakeDatabase{exportQueryErr: wantErr}

	message, ok := exportQuery(database, "SELECT * FROM Album", 15)().(exportFinishedMsg)
	require.True(t, ok)

	assert.Equal(t, 1, database.exportQueryCalls)
	assert.Equal(t, "SELECT * FROM Album", database.exportedQuery)
	assert.True(t, database.exportQueryDeadline)
	assert.Equal(t, uint64(15), message.session)
	assert.ErrorIs(t, message.err, wantErr)
}

func TestExportTable(t *testing.T) {
	wantErr := errors.New("export failed")

	for _, format := range []string{db.ExportTypeCSV, db.ExportTypeJSON} {
		t.Run(format, func(t *testing.T) {
			database := &fakeDatabase{exportErr: wantErr}

			message, ok := exportTable(database, db.Table{Name: "Album"}, format, 15)().(exportFinishedMsg)
			require.True(t, ok)

			assert.Equal(t, 1, database.exportCalls)
			assert.Equal(t, db.Table{Name: "Album"}, database.exportTable)
			assert.Equal(t, format, database.exportType)
			assert.True(t, database.exportDeadline)
			assert.Equal(t, uint64(15), message.session)
			assert.ErrorIs(t, message.err, wantErr)
		})
	}
}
