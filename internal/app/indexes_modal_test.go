package app

import (
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestIndexesModalRendersAndScrolls(t *testing.T) {
	layout := newAppLayout(64, 16)
	modal := newIndexesModal("PostgresIndexExample")
	indexes := make([]db.IndexColumns, 0, 12)
	for index := 1; index <= 12; index++ {
		indexes = append(indexes, db.IndexColumns{
			Name:         "example_index_" + string(rune('A'+index-1)),
			Column:       "column_name",
			Table:        "PostgresIndexExample",
			AccessMethod: "btree",
		})
	}
	modal.finish(indexes, nil, layout)

	view := modal.view(layout, "⠋")
	assert.Contains(t, view, "Indexes · PostgresIndexExample")
	assert.Contains(t, view, "Index Name")
	assert.Contains(t, view, "Esc close")

	modal.scrollFields(1)
	assert.Equal(t, 1, modal.fieldOffset)
	assert.Contains(t, modal.view(layout, "⠋"), "Column")

	modal.scrollRows(100, layout)
	assert.Greater(t, modal.rowOffset, 0)
	modal.scrollRows(-100, layout)
	assert.Zero(t, modal.rowOffset)
}
