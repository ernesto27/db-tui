package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

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

	request := query.beginExecute("SELECT 1")

	assert.Equal(t, uint64(8), request)
	assert.Equal(t, uint64(8), query.request)
	assert.True(t, query.loading)
	assert.Empty(t, query.result.CommandTag)
	assert.NoError(t, query.err)
	assert.Zero(t, query.viewport)
	assert.False(t, query.resultsFocused)
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
	query.finishExecute(result, nil)

	assert.False(t, query.loading)
	assert.Equal(t, result, query.result)
	assert.NoError(t, query.err)
	assert.Zero(t, query.viewport)
	assert.True(t, query.resultsFocused)
	assert.False(t, query.editor.Focused())
}

func TestQueryFinishExecuteKeepsCommandResultUnfocused(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.loading = true
	result := db.QueryResult{CommandTag: "UPDATE 3"}

	query.finishExecute(result, nil)

	assert.False(t, query.loading)
	assert.Equal(t, result, query.result)
	assert.False(t, query.resultsFocused)
}

func TestQueryFinishExecuteReturnsError(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.loading = true
	wantErr := errors.New("query failed")

	query.finishExecute(db.QueryResult{}, wantErr)

	assert.False(t, query.loading)
	assert.ErrorIs(t, query.err, wantErr)
	assert.False(t, query.resultsFocused)
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
	assert.True(t, model.spinnerRunning)
}
