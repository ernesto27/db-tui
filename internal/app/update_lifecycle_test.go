package app

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestUpdateIgnoresTablesFromOldSession(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.loading = true
	model.navigator.tables = []db.Table{{Name: "Current"}}

	updated, command := updateModel(t, model, tablesLoadedMsg{
		tables:  []db.Table{{Name: "Stale"}},
		session: 7,
	})

	assert.Nil(t, command)
	assert.True(t, updated.loading)
	assert.Equal(t, []db.Table{{Name: "Current"}}, updated.navigator.tables)
}

func TestUpdateAppliesTablesFromCurrentSession(t *testing.T) {
	database := &fakeDatabase{name: "chinook"}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = database
	model.session = 8
	model.loading = true
	model.navigator.tables = []db.Table{{Name: "Old"}}
	model.data.page = db.RowPage{Rows: [][]any{{"old"}}}

	updated, command := updateModel(t, model, tablesLoadedMsg{
		tables:  []db.Table{{Name: "Album"}},
		session: 8,
	})

	assert.NotNil(t, command)
	assert.False(t, updated.loading)
	assert.Equal(t, []db.Table{{Name: "Album"}}, updated.navigator.tables)
	assert.True(t, updated.data.loading)
	assert.Empty(t, updated.data.page.Rows)
	assert.Zero(t, updated.data.offset)
}

func TestUpdateIgnoresStaleRowResults(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Track"}}
	model.data = dataModel{
		page:     db.RowPage{Rows: [][]any{{"current"}}},
		offset:   100,
		selected: 0,
		loading:  true,
	}

	tests := []struct {
		name      string
		session   uint64
		tableName string
		offset    int
	}{
		{
			name:      "old session",
			session:   7,
			tableName: "Track",
			offset:    100,
		},
		{
			name:      "old table",
			session:   8,
			tableName: "Album",
			offset:    100,
		},
		{
			name:      "old page",
			session:   8,
			tableName: "Track",
			offset:    0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			updated, command := updateModel(t, model, rowsLoadedMsg{
				tableName: test.tableName,
				offset:    test.offset,
				page:      db.RowPage{Rows: [][]any{{"stale"}}},
				session:   test.session,
			})

			assert.Nil(t, command)
			assert.True(t, updated.data.loading)
			assert.Equal(t, [][]any{{"current"}}, updated.data.page.Rows)
			assert.Equal(t, 100, updated.data.offset)
			assert.Zero(t, updated.data.selected)
		})
	}
}

func TestUpdateAppliesMatchingRowResult(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Track"}}
	model.data = dataModel{
		offset:  100,
		loading: true,
	}
	page := db.RowPage{
		Columns: []string{"id"},
		Rows:    [][]any{{1}, {2}},
	}

	updated, command := updateModel(t, model, rowsLoadedMsg{
		tableName:   "Track",
		offset:      100,
		selectedRow: 1,
		page:        page,
		session:     8,
	})

	assert.Nil(t, command)
	assert.False(t, updated.data.loading)
	assert.Equal(t, page, updated.data.page)
	assert.Equal(t, 1, updated.data.selected)
	assert.NoError(t, updated.data.err)
}

func TestUpdateAppliesMatchingRowError(t *testing.T) {
	wantErr := errors.New("rows failed")
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Track"}}
	model.data = dataModel{
		offset:  100,
		loading: true,
	}

	updated, command := updateModel(t, model, rowsLoadedMsg{
		tableName: "Track",
		offset:    100,
		session:   8,
		err:       wantErr,
	})

	assert.Nil(t, command)
	assert.False(t, updated.data.loading)
	assert.ErrorIs(t, updated.data.err, wantErr)
}

func TestUpdateAppliesOnlyCurrentTableDDLResult(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Album"}}
	modal := newDDLModal("Album")
	model.ddlModal = &modal
	model.ddlRequest = 4

	updated, command := updateModel(t, model, tableDDLLoadedMsg{
		tableName: "Album",
		sql:       "CREATE TABLE public.\"Album\" ();",
		session:   8,
		request:   4,
	})

	assert.Nil(t, command)
	require.NotNil(t, updated.ddlModal)
	assert.False(t, updated.ddlModal.loading)
	assert.Equal(t, "CREATE TABLE public.\"Album\" ();", updated.ddlModal.sql)

	stale, command := updateModel(t, updated, tableDDLLoadedMsg{
		tableName: "Album",
		sql:       "stale",
		session:   8,
		request:   3,
	})

	assert.Nil(t, command)
	require.NotNil(t, stale.ddlModal)
	assert.Equal(t, "CREATE TABLE public.\"Album\" ();", stale.ddlModal.sql)
}

