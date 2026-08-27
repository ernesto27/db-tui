package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditRowFieldDoesNotRenderDataType(t *testing.T) {
	modal := newEditRowModal(
		db.Table{Schema: "reporting", Name: "Album"},
		[]db.Column{{Name: "AlbumId", DataType: "int4"}},
		[]any{11},
	)
	contentWidth := 60
	inputWidth := editRowInputWidth(contentWidth)

	lines := modal.viewField(newEditRowStyles(), 0, contentWidth, inputWidth)

	assert.NotContains(t, ansi.Strip(strings.Join(lines, "\n")), "int4")
}

func TestEditRowFailureWrapsFullErrorMessage(t *testing.T) {
	message := "update PostgreSQL row: ERROR: invalid input syntax for type integer: \"not a number\" (SQLSTATE 22P02)"
	layout := newAppLayout(64, 24)
	modal := editRowModal{
		table: db.Table{Name: "Album"},
		state: editRowFailed,
	}

	view := ansi.Strip(modal.viewStatus(layout, message, colorError))

	assert.NotContains(t, view, "…")
	for _, word := range strings.Fields(message) {
		assert.Contains(t, view, word)
	}
	for _, line := range strings.Split(view, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), editRowContentWidth(layout.width)+6)
	}
}

func TestEditRowRequiresConfirmationBeforeSaving(t *testing.T) {
	modal := newEditRowModal(
		db.Table{Schema: "reporting", Name: "Album"},
		[]db.Column{
			{Name: "AlbumId", IsPrimaryKey: true},
			{Name: "Title"},
		},
		[]any{1, "Before"},
	)
	modal.fields[1].input.SetValue("After")

	confirming, command := modal.update(keyPress(tea.KeyEnter, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, editRowConfirming, confirming.state)
	assert.Equal(t, map[string]any{"Title": "After"}, confirming.setColumns)
	assert.Equal(t, map[string]any{"AlbumId": 1}, confirming.whereColumns)
	assert.Contains(t, ansi.Strip(confirming.view(newAppLayout(80, 24))), "Save changes to this row?")

	editing, command := confirming.update(keyPress(tea.KeyEscape, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, editRowEditing, editing.state)

	confirming, command = editing.update(keyPress(tea.KeyEnter, "", 0))
	require.Nil(t, command)
	saving, command := confirming.update(keyPress(tea.KeyEnter, "", 0))

	require.NotNil(t, command)
	assert.Equal(t, editRowSaving, saving.state)
	assert.Equal(t, editRowSaveMsg{
		table:        db.Table{Schema: "reporting", Name: "Album"},
		setColumns:   map[string]any{"Title": "After"},
		whereColumns: map[string]any{"AlbumId": 1},
	}, command())
}

func TestEditRowRejectsTablesWithoutPrimaryKeys(t *testing.T) {
	modal := newEditRowModal(
		db.Table{Name: "audit_log"},
		[]db.Column{{Name: "Message"}},
		[]any{"Before"},
	)
	modal.fields[0].input.SetValue("After")

	updated, command := modal.prepareSave()

	assert.Nil(t, command)
	assert.Equal(t, editRowFailed, updated.state)
	assert.EqualError(t, updated.err, "cannot edit row: table has no primary key")
	assert.Empty(t, updated.setColumns)
	assert.Empty(t, updated.whereColumns)
}
