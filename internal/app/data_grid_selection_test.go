package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestDataGridTruncatesLongValuesIntoSingleLineCells(t *testing.T) {
	layout := newAppLayout(64, 16)
	data := dataModel{page: db.RowPage{
		Columns: []string{"id", "password"},
		Rows:    [][]any{{1, strings.Repeat("x", 80)}},
	}}
	status := dataStatus{tableName: "credentials"}
	gridTop := data.gridTop(data.title(status, layout), layout)
	grid, bounds, ok := data.visibleDataGrid(layout, gridTop)
	require.True(t, ok)
	assert.Contains(t, grid, "…")
	assert.NotContains(t, grid, strings.Repeat("x", 80))
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
	assert.Equal(t, region.Top, region.Bottom)

	assert.True(t, data.beginTextSelection(startX, startY, layout, gridTop))
	text, copied := data.finishTextSelection(bounds.x+region.Right-1, bounds.y+region.Top, layout, gridTop)

	assert.True(t, copied)
	assert.NotContains(t, text, "\n")
	assert.Contains(t, text, "…")
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

func TestDataGridColumnResizeHonorsDraggedWidth(t *testing.T) {
	testCases := []struct {
		name           string
		id             any
		expectedHeader string
	}{
		{
			name:           "minimum-width column",
			id:             1,
			expectedHeader: "│ id ↔ │",
		},
		{
			name:           "column expanded by a value",
			id:             strings.Repeat("0", 10),
			expectedHeader: "│ id       ↔ │",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			layout := newAppLayout(100, 24)
			data := dataModel{page: db.RowPage{
				Columns: []string{"id", "email", "name"},
				Rows:    [][]any{{testCase.id, "x", "a"}},
			}}
			status := dataStatus{tableName: "people", active: true}
			gridTop := data.gridTop(data.title(status, layout), layout)
			grid, bounds, ok := data.visibleDataGrid(layout, gridTop)
			require.True(t, ok)
			assert.Contains(t, grid, testCase.expectedHeader)
			firstColumn, lastColumn := data.visibleColumnRange(layout.data.width)
			widths := data.dataColumnWidths(layout.data.width, firstColumn, lastColumn)
			require.GreaterOrEqual(t, len(widths), 2)

			dividerX := bounds.x + 1 + widths[0]
			handleX := dividerX - lipgloss.Width(dataColumnResizeHandle)
			assert.True(t, data.beginColumnResize(handleX, bounds.y+1, layout, gridTop))
			assert.False(t, data.selection.Dragging())
			assert.Contains(t, data.title(status, layout), "resizing id")
			assert.True(t, data.resizeColumn(dividerX+3, layout))
			assert.True(t, data.finishColumnResize())

			resizedWidths := data.dataColumnWidths(layout.data.width, firstColumn, lastColumn)
			assert.Equal(t, widths[0]+5, resizedWidths[0])
			assert.Equal(t, widths[1], resizedWidths[1])
		})
	}
}

func TestDataPanelHeaderDividerDragResizesColumnInsteadOfSelectingText(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.activeRelation = activeRelation{item: navigatorItem{name: "people", section: navigatorTables}, set: true}
	model.data.page = db.RowPage{
		Columns: []string{"id", "email"},
		Rows:    [][]any{{1, "person@example.com"}},
	}
	gridTop := model.dataGridTop()
	_, bounds, ok := model.data.visibleDataGrid(model.layout, gridTop)
	require.True(t, ok)
	firstColumn, lastColumn := model.data.visibleColumnRange(model.layout.data.width)
	widths := model.data.dataColumnWidths(model.layout.data.width, firstColumn, lastColumn)
	dividerX := bounds.x + 1 + widths[0]

	updated, command := updateModel(t, model, tea.MouseClickMsg{X: dividerX, Y: bounds.y + 1, Button: tea.MouseLeft})
	require.Nil(t, command)
	require.NotNil(t, updated.data.resizing)
	assert.False(t, updated.data.selection.Dragging())

	updated, command = updateModel(t, updated, tea.MouseMotionMsg{X: dividerX + 4, Y: bounds.y + 1})
	require.Nil(t, command)
	updated, command = updateModel(t, updated, tea.MouseReleaseMsg{X: dividerX + 4, Y: bounds.y + 1, Button: tea.MouseLeft})
	require.Nil(t, command)
	assert.Nil(t, updated.data.resizing)
	assert.Equal(t, widths[0]+4, updated.data.columnWidths[0])
}

func TestDataGridColumnResizeRespectsColumnAndPanelBounds(t *testing.T) {
	layout := newAppLayout(64, 16)
	data := dataModel{page: db.RowPage{
		Columns: []string{"identifier", "email"},
		Rows:    [][]any{{1, "person@example.com"}},
	}}
	status := dataStatus{tableName: "people", active: true}
	gridTop := data.gridTop(data.title(status, layout), layout)
	_, bounds, ok := data.visibleDataGrid(layout, gridTop)
	require.True(t, ok)
	firstColumn, lastColumn := data.visibleColumnRange(layout.data.width)
	widths := data.dataColumnWidths(layout.data.width, firstColumn, lastColumn)
	dividerX := bounds.x + 1 + widths[0]
	require.True(t, data.beginColumnResize(dividerX, bounds.y+1, layout, gridTop))

	require.True(t, data.resizeColumn(dividerX-100, layout))
	assert.Equal(t, data.minimumColumnWidth(0), data.columnWidths[0])
	require.True(t, data.resizeColumn(dividerX+100, layout))
	assert.Equal(t, tableWidth(layout.data.width)-tableOuterBorderWidth, data.columnWidths[0])

	firstColumn, lastColumn = data.visibleColumnRange(layout.data.width)
	assert.Equal(t, 0, firstColumn)
	assert.Equal(t, 1, lastColumn)
}