func TestUpdateIgnoresStaleQueryResult(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 5
	model.query.request = 11
	model.query.loading = true
	model.query.executionDuration = 100 * time.Millisecond

	tests := []struct {
		name    string
		session uint64
		request uint64
	}{
		{name: "old session", session: 4, request: 11},
		{name: "old request", session: 5, request: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			updated, command := updateModel(t, model, queryFinishedMsg{
				result:  db.QueryResult{CommandTag: "SELECT 1"},
				session: test.session,
				request: test.request,
			})

			assert.Nil(t, command)
			assert.True(t, updated.query.loading)
			assert.Empty(t, updated.query.result.CommandTag)
			assert.Equal(t, 100*time.Millisecond, updated.query.executionDuration)
		})
	}
}

func TestUpdateAppliesMatchingQueryResult(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 5
	model.query.request = 11
	model.query.loading = true
	result := db.QueryResult{
		Columns:    []string{"id"},
		Rows:       [][]any{{1}},
		CommandTag: "SELECT 1",
	}

	updated, command := updateModel(t, model, queryFinishedMsg{
		result:  result,
		session: 5,
		request: 11,
		elapsed: time.Second,
	})

	assert.Nil(t, command)
	assert.False(t, updated.query.loading)
	assert.Equal(t, result, updated.query.result)
	assert.Equal(t, time.Second, updated.query.executionDuration)
	assert.True(t, updated.query.resultsFocused)
}

func TestUpdateAppliesMatchingQueryError(t *testing.T) {
	wantErr := errors.New("query failed")
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 5
	model.query.request = 11
	model.query.loading = true

	updated, command := updateModel(t, model, queryFinishedMsg{
		session: 5,
		request: 11,
		elapsed: time.Second,
		err:     wantErr,
	})

	assert.Nil(t, command)
	assert.False(t, updated.query.loading)
	assert.ErrorIs(t, updated.query.err, wantErr)
	assert.Equal(t, time.Second, updated.query.executionDuration)
}

func TestUpdateClosesDatabaseFromStaleConnectionAttempt(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newConnectionModal(ConnectionSettings{})
	model.modal = &modal
	model.connectionAttempt = 4
	staleDatabase := &fakeDatabase{name: "stale"}

	updated, command := updateModel(t, model, connectionFinishedMsg{
		database: staleDatabase,
		attempt:  3,
	})

	assert.Nil(t, command)
	assert.Equal(t, 1, staleDatabase.closeCalls)
	assert.Nil(t, updated.database)
	assert.NotNil(t, updated.modal)
}

func TestUpdateReplacesActiveDatabaseAfterCurrentConnectionAttempt(t *testing.T) {
	oldDatabase := &fakeDatabase{name: "old"}
	newDatabase := &fakeDatabase{name: "chinook"}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = oldDatabase
	model.databaseName = "old"
	model.session = 10
	model.navigator.tables = []db.Table{{Name: "Old"}}
	model.data.page = db.RowPage{Rows: [][]any{{"old"}}}
	model.query.result = db.QueryResult{CommandTag: "SELECT 1"}
	modal := newConnectionModal(ConnectionSettings{})
	modal.connecting = true
	model.modal = &modal
	model.connectionAttempt = 4
	settings := ConnectionSettings{DSN: "postgres://new"}

	updated, command := updateModel(t, model, connectionFinishedMsg{
		database: newDatabase,
		settings: settings,
		attempt:  4,
	})

	require.NotNil(t, command)
	assert.Equal(t, 1, oldDatabase.closeCalls)
	assert.Zero(t, newDatabase.closeCalls)
	assert.Same(t, newDatabase, updated.database)
	assert.Equal(t, "chinook", updated.databaseName)
	assert.Equal(t, settings.Engine, updated.databaseEngine)
	assert.Equal(t, settings, updated.savedConnection)
	assert.Equal(t, uint64(11), updated.session)
	assert.Nil(t, updated.modal)
	assert.Empty(t, updated.navigator.tables)
	assert.Empty(t, updated.data.page.Rows)
	assert.Empty(t, updated.query.result.CommandTag)
	assert.True(t, updated.loading)
	assert.True(t, updated.spinnerRunning)
}

func TestUpdateIgnoresDumpResultFromOldSession(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 6
	modal := newDumpModal("chinook")
	modal.state = dumpRunning
	model.dumpModal = &modal

	updated, command := updateModel(t, model, dumpFinishedMsg{session: 5})

	assert.Nil(t, command)
	require.NotNil(t, updated.dumpModal)
	assert.Equal(t, dumpRunning, updated.dumpModal.state)
}

func TestUpdateIgnoresDumpResultWithoutModal(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 6

	updated, command := updateModel(t, model, dumpFinishedMsg{session: 6})

	assert.Nil(t, command)
	assert.Nil(t, updated.dumpModal)
}

