package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestDDLModalWrapsAndScrollsSQL(t *testing.T) {
	layout := newAppLayout(64, 16)
	modal := newDDLModal(db.Table{Schema: "public", Name: "Album"})
	modal.finish("CREATE TABLE public.\"Album\" (\n\t\"AlbumId\" int4 NOT NULL,\n\t\"Title\" varchar(160) NOT NULL\n);", nil, layout)

	assert.Contains(t, modal.view(layout, "⠋"), "CREATE TABLE public.")
	assert.Contains(t, modal.view(layout, "⠋"), "DDL · public.Album")
	assert.Contains(t, modal.view(layout, "⠋"), "\"AlbumId\" int4 NOT NULL")
	assert.Contains(t, modal.view(layout, "⠋"), "Esc close")

	modal.scroll(100, layout)
	assert.GreaterOrEqual(t, modal.offset, 0)
	modal.scroll(-100, layout)
	assert.Zero(t, modal.offset)
}

func TestDDLModalCopiesOriginalSQL(t *testing.T) {
	model := Model{
		layout: newAppLayout(80, 24),
		ddlModal: &ddlModal{
			sql:    "CREATE TABLE public.\"Album\" (\n    \"Title\" varchar(160)\n);",
			offset: 1,
		},
	}
	assert.Contains(t, model.ddlModal.view(model.layout, "⠋"), "c copy")

	updated, command := updateModel(t, model, keyPress('c', "c", 0))

	assert.NotNil(t, command)
	assert.Equal(t, model.ddlModal.sql, fmt.Sprint(command()))
	assert.True(t, updated.ddlModal.copied)
	assert.Equal(t, 1, updated.ddlModal.offset)
	assert.Contains(t, updated.ddlModal.view(updated.layout, "⠋"), "Copied DDL")
}

func TestDDLModalDoesNotCopyIneligibleDDL(t *testing.T) {
	tests := map[string]ddlModal{
		"loading": {loading: true, sql: "CREATE TABLE example ();"},
		"error":   {sql: "CREATE TABLE example ();", err: errors.New("load failed")},
		"empty":   {},
	}

	for name, modal := range tests {
		t.Run(name, func(t *testing.T) {
			model := Model{layout: newAppLayout(80, 24), ddlModal: &modal}

			updated, command := updateModel(t, model, keyPress('c', "c", 0))

			assert.Nil(t, command)
			assert.False(t, updated.ddlModal.copied)
		})
	}
}
