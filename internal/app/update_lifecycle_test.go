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

	assert.Nil(t, command)
	assert.False(t, updated.loading)
	assert.Equal(t, []db.Table{{Name: "Album"}}, updated.navigator.tables)
	assert.False(t, updated.data.loading)
	assert.Equal(t, [][]any{{"old"}}, updated.data.page.Rows)
	assert.Zero(t, updated.data.offset)
}

func TestUpdateSelectsViewForViewsOnlyDatabaseRegardlessOfLoadOrder(t *testing.T) {
	for _, test := range []struct {
		name  string
		first tea.Msg
		last  tea.Msg
	}{
		{
			name:  "views load first",
			first: viewsLoadedMsg{views: []db.View{{Name: "ExampleView"}}, session: 8},
			last:  tablesLoadedMsg{session: 8},
		},
		{
			name:  "tables load first",
			first: tablesLoadedMsg{session: 8},
			last:  viewsLoadedMsg{views: []db.View{{Name: "ExampleView"}}, session: 8},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = &fakeDatabase{name: "views-only"}
			model.session = 8
			model.loading = true
			model.viewsLoading = true

			updated, command := updateModel(t, model, test.first)
			assert.Nil(t, command)

			updated, command = updateModel(t, updated, test.last)
			assert.Nil(t, command)
			assert.Equal(t, navigatorViews, updated.navigator.section)
			assert.Equal(t, "ExampleView", updated.navigator.selectedName())
			assert.False(t, updated.data.loading)
			assert.False(t, updated.activeRelation.set)
		})
	}
}

func TestUpdateSelectsMaterializedViewForMaterializedViewsOnlyDatabase(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "materialized-views-only", engine: db.EnginePostgreSQL}
	model.session = 8
	model.loading = true
	model.viewsLoading = true
	model.materializedViewsLoading = true
	model.navigator.setMaterializedViewsAvailable(true)

	updated, command := updateModel(t, model, tablesLoadedMsg{session: 8})
	assert.Nil(t, command)

	updated, command = updateModel(t, updated, viewsLoadedMsg{session: 8})
	assert.Nil(t, command)

	updated, command = updateModel(t, updated, materializedViewsLoadedMsg{
		materializedViews: []db.MaterializedView{{Name: "SalesByCountry"}},
		session:           8,
	})

	assert.Nil(t, command)
	assert.Equal(t, navigatorMaterializedViews, updated.navigator.section)
	assert.Equal(t, "SalesByCountry", updated.navigator.selectedName())
	assert.False(t, updated.data.loading)
	assert.False(t, updated.activeRelation.set)
}

func TestUpdateSelectsAvailableRelationWhenOtherDiscoveryFailsLast(t *testing.T) {
	discoveryErr := errors.New("materialized-view discovery failed")

	tests := []struct {
		name        string
		messages    []tea.Msg
		wantSection navigatorSection
		wantName    string
	}{
		{
			name: "materialized-view discovery fails after ordinary views load",
			messages: []tea.Msg{
				viewsLoadedMsg{views: []db.View{{Name: "ExampleView"}}, session: 8},
				tablesLoadedMsg{session: 8},
				materializedViewsLoadedMsg{session: 8, err: discoveryErr},
			},
			wantSection: navigatorViews,
			wantName:    "ExampleView",
		},
		{
			name: "ordinary-view discovery fails after materialized views load",
			messages: []tea.Msg{
				materializedViewsLoadedMsg{
					materializedViews: []db.MaterializedView{{Name: "SalesByCountry"}},
					session:           8,
				},
				tablesLoadedMsg{session: 8},
				viewsLoadedMsg{session: 8, err: discoveryErr},
			},
			wantSection: navigatorMaterializedViews,
			wantName:    "SalesByCountry",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = &fakeDatabase{name: "relations", engine: db.EnginePostgreSQL}
			model.session = 8
			model.loading = true
			model.viewsLoading = true
			model.materializedViewsLoading = true
			model.navigator.setMaterializedViewsAvailable(true)

			var command tea.Cmd
			for _, message := range test.messages {
				model, command = updateModel(t, model, message)
			}

			assert.Nil(t, command)
			assert.Equal(t, test.wantSection, model.navigator.section)
			assert.Equal(t, test.wantName, model.navigator.selectedName())
			assert.False(t, model.data.loading)
			assert.False(t, model.activeRelation.set)
		})
	}
}