func TestUpdateAppliesMatchingDumpResult(t *testing.T) {
	wantErr := errors.New("dump failed")
	tests := []struct {
		name      string
		err       error
		wantState dumpModalState
	}{
		{name: "success", wantState: dumpSucceeded},
		{name: "failure", err: wantErr, wantState: dumpFailed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.session = 6
			modal := newDumpModal("chinook")
			modal.state = dumpRunning
			model.dumpModal = &modal

			updated, command := updateModel(t, model, dumpFinishedMsg{
				session: 6,
				err:     test.err,
			})

			assert.Nil(t, command)
			require.NotNil(t, updated.dumpModal)
			assert.Equal(t, test.wantState, updated.dumpModal.state)
			assert.ErrorIs(t, updated.dumpModal.err, test.err)
		})
	}
}

func TestUpdateResizesAndClampsDataState(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.data = dataModel{
		page: db.RowPage{
			Columns: []string{"id", "name"},
			Rows:    [][]any{{1, "one"}, {2, "two"}},
		},
		selected:     10,
		columnOffset: 10,
	}

	updated, command := updateModel(t, model, tea.WindowSizeMsg{
		Width:  20,
		Height: 5,
	})

	assert.Nil(t, command)
	assert.Equal(t, 64, updated.layout.width)
	assert.Equal(t, 16, updated.layout.height)
	assert.Equal(t, 1, updated.data.columnOffset)
	assert.Equal(t, 1, updated.data.selected)
}

func TestUpdateAdvancesSpinnerWhileWorkIsLoading(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.loading = true
	model.spinnerRunning = true

	updated, command := updateModel(t, model, spinnerTickMsg{})

	assert.NotNil(t, command)
	assert.Equal(t, 1, updated.spinnerFrame)
	assert.True(t, updated.spinnerRunning)
}

func TestUpdateStopsSpinnerWhenWorkFinishes(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.spinnerRunning = true

	updated, command := updateModel(t, model, spinnerTickMsg{})

	assert.Nil(t, command)
	assert.False(t, updated.spinnerRunning)
	assert.Zero(t, updated.spinnerFrame)
}

func TestUpdateIgnoresMouseWhileConnectionModalIsOpen(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newConnectionModal(ConnectionSettings{})
	model.modal = &modal
	model.focus = focusNavigator

	updated, command := updateModel(t, model, tea.MouseClickMsg{
		X:      model.layout.navigator.width + 2,
		Y:      model.layout.navigatorListY,
		Button: tea.MouseLeft,
	})

	assert.Nil(t, command)
	assert.Equal(t, focusNavigator, updated.focus)
	assert.NotNil(t, updated.modal)
}

func TestUpdateIgnoresMouseWhileConnectionsModalIsOpen(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newConnectionsModal(config.Config{})
	model.connectionsModal = &modal
	model.focus = focusNavigator

	updated, command := updateModel(t, model, tea.MouseWheelMsg{
		X:      model.layout.navigator.width + 2,
		Y:      model.layout.navigatorListY,
		Button: tea.MouseWheelDown,
	})

	assert.Nil(t, command)
	assert.Equal(t, focusNavigator, updated.focus)
	assert.NotNil(t, updated.connectionsModal)
}

func TestUpdateClosesConnectionModalAfterCancelMessage(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newConnectionModal(ConnectionSettings{})
	model.modal = &modal

	updated, command := updateModel(t, model, cancelConnectionMsg{})

	assert.Nil(t, command)
	assert.Nil(t, updated.modal)
}

func TestConnectionModalEscapeEmitsCancelMessage(t *testing.T) {
	modal := newConnectionModal(ConnectionSettings{})

	_, command := modal.update(keyPress(tea.KeyEscape, "", 0))
	require.NotNil(t, command)
	_, ok := command().(cancelConnectionMsg)
	assert.True(t, ok)
}

func TestConnectionModalIgnoresInputWhileConnecting(t *testing.T) {
	modal := newConnectionModal(ConnectionSettings{})
	modal.connecting = true
	before := modal.inputs[hostInput].Value()

	updated, command := modal.update(keyPress('x', "x", 0))

	assert.Nil(t, command)
	assert.Equal(t, before, updated.inputs[hostInput].Value())
}

func TestDumpModalIgnoresKeysWhileRunning(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newDumpModal("chinook")
	modal.state = dumpRunning
	model.dumpModal = &modal

	updated, command := updateModel(t, model, keyPress(tea.KeyEscape, "", 0))

	assert.Nil(t, command)
	require.NotNil(t, updated.dumpModal)
	assert.Equal(t, dumpRunning, updated.dumpModal.state)
}
