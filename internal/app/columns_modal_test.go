package app

import (
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestColumnsModalRendersAndScrolls(t *testing.T) {
	layout := newAppLayout(64, 16)
	modal := newColumnsModal("Album")
	columns := make([]db.Column, 0, 12)
	for index := 1; index <= 12; index++ {
		columns = append(columns, db.Column{
			Name:            "Column" + string(rune('A'+index-1)),
			OrdinalPosition: index,
			DataType:        "varchar(160)",
			Collation:       "default",
			NotNull:         true,
		})
	}
	modal.finish(columns, nil, layout)
	fields := modal.visibleFields(114)
	assert.Equal(t, "Default", fields[len(fields)-1].title)

	view := modal.view(layout, "⠋")
	assert.Contains(t, view, "Columns · Album")
	assert.Contains(t, view, "Column Name")
	assert.Contains(t, view, "Esc close")

	modal.scrollFields(1)
	assert.Equal(t, 1, modal.fieldOffset)
	assert.Contains(t, modal.view(layout, "⠋"), "Data type")
	modal.scrollFields(4)
	assert.Contains(t, modal.view(layout, "⠋"), "[v]")

	modal.scrollRows(100, layout)
	assert.Greater(t, modal.rowOffset, 0)
	modal.scrollRows(-100, layout)
	assert.Zero(t, modal.rowOffset)
}