func TestUpdateIgnoresStaleRowResults(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Track"}}
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Track", section: navigatorTables},
		request: 6,
		set:     true,
	}
	model.data = dataModel{
		page:     db.RowPage{Rows: [][]any{{"current"}}},
		offset:   100,
		selected: 0,
		loading:  true,
	}

	tests := []struct {
		name     string
		session  uint64
		relation navigatorItem
		offset   int
	}{
		{
			name:     "old session",
			session:  7,
			relation: navigatorItem{name: "Track", section: navigatorTables},
			offset:   100,
		},
		{
			name:     "old table",
			session:  8,
			relation: navigatorItem{name: "Album", section: navigatorTables},
			offset:   100,
		},
		{
			name:     "old page",
			session:  8,
			relation: navigatorItem{name: "Track", section: navigatorTables},
			offset:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			updated, command := updateModel(t, model, rowsLoadedMsg{
				relation: test.relation,
				offset:   test.offset,
				page:     db.RowPage{Rows: [][]any{{"stale"}}},
				session:  test.session,
				request:  6,
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
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Track", section: navigatorTables},
		request: 6,
		set:     true,
	}
	model.data = dataModel{
		offset:  100,
		loading: true,
	}
	page := db.RowPage{
		Columns: []string{"id"},
		Rows:    [][]any{{1}, {2}},
	}

	updated, command := updateModel(t, model, rowsLoadedMsg{
		relation:    navigatorItem{name: "Track", section: navigatorTables},
		offset:      100,
		selectedRow: 1,
		page:        page,
		session:     8,
		request:     6,
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
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Track", section: navigatorTables},
		request: 6,
		set:     true,
	}
	model.data = dataModel{
		offset:  100,
		loading: true,
	}

	updated, command := updateModel(t, model, rowsLoadedMsg{
		relation: navigatorItem{name: "Track", section: navigatorTables},
		offset:   100,
		session:  8,
		request:  6,
		err:      wantErr,
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

func TestUpdateAppliesOnlyCurrentColumnsResult(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Album"}}
	modal := newColumnsModal("Album")
	model.columnsModal = &modal
	model.columnsRequest = 4
	columns := []db.Column{{Name: "AlbumId", OrdinalPosition: 1, DataType: "int4", NotNull: true}}

	updated, command := updateModel(t, model, columnsLoadedMsg{
		tableName: "Album",
		columns:   columns,
		session:   8,
		request:   4,
	})

	assert.Nil(t, command)
	require.NotNil(t, updated.columnsModal)
	assert.False(t, updated.columnsModal.loading)
	assert.Equal(t, columns, updated.columnsModal.columns)

	stale, command := updateModel(t, updated, columnsLoadedMsg{
		tableName: "Album",
		columns:   []db.Column{{Name: "stale"}},
		session:   8,
		request:   3,
	})

	assert.Nil(t, command)
	require.NotNil(t, stale.columnsModal)
	assert.Equal(t, columns, stale.columnsModal.columns)
}

func TestUpdateAppliesOnlyCurrentIndexesResult(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Album"}}
	modal := newIndexesModal("Album")
	model.indexesModal = &modal
	model.indexesRequest = 4
	indexes := []db.IndexColumns{{Name: "album_pkey", Column: "AlbumId", Table: "Album", AccessMethod: "btree"}}

	updated, command := updateModel(t, model, indexesLoadedMsg{
		tableName: "Album",
		indexes:   indexes,
		session:   8,
		request:   4,
	})

	assert.Nil(t, command)
	require.NotNil(t, updated.indexesModal)
	assert.False(t, updated.indexesModal.loading)
	assert.Equal(t, indexes, updated.indexesModal.indexes)

	stale, command := updateModel(t, updated, indexesLoadedMsg{
		tableName: "Album",
		indexes:   []db.IndexColumns{{Name: "stale"}},
		session:   8,
		request:   3,
	})

	assert.Nil(t, command)
	require.NotNil(t, stale.indexesModal)
	assert.Equal(t, indexes, stale.indexesModal.indexes)
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
	oldDatabase := &fakeDatabase{name: "old", engine: db.EnginePostgreSQL}
	newDatabase := &fakeDatabase{name: "chinook", engine: db.EnginePostgreSQL}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = oldDatabase
	model.session = 10
	model.navigator.tables = []db.Table{{Name: "Old"}}
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Old", section: navigatorTables},
		request: 4,
		set:     true,
	}
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
	assert.Equal(t, "chinook", updated.database.Name())
	assert.Equal(t, db.EnginePostgreSQL, updated.database.Engine())
	assert.Equal(t, settings, updated.savedConnection)
	assert.Equal(t, uint64(11), updated.session)
	assert.Nil(t, updated.modal)
	assert.Empty(t, updated.navigator.tables)
	assert.False(t, updated.activeRelation.set)
	assert.Zero(t, updated.activeRelation.request)
	assert.Empty(t, updated.data.page.Rows)
	assert.Empty(t, updated.query.result.CommandTag)
	assert.True(t, updated.loading)
	assert.True(t, updated.spinnerRunning)
}

func TestUpdateEnablesMaterializedViewsForOracleConnection(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newConnectionModal(ConnectionSettings{})
	modal.connecting = true
	model.modal = &modal
	model.connectionAttempt = 1

	updated, command := updateModel(t, model, connectionFinishedMsg{
		database: &fakeDatabase{name: "FREEPDB1", engine: db.EngineOracle},
		attempt:  1,
	})

	require.NotNil(t, command)
	assert.True(t, updated.navigator.materializedViewsAvailable)
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
