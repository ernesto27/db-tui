package app

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type fakeDatabase struct {
	name string

	tables    []db.Table
	tablesErr error

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

func (f *fakeDatabase) ListTables(ctx context.Context) ([]db.Table, error) {
	f.listTablesCalls++
	_, f.listTablesDeadline = ctx.Deadline()
	return f.tables, f.tablesErr
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
