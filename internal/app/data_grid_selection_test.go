package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestDataGridSelectionKeepsWrappedTextWithinOneCell(t *testing.T) {
	layout := newAppLayout(64, 16)
	data := dataModel{page: db.RowPage{
		Columns: []string{"id", "password"},
		Rows:    [][]any{{1, strings.Repeat("x", 80)}},
	}}
	status := dataStatus{tableName: "credentials"}
	gridTop := data.gridTop(data.title(status, layout), layout)
	_, bounds, ok := data.visibleDataGrid(layout, gridTop)
	require.True(t, ok)
	firstColumn, lastColumn := data.visibleColumnRange(layout.data.width)
	widths := data.dataColumnWidths(layout.data.width, firstColumn, lastColumn)
	require.Len(t, widths, 2)
	passwordLeft := 1 + widths[0] + tableColumnBorderWidth

	startX := bounds.x + passwordLeft + 1
	startY := bounds.y + 3
	point, ok := data.selectionPointAt(startX, startY, layout, gridTop)
	require.True(t, ok)
	region, ok := data.cellBoundsAt(point, layout)
	require.True(t, ok)
	require.Greater(t, region.Bottom, region.Top)

	assert.True(t, data.beginTextSelection(startX, startY, layout, gridTop))
	assert.True(t, data.extendTextSelection(bounds.x+region.Right-1, bounds.y+region.Bottom, layout, gridTop))
	text, copied := data.finishTextSelection(bounds.x+region.Right-1, bounds.y+region.Bottom, layout, gridTop)

	assert.True(t, copied)
	assert.Contains(t, text, "\n")
	assert.NotContains(t, text, "1")
}

func TestDataGridTopAccountsForWrappedTitle(t *testing.T) {
	layout := newAppLayout(64, 16)
	data := dataModel{page: db.RowPage{
		Columns: []string{"id"},
		Rows:    [][]any{{1}},
	}}
	shortStatus := dataStatus{tableName: "people"}
	longStatus := dataStatus{tableName: strings.Repeat("very_long_relation_name_", 4)}

	shortTop := data.gridTop(data.title(shortStatus, layout), layout)
	longTop := data.gridTop(data.title(longStatus, layout), layout)

	assert.Greater(t, longTop, shortTop)
	_, bounds, ok := data.visibleDataGrid(layout, longTop)
	require.True(t, ok)
	assert.Equal(t, longTop, bounds.y)
}
