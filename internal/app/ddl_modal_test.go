package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDDLModalWrapsAndScrollsSQL(t *testing.T) {
	layout := newAppLayout(64, 16)
	modal := newDDLModal("Album")
	modal.finish("CREATE TABLE public.\"Album\" (\n\t\"AlbumId\" int4 NOT NULL,\n\t\"Title\" varchar(160) NOT NULL\n);", nil, layout)

	assert.Contains(t, modal.view(layout, "⠋"), "CREATE TABLE public.")
	assert.Contains(t, modal.view(layout, "⠋"), "\"AlbumId\" int4 NOT NULL")
	assert.Contains(t, modal.view(layout, "⠋"), "Esc close")

	modal.scroll(100, layout)
	assert.GreaterOrEqual(t, modal.offset, 0)
	modal.scroll(-100, layout)
	assert.Zero(t, modal.offset)
}
