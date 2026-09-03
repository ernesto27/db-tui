package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
)

func TestShortcutsModalLifecycle(t *testing.T) {
	tests := []struct {
		name  string
		close tea.KeyPressMsg
	}{
		{name: "escape", close: keyPress(tea.KeyEscape, "", 0)},
		{name: "shortcut toggle", close: keyPress('k', "", tea.ModCtrl)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			opened, command := updateModel(t, model, keyPress('k', "", tea.ModCtrl))

			require.NotNil(t, opened.shortcutsModal)
			assert.Nil(t, command)

			closed, command := updateModel(t, opened, test.close)
			assert.Nil(t, closed.shortcutsModal)
			assert.Nil(t, command)
		})
	}
}

func TestShortcutsModalCapturesInputAndScrolls(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newShortcutsModal(model.layout)
	model.shortcutsModal = &modal

	updated, command := updateModel(t, model, keyPress(tea.KeyDown, "", 0))

	require.NotNil(t, updated.shortcutsModal)
	assert.Equal(t, 1, updated.shortcutsModal.offset)
	assert.Equal(t, focusNavigator, updated.focus)
	assert.Nil(t, command)

	updated, _ = updateModel(t, updated, keyPress(tea.KeyEnd, "", 0))
	assert.Greater(t, updated.shortcutsModal.offset, 1)

	updated, _ = updateModel(t, updated, keyPress(tea.KeyHome, "", 0))
	assert.Zero(t, updated.shortcutsModal.offset)
}

func TestShortcutsModalViewShowsSectionedKeyTable(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.layout = newAppLayout(100, 60)
	modal := newShortcutsModal(model.layout)
	model.shortcutsModal = &modal

	view := model.View().Content

	assert.Contains(t, view, "Keyboard shortcuts")
	assert.Contains(t, view, "Global")
	assert.Contains(t, view, "Navigation")
	assert.Contains(t, view, "Tables and data")
	assert.Contains(t, view, "Query editor")
	assert.Contains(t, view, "Dialogs")
	assert.Contains(t, view, "Ctrl+K")
	assert.Contains(t, view, "Open or close keyboard shortcuts")
}

func TestFooterAdvertisesShortcuts(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)

	assert.Contains(t, model.footerText(), "Ctrl+K shortcuts")
}
