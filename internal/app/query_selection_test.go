package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestQueryMouseSelectionResolvesLogicalSQL(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		width    int
		startCol int
		startRow int
		endCol   int
		endRow   int
		want     string
	}{
		{
			name:     "forward multiline drag",
			sql:      "one\nTWO\nthree",
			startCol: 1, startRow: 0,
			endCol: 1, endRow: 2,
			want: "ne\nTWO\nth",
		},
		{
			name:     "reverse multiline drag",
			sql:      "one\nTWO\nthree",
			startCol: 1, startRow: 2,
			endCol: 1, endRow: 0,
			want: "ne\nTWO\nth",
		},
		{
			name:     "soft wrapped selection preserves whitespace",
			sql:      "ab cd",
			width:    4,
			startCol: 2, startRow: 0,
			endCol: 1, endRow: 1,
			want: " cd",
		},
		{
			name:     "wide character coordinates select the rune once",
			sql:      "A界B",
			startCol: 2, startRow: 0,
			endCol: 3, endRow: 0,
			want: "界B",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := newQuerySelectionModel(test.sql, test.width)
			updated := dragQuerySelection(t, model, test.startCol, test.startRow, test.endCol, test.endRow)

			selected, ok := updated.query.selection.selectedSQL(updated.query.editor.Value())
			assert.True(t, ok)
			assert.Equal(t, test.want, selected)
			assert.Equal(t, ansi.Strip(updated.query.editor.View()), ansi.Strip(updated.query.editorView(updated.layout)))
			assert.NotEqual(t, updated.query.editor.View(), updated.query.editorView(updated.layout))
		})
	}
}

func TestQueryMouseSelectionBoundariesAndClicks(t *testing.T) {
	t.Run("editor boundary characters can both be selected", func(t *testing.T) {
		model := newQuerySelectionModel("SELECT", 0)
		updated := dragQuerySelection(t, model, 0, 0, len("SELECT")-1, 0)

		selected, ok := updated.query.selection.selectedSQL(updated.query.editor.Value())
		assert.True(t, ok)
		assert.Equal(t, "SELECT", selected)
	})

	t.Run("click without drag leaves no active selection", func(t *testing.T) {
		model := newQuerySelectionModel("SELECT 1", 0)
		left, top := queryEditorOrigin(model.layout)

		updated, _ := updateModel(t, model, tea.MouseClickMsg{X: left, Y: top, Button: tea.MouseLeft})
		updated, _ = updateModel(t, updated, tea.MouseReleaseMsg{X: left, Y: top, Button: tea.MouseLeft})

		assert.False(t, updated.query.selection.active)
		_, ok := updated.query.selection.selectedSQL(updated.query.editor.Value())
		assert.False(t, ok)
	})

	t.Run("dragging past the editor clamps to visible text", func(t *testing.T) {
		model := newQuerySelectionModel("SELECT", 0)
		left, top := queryEditorOrigin(model.layout)
		start := tea.MouseClickMsg{X: left, Y: top, Button: tea.MouseLeft}

		updated, _ := updateModel(t, model, start)
		updated, _ = updateModel(t, updated, tea.MouseMotionMsg{
			X: left + updated.query.editor.Width() + 10,
			Y: top + updated.query.editor.Height() + 10,
		})
		updated, _ = updateModel(t, updated, tea.MouseReleaseMsg{
			X:      left + updated.query.editor.Width() + 10,
			Y:      top + updated.query.editor.Height() + 10,
			Button: tea.MouseNone,
		})

		selected, ok := updated.query.selection.selectedSQL(updated.query.editor.Value())
		assert.True(t, ok)
		assert.Equal(t, "SELECT", selected)
	})

	t.Run("mouse events outside the editor do not create a selection", func(t *testing.T) {
		model := newQuerySelectionModel("SELECT 1", 0)

		updated, _ := updateModel(t, model, tea.MouseClickMsg{X: 1, Y: model.layout.data.y, Button: tea.MouseLeft})
		updated, _ = updateModel(t, updated, tea.MouseMotionMsg{X: 2, Y: model.layout.data.y})
		updated, _ = updateModel(t, updated, tea.MouseReleaseMsg{X: 2, Y: model.layout.data.y, Button: tea.MouseLeft})

		assert.False(t, updated.query.selection.active)
		assert.False(t, updated.query.selection.dragging())
	})
}

func TestQueryExecutionUsesSelectedSQL(t *testing.T) {
	t.Run("executes the active selection and retains it after completion", func(t *testing.T) {
		database := &fakeDatabase{name: "chinook", queryResult: db.QueryResult{
			Columns:    []string{"id"},
			Rows:       [][]any{{1}},
			CommandTag: "SELECT 1",
		}}
		model := newQuerySelectionModel("SELECT one;\nSELECT two;", 0)
		model.database = database
		model.query.selection = sqlSelection{anchor: 12, head: 22, active: true}

		command := model.startQuery()
		require.NotNil(t, command)
		batch, ok := command().(tea.BatchMsg)
		require.True(t, ok)
		finished, ok := batch[0]().(queryFinishedMsg)
		require.True(t, ok)

		updated, _ := updateModel(t, model, finished)
		assert.Equal(t, "SELECT two;", database.executedSQL)
		assert.Equal(t, "SELECT two;", updated.query.lastExecutedSQL)
		assert.True(t, updated.query.selection.active)
		assert.True(t, updated.query.resultsFocused)

		updated, command = updateModel(t, updated, keyPress('e', "", tea.ModCtrl))
		assert.Nil(t, command)
		require.NotNil(t, updated.exportModal)
		assert.Equal(t, "SELECT two;", updated.exportModal.query)
	})

	t.Run("falls back to the full editor when no selection exists", func(t *testing.T) {
		database := &fakeDatabase{name: "chinook"}
		model := newQuerySelectionModel("SELECT one;\nSELECT two;", 0)
		model.database = database

		command := model.startQuery()
		require.NotNil(t, command)
		batch := command().(tea.BatchMsg)
		_, ok := batch[0]().(queryFinishedMsg)
		require.True(t, ok)
		assert.Equal(t, "SELECT one;\nSELECT two;", database.executedSQL)
		assert.Equal(t, "SELECT one;\nSELECT two;", model.query.lastExecutedSQL)
	})

	t.Run("does not execute a whitespace-only active selection", func(t *testing.T) {
		database := &fakeDatabase{name: "chinook"}
		model := newQuerySelectionModel("SELECT 1   ", 0)
		model.database = database
		model.query.selection = sqlSelection{anchor: 8, head: 10, active: true}

		assert.Nil(t, model.startQuery())
		assert.Zero(t, database.executeCalls)
		assert.False(t, model.query.loading)
	})
}

