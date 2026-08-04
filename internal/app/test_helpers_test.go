package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/db"
)

var appTestHome string

// TestMain keeps app tests from reading or writing the user's real db-tui config.
func TestMain(m *testing.M) {
	var err error
	appTestHome, err = os.MkdirTemp("", "db-tui-app-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create app test home:", err)
		os.Exit(1)
	}
	if err := os.Setenv("HOME", appTestHome); err != nil {
		_ = os.RemoveAll(appTestHome)
		fmt.Fprintln(os.Stderr, "set app test home:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join(appTestHome, ".config", "db-tui"), 0o700); err != nil {
		_ = os.RemoveAll(appTestHome)
		fmt.Fprintln(os.Stderr, "create app test config directory:", err)
		os.Exit(1)
	}

	code := m.Run()
	if err := os.RemoveAll(appTestHome); err != nil && code == 0 {
		fmt.Fprintln(os.Stderr, "remove app test home:", err)
		code = 1
	}
	os.Exit(code)
}

type fakeDatabase struct {
	name   string
	engine string
	host   string

	tables     []db.Table
	tablesErr  error
	columns    []db.Column
	columnsErr error
	indexes    []db.IndexColumns
	indexesErr error

	page    db.RowPage
	pageErr error

	queryResult db.QueryResult
	queryErr    error
	ddl         string
	ddlErr      error

	dumpErr        error
	exportErr      error
	exportQueryErr error

	listTablesCalls     int
	listTablesDeadline  bool
	listColumnsCalls    int
	listColumnsTable    db.Table
	listColumnsDeadline bool
	listIndexesCalls    int
	listIndexesTable    db.Table
	listIndexesDeadline bool
	getRowsCalls        int
	getRowsTable        db.Table
	getRowsRequest      db.PageRequest
	getRowsDeadline     bool
	executeCalls        int
	executedSQL         string
	executeDeadline     bool
	tableDDLCalls       int
	tableDDLTable       db.Table
	tableDDLDeadline    bool
	dumpCalls           int
	dumpDeadline        bool
	exportCalls         int
	exportTable         db.Table
	exportType          string
	exportDeadline      bool
	exportQueryCalls    int
	exportedQuery       string
	exportQueryDeadline bool
	closeCalls          int
}

func (f *fakeDatabase) Name() string {
	return f.name
}

func (f *fakeDatabase) Engine() string {
	return f.engine
}

func (f *fakeDatabase) Host() string {
	return f.host
}

func (f *fakeDatabase) ListTables(ctx context.Context) ([]db.Table, error) {
	f.listTablesCalls++
	_, f.listTablesDeadline = ctx.Deadline()
	return f.tables, f.tablesErr
}

func (f *fakeDatabase) ListColumns(ctx context.Context, table db.Table) ([]db.Column, error) {
	f.listColumnsCalls++
	f.listColumnsTable = table
	_, f.listColumnsDeadline = ctx.Deadline()
	return f.columns, f.columnsErr
}

func (f *fakeDatabase) ListIndexes(ctx context.Context, table db.Table) ([]db.IndexColumns, error) {
	f.listIndexesCalls++
	f.listIndexesTable = table
	_, f.listIndexesDeadline = ctx.Deadline()
	return f.indexes, f.indexesErr
}

func (f *fakeDatabase) GetRows(
	ctx context.Context,
	table db.Table,
	request db.PageRequest,
) (db.RowPage, error) {
	f.getRowsCalls++
	f.getRowsTable = table
	f.getRowsRequest = request
	_, f.getRowsDeadline = ctx.Deadline()
	return f.page, f.pageErr
}

func (f *fakeDatabase) Execute(
	ctx context.Context,
	sql string,
) (db.QueryResult, error) {
	f.executeCalls++
	f.executedSQL = sql
	_, f.executeDeadline = ctx.Deadline()
	return f.queryResult, f.queryErr
}

func (f *fakeDatabase) TableDDL(ctx context.Context, table db.Table) (string, error) {
	f.tableDDLCalls++
	f.tableDDLTable = table
	_, f.tableDDLDeadline = ctx.Deadline()
	return f.ddl, f.ddlErr
}

func (f *fakeDatabase) Dump(ctx context.Context) error {
	f.dumpCalls++
	_, f.dumpDeadline = ctx.Deadline()
	return f.dumpErr
}

func (f *fakeDatabase) Export(ctx context.Context, table db.Table, typeVal string) error {
	f.exportCalls++
	f.exportTable = table
	f.exportType = typeVal
	_, f.exportDeadline = ctx.Deadline()
	return f.exportErr
}

func (f *fakeDatabase) ExportQuery(ctx context.Context, query string) error {
	f.exportQueryCalls++
	f.exportedQuery = query
	_, f.exportQueryDeadline = ctx.Deadline()
	return f.exportQueryErr
}

func (f *fakeDatabase) Close() {
	f.closeCalls++
}

func keyPress(code rune, text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{
		Code: code,
		Text: text,
		Mod:  mod,
	}
}

func updateModel(t *testing.T, model Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()

	next, command := model.Update(msg)
	updated, ok := next.(Model)
	require.True(t, ok, "Update returned %T instead of app.Model", next)
	return updated, command
}
