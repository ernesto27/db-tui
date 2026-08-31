package app

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestQueryBeginExecuteResetsPreviousState(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.result = db.QueryResult{CommandTag: "old"}
	query.err = errors.New("old")
	query.viewport = 4
	query.resultsFocused = true
	query.request = 7
	query.executionDuration = time.Second

	request := query.beginExecute("SELECT 1")

	assert.Equal(t, uint64(8), request)
	assert.Equal(t, uint64(8), query.request)
	assert.True(t, query.loading)
	assert.Empty(t, query.result.CommandTag)
	assert.NoError(t, query.err)
	assert.Zero(t, query.viewport)
	assert.False(t, query.resultsFocused)
	assert.Zero(t, query.executionDuration)
	assert.Equal(t, "SELECT 1", query.lastExecutedSQL)
}

func TestQueryFinishExecuteFocusesReturnedRows(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.loading = true
	_ = query.editor.Focus()

	result := db.QueryResult{
		Columns:    []string{"id"},
		Rows:       [][]any{{1}, {2}},
		CommandTag: "SELECT 2",
	}
	query.finishExecute(result, time.Millisecond, nil)

	assert.False(t, query.loading)
	assert.Equal(t, result, query.result)
	assert.NoError(t, query.err)
	assert.Equal(t, time.Millisecond, query.executionDuration)
	assert.Zero(t, query.viewport)
	assert.True(t, query.resultsFocused)
	assert.False(t, query.editor.Focused())
}

func TestQueryFinishExecuteKeepsCommandResultUnfocused(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.loading = true
	result := db.QueryResult{CommandTag: "UPDATE 3"}

	query.finishExecute(result, time.Millisecond, nil)

	assert.False(t, query.loading)
	assert.Equal(t, result, query.result)
	assert.Equal(t, time.Millisecond, query.executionDuration)
	assert.False(t, query.resultsFocused)
}

func TestQueryFinishExecuteReturnsError(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.loading = true
	wantErr := errors.New("query failed")

	query.finishExecute(db.QueryResult{}, time.Millisecond, wantErr)

	assert.False(t, query.loading)
	assert.ErrorIs(t, query.err, wantErr)
	assert.Equal(t, time.Millisecond, query.executionDuration)
	assert.False(t, query.resultsFocused)
}

func TestQueryResultViewIncludesExecutionTime(t *testing.T) {
	layout := newAppLayout(100, 24)
	tests := []struct {
		name   string
		result db.QueryResult
		err    error
		want   string
	}{
		{
			name:   "command",
			result: db.QueryResult{CommandTag: "UPDATE 3"},
			want:   "Command completed: UPDATE 3  •  Execution time: 1.25s",
		},
		{
			name:   "empty rows",
			result: db.QueryResult{Columns: []string{"id"}, CommandTag: "SELECT 0"},
			want:   "Query returned no rows.  •  SELECT 0  •  Execution time: 1.25s",
		},
		{
			name: "rows",
			result: db.QueryResult{
				Columns:    []string{"id"},
				Rows:       [][]any{{1}},
				CommandTag: "SELECT 1",
			},
			want: "Execution time: 1.25s",
		},
		{
			name: "error",
			err:  errors.New("query failed"),
			want: "Query failed  •  Execution time: 1.25s",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := newQueryModel(layout)
			query.result = test.result
			query.err = test.err
			query.executionDuration = 1250 * time.Millisecond

			assert.Contains(t, query.resultView(layout, true), test.want)
		})
	}
}

func TestQueryResultViewShowsElapsedTimeWhileLoading(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.loading = true
	query.executionDuration = 2 * time.Second

	result := query.resultView(layout, true)
	assert.Contains(t, result, "Query executing: 2s")
	assert.Contains(t, result, queryCancelControlText)
}

func TestQueryCancelControlClickCallsCancel(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.panel = panelQuery
	model.query.loading = true
	model.query.executionDuration = 2 * time.Second
	canceled := false
	model.query.cancel = func() { canceled = true }

	controlX := model.layout.data.x + 2 + len(queryExecutingText) + len(model.query.executionDuration.String()) + 2
	controlY := model.layout.data.y + querySectionHeight(model.layout) + 1
	updated, command := updateModel(t, model, tea.MouseClickMsg{X: controlX, Y: controlY, Button: tea.MouseLeft})

	assert.Nil(t, command)
	assert.True(t, updated.query.loading)
	assert.True(t, canceled)
}

func TestQueryCancelControlCancelsExecutingQueryContext(t *testing.T) {
	database := newBlockingFakeDatabase()
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = database
	model.panel = panelQuery
	model.query.editor.SetValue("SELECT SLEEP(10)")

	command := model.startQuery()
	require.NotNil(t, command)
	batch, ok := command().(tea.BatchMsg)
	require.True(t, ok)
	require.NotEmpty(t, batch)

	controlX := model.layout.data.x + 2 + len(queryExecutingText) + len(model.query.executionDuration.String()) + 2
	controlY := model.layout.data.y + querySectionHeight(model.layout) + 1
	updated, clickCommand := updateModel(t, model, tea.MouseClickMsg{X: controlX, Y: controlY, Button: tea.MouseLeft})

	assert.Nil(t, clickCommand)
	finished, ok := batch[0]().(queryFinishedMsg)
	require.True(t, ok)
	assert.ErrorIs(t, finished.err, context.Canceled)
	assert.True(t, database.executeCanceled)

	updated, _ = updateModel(t, updated, finished)
	assert.False(t, updated.query.loading)
	assert.Nil(t, updated.query.cancel)
}