func TestSelectedQueryExecutionSavesCompleteScript(t *testing.T) {
	const (
		connectionName = "selected query save"
		fullSQL        = "SELECT one;\nSELECT two;"
		selectedSQL    = "SELECT two;"
	)
	database := &fakeDatabase{name: "chinook"}
	model := newQuerySelectionModel(fullSQL, 0)
	model.database = database
	model.config = config.Config{Connections: []config.Connection{{Name: connectionName}}}
	model.activeConnectionIndex = 0
	model.query.selection = sqlSelection{anchor: 12, head: 22, active: true}

	command := model.startQuery()
	require.NotNil(t, command)
	batch := command().(tea.BatchMsg)
	finished, ok := batch[0]().(queryFinishedMsg)
	require.True(t, ok)
	saved, ok := batch[len(batch)-1]().(sqlScriptSavedMsg)
	require.True(t, ok)
	require.NoError(t, saved.err)

	scripts, err := model.sqlScripts.getList(connectionName)
	require.NoError(t, err)
	require.Len(t, scripts, 1)
	assert.Equal(t, fullSQL, scripts[0].content)
	assert.Equal(t, selectedSQL, database.executedSQL)
	assert.Equal(t, selectedSQL, model.query.lastExecutedSQL)
	assert.True(t, model.query.selection.active)

	updated, _ := updateModel(t, model, saved)
	updated, _ = updateModel(t, updated, finished)
	assert.Equal(t, selectedSQL, updated.query.lastExecutedSQL)
	assert.True(t, updated.query.selection.active)
}

func TestQuerySelectionClearsWhenEditorStateChanges(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Model)
		message tea.Msg
		wantSQL string
	}{
		{
			name: "editor mutation",
			setup: func(model *Model) {
				_ = model.query.focusEditor()
			},
			message: keyPress('x', "x", 0),
			wantSQL: "SELECT 1x",
		},
		{
			name: "cursor movement",
			setup: func(model *Model) {
				_ = model.query.focusEditor()
			},
			message: keyPress(tea.KeyLeft, "", 0),
			wantSQL: "SELECT 1",
		},
		{
			name: "paste",
			setup: func(model *Model) {
				_ = model.query.focusEditor()
			},
			message: tea.PasteMsg{Content: " + 1"},
			wantSQL: "SELECT 1 + 1",
		},
		{
			name: "script replacement",
			setup: func(model *Model) {
				modal := newSQLScriptsModal("query selection", 1)
				modal.loading = false
				modal.scripts = []SqlScript{{name: "replacement.sql", content: "SELECT 2"}}
				model.sqlScriptsModal = &modal
			},
			message: keyPress(tea.KeyEnter, "", 0),
			wantSQL: "SELECT 2",
		},
		{
			name:    "new script reset",
			message: keyPress('n', "", tea.ModCtrl),
			wantSQL: "",
		},
		{
			name: "reconnection",
			setup: func(model *Model) {
				modal := newConnectionModal(ConnectionSettings{})
				model.modal = &modal
				model.connectionAttempt = 1
			},
			message: connectionFinishedMsg{
				database: &fakeDatabase{name: "chinook"},
				attempt:  1,
			},
			wantSQL: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := newQuerySelectionModel("SELECT 1", 0)
			model.query.selection = sqlSelection{anchor: 0, head: 5, active: true}
			if test.setup != nil {
				test.setup(&model)
			}

			updated, _ := updateModel(t, model, test.message)

			assert.False(t, updated.query.selection.active)
			assert.Equal(t, test.wantSQL, updated.query.editor.Value())
		})
	}
}

func newQuerySelectionModel(sql string, width int) Model {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.panel = panelQuery
	model.query.editor.SetValue(sql)
	if width > 0 {
		model.query.editor.SetWidth(width)
	}
	return model
}

func dragQuerySelection(t *testing.T, model Model, startCol, startRow, endCol, endRow int) Model {
	t.Helper()
	left, top := queryEditorOrigin(model.layout)
	updated, _ := updateModel(t, model, tea.MouseClickMsg{
		X: left + startCol, Y: top + startRow, Button: tea.MouseLeft,
	})
	updated, _ = updateModel(t, updated, tea.MouseMotionMsg{X: left + endCol, Y: top + endRow})
	updated, _ = updateModel(t, updated, tea.MouseReleaseMsg{
		X: left + endCol, Y: top + endRow, Button: tea.MouseLeft,
	})
	return updated
}
