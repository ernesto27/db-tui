package app

import (
	"context"
	"errors"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	tableLoadTimeout      = 5 * time.Second
	dumpTimeout           = 30 * time.Minute
	queryExecutionTimeout = 20 * time.Minute
)

type tablesLoadedMsg struct {
	tables  []db.Table
	session uint64
	err     error
}

type viewsLoadedMsg struct {
	views   []db.View
	session uint64
	err     error
}

type rowsLoadedMsg struct {
	relation    navigatorItem
	offset      int
	selectedRow int
	page        db.RowPage
	session     uint64
	request     uint64
	err         error
}

type tableDDLLoadedMsg struct {
	tableName string
	sql       string
	session   uint64
	request   uint64
	err       error
}

type columnsLoadedMsg struct {
	tableName string
	columns   []db.Column
	session   uint64
	request   uint64
	err       error
}

type indexesLoadedMsg struct {
	tableName string
	indexes   []db.IndexColumns
	session   uint64
	request   uint64
	err       error
}

type queryFinishedMsg struct {
	result  db.QueryResult
	session uint64
	request uint64
	err     error
	elapsed time.Duration
}

type materializedViewsLoadedMsg struct {
	materializedViews []db.MaterializedView
	session           uint64
	err               error
}

func loadSQLScripts(sqlScripts ListSqlScript, connectionName string, request uint64) tea.Cmd {
	return func() tea.Msg {
		scripts, err := sqlScripts.getList(connectionName)
		if errors.Is(err, os.ErrNotExist) {
			scripts = []SqlScript{}
			err = nil
		}
		return sqlScriptsLoadedMsg{
			connectionName: connectionName,
			request:        request,
			scripts:        scripts,
			err:            err,
		}
	}
}

func saveSQLScript(sqlScripts ListSqlScript, connectionName, fileName, content string, session, request uint64) tea.Cmd {
	return func() tea.Msg {
		var err error
		if fileName == "" {
			err = sqlScripts.createByConnection(connectionName, content)
		} else {
			err = sqlScripts.editByConnection(connectionName, fileName, content)
		}
		return sqlScriptSavedMsg{session: session, request: request, err: err}
	}
}

func loadTables(database db.Database, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		tables, err := database.ListTables(ctx)
		return tablesLoadedMsg{tables: tables, session: session, err: err}
	}
}

func loadViews(database db.Database, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		views, err := database.ListViews(ctx)
		return viewsLoadedMsg{views: views, session: session, err: err}
	}
}

func loadRows(database db.Database, relation navigatorItem, offset, selectedRow, maxPageSize int, session, request uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		page, err := database.GetRows(ctx, relation.rowSource(), db.PageRequest{
			Offset: offset,
			Limit:  maxPageSize,
		})
		return rowsLoadedMsg{
			relation:    relation,
			offset:      offset,
			selectedRow: selectedRow,
			page:        page,
			session:     session,
			request:     request,
			err:         err,
		}
	}
}

func saveSettings(appConfig config.Config, maxPageSize int) tea.Cmd {
	return func() tea.Msg {
		appConfig.MaxPageSize = maxPageSize
		return settingsSavedMsg{maxPageSize: maxPageSize, err: appConfig.Save()}
	}
}

func loadTableDDL(database db.Database, table db.Table, session, request uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		sql, err := database.TableDDL(ctx, table)
		return tableDDLLoadedMsg{
			tableName: table.Name,
			sql:       sql,
			session:   session,
			request:   request,
			err:       err,
		}
	}
}

func loadColumns(database db.Database, table db.Table, session, request uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		columns, err := database.ListColumns(ctx, table)
		return columnsLoadedMsg{
			tableName: table.Name,
			columns:   columns,
			session:   session,
			request:   request,
			err:       err,
		}
	}
}

func loadIndexes(database db.Database, table db.Table, session, request uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		indexes, err := database.ListIndexes(ctx, table)
		return indexesLoadedMsg{
			tableName: table.Name,
			indexes:   indexes,
			session:   session,
			request:   request,
			err:       err,
		}
	}
}

func loadMaterializedViews(database db.Database, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		materializedViews, err := database.ListMaterializedViews(ctx)

		return materializedViewsLoadedMsg{
			materializedViews: materializedViews,
			session:           session,
			err:               err,
		}
	}
}

func executeQuery(database db.Database, sql string, session, request uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), queryExecutionTimeout)
		defer cancel()

		started := time.Now()
		result, err := database.Execute(ctx, sql)
		return queryFinishedMsg{
			result:  result,
			session: session,
			request: request,
			err:     err,
			elapsed: time.Since(started),
		}
	}
}

type dumpFinishedMsg struct {
	session uint64
	err     error
}

type exportFinishedMsg struct {
	session uint64
	err     error
}

func dumpDatabase(database db.Database, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), dumpTimeout)
		defer cancel()

		err := database.Dump(ctx)
		return dumpFinishedMsg{
			session: session,
			err:     err,
		}
	}
}

func exportTable(database db.Database, table db.Table, format string, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), dumpTimeout)
		defer cancel()

		err := database.Export(ctx, table, format)
		return exportFinishedMsg{
			session: session,
			err:     err,
		}
	}
}

func exportQuery(database db.Database, sql string, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), dumpTimeout)
		defer cancel()

		err := database.ExportQuery(ctx, sql)
		return exportFinishedMsg{
			session: session,
			err:     err,
		}
	}
}

type editRowCancelMsg struct{}

type editRowSaveMsg struct {
	table        db.Table
	setColumns   map[string]any
	whereColumns map[string]any
}

type editRowSavedMsg struct {
	session uint64
	err     error
}

type editRowColumnsLoadedMsg struct {
	table   db.Table
	columns []db.Column
	row     []any
	session uint64
	err     error
}

func loadEditRowColumns(database db.Database, table db.Table, row []any, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()
		columns, err := database.ListColumns(ctx, table)
		return editRowColumnsLoadedMsg{
			table:   table,
			columns: columns,
			row:     row,
			session: session,
			err:     err,
		}
	}
}

func saveRowEdit(database db.Database, table db.Table, setColumns, whereColumns map[string]any, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()
		err := database.UpdateRow(ctx, table, setColumns, whereColumns)
		return editRowSavedMsg{session: session, err: err}
	}
}

type deleteRowCancelMsg struct{}

type deleteRowConfirmMsg struct {
	table        db.Table
	whereColumns map[string]any
}

type deleteRowFinishedMsg struct {
	session uint64
	err     error
}

func deleteRow(
	database db.Database,
	table db.Table,
	whereColumns map[string]any,
	session uint64,
) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		err := database.DeleteRow(ctx, table, whereColumns)
		return deleteRowFinishedMsg{
			session: session,
			err:     err,
		}
	}
}