func TestModelCloseCancelsActiveQueryBeforeClosingDatabase(t *testing.T) {
	database := newBlockingFakeDatabase()
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = database
	model.panel = panelQuery
	model.query.editor.SetValue("SELECT SLEEP(10)")

	finished := startCancelableQuery(t, &model, database)
	model.Close()

	assert.True(t, database.closeSawCanceledQuery)
	assertCanceledQueryFinished(t, finished)
}

func TestConnectionReplacementCancelsActiveQueryBeforeClosingDatabase(t *testing.T) {
	oldDatabase := newBlockingFakeDatabase()
	newDatabase := &fakeDatabase{name: "new", engine: db.EnginePostgreSQL}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = oldDatabase
	model.panel = panelQuery
	model.query.editor.SetValue("SELECT SLEEP(10)")
	modal := newConnectionModal(ConnectionSettings{})
	modal.connecting = true
	model.modal = &modal
	model.connectionAttempt = 1

	finished := startCancelableQuery(t, &model, oldDatabase)
	updated, _ := updateModel(t, model, connectionFinishedMsg{
		database: newDatabase,
		attempt:  1,
	})

	assert.True(t, oldDatabase.closeSawCanceledQuery)
	assert.Same(t, newDatabase, updated.database)
	assertCanceledQueryFinished(t, finished)
}

func TestDeletingActiveConnectionCancelsQueryBeforeClosingDatabase(t *testing.T) {
	appConfig := config.Config{Connections: []config.Connection{{Name: "active", Engine: db.EnginePostgreSQL}}}
	database := newBlockingFakeDatabase()
	model := New(appConfig, ConnectionSettings{}, nil)
	model.database = database
	model.activeConnectionIndex = 0
	model.panel = panelQuery
	model.query.editor.SetValue("SELECT SLEEP(10)")
	connectionsModal := newConnectionsModal(appConfig)
	model.connectionsModal = &connectionsModal

	finished := startCancelableQuery(t, &model, database)
	updated, _ := updateModel(t, model, deleteConnectionMsg{index: 0, connection: appConfig.Connections[0]})

	assert.True(t, database.closeSawCanceledQuery)
	assert.Nil(t, updated.database)
	assertCanceledQueryFinished(t, finished)
}

func newBlockingFakeDatabase() *fakeDatabase {
	return &fakeDatabase{
		name:                      "chinook",
		blockExecuteUntilCanceled: true,
		executeStarted:            make(chan struct{}),
	}
}

func startCancelableQuery(t *testing.T, model *Model, database *fakeDatabase) <-chan tea.Msg {
	t.Helper()

	command := model.startQuery()
	require.NotNil(t, command)
	batch, ok := command().(tea.BatchMsg)
	require.True(t, ok)
	require.NotEmpty(t, batch)

	finished := make(chan tea.Msg, 1)
	go func() {
		finished <- batch[0]()
	}()

	select {
	case <-database.executeStarted:
	case <-time.After(time.Second):
		t.Fatal("query did not begin execution")
	}
	return finished
}

func assertCanceledQueryFinished(t *testing.T, finished <-chan tea.Msg) {
	t.Helper()

	select {
	case message := <-finished:
		queryFinished, ok := message.(queryFinishedMsg)
		require.True(t, ok)
		assert.ErrorIs(t, queryFinished.err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("query did not finish after cancellation")
	}
}

func TestQueryScrollResultsClamps(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.result.Rows = [][]any{{1}, {2}, {3}}

	query.scrollResults(100)
	assert.Equal(t, 2, query.viewport)

	query.scrollResults(-100)
	assert.Zero(t, query.viewport)
}

func TestQueryScrollResultsResetsEmptyResult(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.viewport = 5

	query.scrollResults(1)

	assert.Zero(t, query.viewport)
}

func TestQueryToggleFocus(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.result.Rows = [][]any{{1}}
	_ = query.focusEditor()

	command := query.toggleFocus()
	assert.Nil(t, command)
	assert.True(t, query.resultsFocused)
	assert.False(t, query.editor.Focused())

	command = query.toggleFocus()
	assert.NotNil(t, command)
	assert.False(t, query.resultsFocused)
	assert.True(t, query.editor.Focused())
}

func TestModelStartQueryRequiresDatabaseAndSQL(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.query.editor.SetValue("SELECT 1")

	assert.Nil(t, model.startQuery())

	model.database = &fakeDatabase{name: "chinook"}
	model.query.editor.SetValue(" \n\t ")
	assert.Nil(t, model.startQuery())
	assert.False(t, model.query.loading)
}

func TestModelStartQueryBeginsExecution(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.query.editor.SetValue("SELECT 1")

	command := model.startQuery()

	assert.NotNil(t, command)
	assert.True(t, model.query.loading)
	assert.Equal(t, uint64(1), model.query.request)
	assert.False(t, model.spinnerRunning)
}

func TestModelStartQueryIgnoresSubmissionWhileExecuting(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.query.editor.SetValue("SELECT 1")
	model.query.loading = true
	model.query.request = 4

	command := model.startQuery()

	assert.Nil(t, command)
	assert.True(t, model.query.loading)
	assert.Equal(t, uint64(4), model.query.request)
}
