package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestUpdateKeyRouting(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Model)
		message tea.KeyPressMsg
		assert  func(*testing.T, Model, tea.Cmd)
	}{
		{
			name:    "opens connection modal",
			message: keyPress('n', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				require.NotNil(t, got.modal)
				assert.True(t, got.creatingConnection)
				assert.Equal(t, -1, got.editingConnection)
			},
		},
		{
			name:    "opens connections modal",
			message: keyPress('l', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				require.NotNil(t, got.connectionsModal)
			},
		},
		{
			name:    "switches to query panel",
			message: keyPress('r', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, panelQuery, got.panel)
				assert.Equal(t, focusData, got.focus)
				assert.True(t, got.query.editor.Focused())
			},
		},
		{
			name:    "switches to data panel",
			setup:   func(model *Model) { model.panel = panelQuery },
			message: keyPress('t', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, panelData, got.panel)
				assert.Equal(t, focusData, got.focus)
			},
		},
		{
			name:    "starts navigator search",
			setup:   func(model *Model) { model.database = &fakeDatabase{name: "chinook"} },
			message: keyPress('f', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, focusNavigator, got.focus)
				assert.True(t, got.navigator.searching)
				assert.True(t, got.navigator.filter.Focused())
			},
		},
		{
			name: "opens objects modal",
			setup: func(model *Model) {
				model.database = &fakeDatabase{name: "chinook", engine: db.EnginePostgreSQL}
				model.navigator.setFunctionsAvailable(true)
			},
			message: keyPress('o', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				require.NotNil(t, got.databaseExplorerModal)
			},
		},
		{
			name: "does not quit when query editor contains q",
			setup: func(model *Model) {
				model.panel = panelQuery
				_ = model.query.focusEditor()
			},
			message: keyPress('q', "q", 0),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, "q", got.query.editor.Value())
			},
		},
		{
			name:    "ignores dump without database",
			message: keyPress('d', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, command tea.Cmd) {
				assert.Nil(t, command)
				assert.Nil(t, got.dumpModal)
			},
		},
		{
			name: "opens actions modal from raw query mode",
			setup: func(model *Model) {
				model.database = &fakeDatabase{name: "chinook", ddl: "CREATE TABLE public.\"Album\" ();"}
				model.navigator.tables = []db.Table{{Name: "Album"}}
				model.panel = panelQuery
				model.activeConnectionIndex = 0
				model.config = config.Config{
					Connections: []config.Connection{
						{Name: "Test", Engine: "postgres"},
					},
				}
			},
			message: keyPress('g', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, command tea.Cmd) {
				require.NotNil(t, got.actionsModal)
				assert.Equal(t, "Album", got.actionsModal.tableName)
				assert.Nil(t, command)
			},
		},
		{
			name:    "tab moves from table list to data list",
			message: keyPress(tea.KeyTab, "", 0),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, focusData, got.focus)
				assert.False(t, got.navigator.searching)
				assert.False(t, got.navigator.filter.Focused())
			},
		},
		{
			name:    "tab moves from table search to data list",
			setup:   func(model *Model) { _ = model.navigator.startSearch() },
			message: keyPress(tea.KeyTab, "", 0),
			assert: func(t *testing.T, got Model, command tea.Cmd) {
				assert.Nil(t, command)
				assert.Equal(t, focusData, got.focus)
				assert.False(t, got.navigator.searching)
				assert.False(t, got.navigator.filter.Focused())
			},
		},
		{
			name:    "tab wraps from data list to table list",
			setup:   func(model *Model) { model.focus = focusData },
			message: keyPress(tea.KeyTab, "", 0),
			assert: func(t *testing.T, got Model, command tea.Cmd) {
				assert.Nil(t, command)
				assert.Equal(t, focusNavigator, got.focus)
				assert.False(t, got.navigator.searching)
			},
		},
		{
			name: "tab preserves query editor results toggle",
			setup: func(model *Model) {
				model.panel = panelQuery
				_ = model.query.focusEditor()
			},
			message: keyPress(tea.KeyTab, "", 0),
			assert: func(t *testing.T, got Model, command tea.Cmd) {
				assert.Nil(t, command)
				assert.True(t, got.query.resultsFocused)
				assert.False(t, got.query.editor.Focused())
			},
		},
		{
			name: "results navigation takes precedence over stale table completion",
			setup: func(model *Model) {
				model.panel = panelQuery
				model.query.result = db.QueryResult{Rows: [][]any{{1}, {2}}}
				model.query.resultsFocused = true
				model.query.completion.visible = true
			},
			message: keyPress(tea.KeyDown, "", 0),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, 1, got.query.viewport)
			},
		},
		{
			name: "does not open table completion while tables load",
			setup: func(model *Model) {
				model.database = &fakeDatabase{engine: db.EnginePostgreSQL}
				model.panel = panelQuery
				model.loading = true
				model.navigator.tables = []db.Table{{Name: "Album"}}
				model.query.editor.SetValue("SELECT * FROM ")
				_ = model.query.focusEditor()
			},
			message: keyPress('a', "a", 0),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.False(t, got.query.completion.visible)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			if test.setup != nil {
				test.setup(&model)
			}

			got, command := updateModel(t, model, test.message)
			test.assert(t, got, command)
		})
	}
}

func TestUpdateQueryEditorOpensTableCompletionForEveryEngine(t *testing.T) {
	for _, engine := range []string{
		db.EnginePostgreSQL,
		db.EngineMySQL,
		db.EngineOracle,
		db.EngineSQLite,
		db.EngineSQLServer,
	} {
		t.Run(engine, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = &fakeDatabase{engine: engine}
			model.panel = panelQuery
			model.navigator.tables = []db.Table{{Name: "Album"}}
			model.query.editor.SetValue("SELECT * FROM ")
			_ = model.query.focusEditor()

			got, command := updateModel(t, model, keyPress('a', "a", 0))

			assert.Nil(t, command)
			require.True(t, got.query.completion.visible)
			assert.Equal(t, "Album", got.query.completion.matches[0].Name)
		})
	}
}
