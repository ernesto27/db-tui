package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	tableLoadTimeout      = 5 * time.Second
	dumpTimeout           = 30 * time.Minute
	queryExecutionTimeout = 20 * time.Minute
	rowPageSize           = 100
)

type tablesLoadedMsg struct {
	tables  []db.Table
	session uint64
	err     error
}

type rowsLoadedMsg struct {
	tableName   string
	offset      int
	selectedRow int
	page        db.RowPage
	session     uint64
	err         error
}

type tableDDLLoadedMsg struct {
	tableName string
	sql       string
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

func loadTables(database db.Database, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		tables, err := database.ListTables(ctx)
		return tablesLoadedMsg{tables: tables, session: session, err: err}
	}
}

func loadRows(database db.Database, table db.Table, offset, selectedRow int, session uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		page, err := database.GetRows(ctx, table, db.PageRequest{
			Offset: offset,
			Limit:  rowPageSize,
		})
		return rowsLoadedMsg{
			tableName:   table.Name,
			offset:      offset,
			selectedRow: selectedRow,
			page:        page,
			session:     session,
			err:         err,
		}
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
