package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
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
			message: keyPress('f', "", tea.ModCtrl),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, focusNavigator, got.focus)
				assert.True(t, got.navigator.searching)
				assert.True(t, got.navigator.filter.Focused())
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
			name:    "tab starts table search from table list",
			message: keyPress(tea.KeyTab, "", 0),
			assert: func(t *testing.T, got Model, _ tea.Cmd) {
				assert.Equal(t, focusNavigator, got.focus)
				assert.True(t, got.navigator.searching)
				assert.True(t, got.navigator.filter.Focused())
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
