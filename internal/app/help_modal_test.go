package app

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestHelpModalViewListsShortcutSectionsAndBindings(t *testing.T) {
	keys := defaultKeyMap()
	modal := newHelpModal(keys)

	view := modal.view(100)

	assert.Contains(t, view, "Keyboard shortcuts")
	assert.Contains(t, view, "General")
	assert.Contains(t, view, "Specific")
	assert.Contains(t, view, "Esc close")

	for _, shortcut := range append(modal.general, modal.specific...) {
		assert.Contains(t, view, shortcut.key)
		assert.Contains(t, view, shortcut.description)
	}
}

func TestHelpModalFitsNarrowLayoutWidth(t *testing.T) {
	modal := newHelpModal(defaultKeyMap())

	view := modal.view(64)

	assert.LessOrEqual(t, lipgloss.Width(view), 64)
	assert.LessOrEqual(t, lipgloss.Height(view), 24)
}
