package app

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"

	"github.com/ernestoponce27/db-tui/internal/app/textselection"
)

const (
	tableHorizontalPadding = 2
	tableOuterBorderWidth  = 2
	tableColumnBorderWidth = 1
)

var textSelectionStyle = lipgloss.NewStyle().
	Foreground(colorSelectionForeground).
	Background(colorSelectionBackground)

type dataGridBounds struct {
	x      int
	y      int
	width  int
	height int
}

func (m dataModel) visibleColumnRange(width int) (int, int) {
	if len(m.page.Columns) == 0 {
		return 0, 0
	}

	firstColumn := min(max(m.columnOffset, 0), len(m.page.Columns)-1)
	availableWidth := tableWidth(width) - tableOuterBorderWidth
	usedWidth := 0
	lastColumn := firstColumn

	for columnIndex := firstColumn; columnIndex < len(m.page.Columns); columnIndex++ {
		columnWidth := lipgloss.Width(sanitizeText(m.page.Columns[columnIndex])) + tableHorizontalPadding
		if columnIndex > firstColumn {
			columnWidth += tableColumnBorderWidth
		}
		if lastColumn > firstColumn && usedWidth+columnWidth > availableWidth {
			break
		}

		usedWidth += columnWidth
		lastColumn++
	}

	return firstColumn, lastColumn
}

func (m dataModel) visibleDataEnd(width, height, firstColumn, lastColumn, firstRow int) int {
	availableHeight := max(1, height-2)
	lastRow := firstRow
	for lastRow < len(m.page.Rows) {
		grid := m.dataGrid(width, firstColumn, lastColumn, firstRow, lastRow+1)
		if lipgloss.Height(grid.String()) > availableHeight {
			if lastRow == firstRow {
				return lastRow + 1
			}
			break
		}
		lastRow++
	}
	return lastRow
}

func (m dataModel) dataGrid(width, firstColumn, lastColumn, firstRow, lastRow int) *table.Table {
	headers := make([]string, 0, lastColumn-firstColumn)
	for _, column := range m.page.Columns[firstColumn:lastColumn] {
		headers = append(headers, sanitizeText(column))
	}
	columnWidths := m.dataColumnWidths(width, firstColumn, lastColumn)

	rows := make([][]string, 0, lastRow-firstRow)
	for _, row := range m.page.Rows[firstRow:lastRow] {
		values := make([]string, 0, lastColumn-firstColumn)
		for columnIndex := firstColumn; columnIndex < lastColumn; columnIndex++ {
			var value any
			if columnIndex < len(row) {
				value = row[columnIndex]
			}
			values = append(values, formatCell(value))
		}
		rows = append(rows, values)
	}

	return table.New().
		Headers(headers...).
		Rows(rows...).
		Width(totalTableWidth(columnWidths)).
		Wrap(true).
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorderInactive)).
		StyleFunc(func(row, column int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1).Width(columnWidths[column])
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(colorAccent)
			}
			if row == m.selected-firstRow {
				return style.Foreground(colorSelectionForeground).Background(colorSelectionBackground)
			}
			return style.Foreground(colorText)
		})
}

func (m dataModel) visibleDataGrid(layout appLayout, gridTop int) (string, dataGridBounds, bool) {
	if len(m.page.Columns) == 0 || len(m.page.Rows) == 0 {
		return "", dataGridBounds{}, false
	}

	firstColumn, lastColumn := m.visibleColumnRange(layout.data.width)
	firstVisibleRow := m.viewport
	lastVisibleRow := m.visibleDataEnd(layout.data.width, layout.data.height, firstColumn, lastColumn, firstVisibleRow)
	rendered := m.dataGrid(layout.data.width, firstColumn, lastColumn, firstVisibleRow, lastVisibleRow).String()
	return rendered, dataGridBounds{
		x:      layout.data.x + 2, // panel border and left padding
		y:      gridTop,
		width:  lipgloss.Width(rendered),
		height: lipgloss.Height(rendered),
	}, true
}

func (m *dataModel) beginTextSelection(x, y int, layout appLayout, gridTop int) bool {
	point, ok := m.selectionPointAt(x, y, layout, gridTop)
	if !ok {
		return false
	}
	region, ok := m.cellBoundsAt(point, layout)
	if !ok {
		return false
	}
	return m.selection.Start(point, region)
}

func (m *dataModel) extendTextSelection(x, y int, layout appLayout, gridTop int) bool {
	if !m.selection.Dragging() {
		return false
	}
	point, ok := m.selectionPointAt(x, y, layout, gridTop)
	if !ok {
		return false
	}
	return m.selection.Update(point)
}

func (m *dataModel) finishTextSelection(x, y int, layout appLayout, gridTop int) (string, bool) {
	if !m.selection.Dragging() {
		return "", false
	}

	point, ok := m.selectionPointAt(x, y, layout, gridTop)
	if !ok {
		m.clearTextSelection()
		return "", false
	}
	if !m.selection.Release(point) {
		return "", false
	}

	grid, _, ok := m.visibleDataGrid(layout, gridTop)
	if !ok {
		m.clearTextSelection()
		return "", false
	}
	return m.selection.Text(grid), true
}

func (m *dataModel) clearTextSelection() {
	m.selection.Clear()
}

func (m dataModel) selectionPointAt(x, y int, layout appLayout, gridTop int) (textselection.Point, bool) {
	_, bounds, ok := m.visibleDataGrid(layout, gridTop)
	if !ok || x <= bounds.x || x >= bounds.x+bounds.width-1 || y <= bounds.y || y >= bounds.y+bounds.height-1 {
		return textselection.Point{}, false
	}
	return textselection.Point{X: x - bounds.x, Y: y - bounds.y}, true
}

func (m dataModel) cellBoundsAt(point textselection.Point, layout appLayout) (textselection.Region, bool) {
	// The header separator is not a selectable row.
	if point.Y == 2 {
		return textselection.Region{}, false
	}

	firstColumn, lastColumn := m.visibleColumnRange(layout.data.width)
	columnWidths := m.dataColumnWidths(layout.data.width, firstColumn, lastColumn)
	cellLeft, cellRight := 0, 0
	foundColumn := false
	left := 1 // outer table border
	for _, width := range columnWidths {
		right := left + width
		if point.X >= left && point.X < right {
			cellLeft, cellRight, foundColumn = left, right-1, true
			break
		}
		left = right + tableColumnBorderWidth
	}
	if !foundColumn {
		return textselection.Region{}, false
	}

	if point.Y == 1 {
		return textselection.Region{Left: cellLeft, Right: cellRight, Top: 1, Bottom: 1}, true
	}

	firstVisibleRow := m.viewport
	lastVisibleRow := m.visibleDataEnd(layout.data.width, layout.data.height, firstColumn, lastColumn, firstVisibleRow)
	top := 3 // top border, header, and header separator
	for row := firstVisibleRow; row < lastVisibleRow; row++ {
		rowHeight := lipgloss.Height(m.dataGrid(layout.data.width, firstColumn, lastColumn, row, row+1).String()) - 4
		bottom := top + rowHeight - 1
		if point.Y >= top && point.Y <= bottom {
			return textselection.Region{Left: cellLeft, Right: cellRight, Top: top, Bottom: bottom}, true
		}
		top = bottom + 1
	}
	return textselection.Region{}, false
}

func (m dataModel) dataColumnWidths(width, firstColumn, lastColumn int) []int {
	columnWidths := make([]int, lastColumn-firstColumn)
	desiredWidths := make([]int, len(columnWidths))
	usedWidth := 0

	for columnIndex := firstColumn; columnIndex < lastColumn; columnIndex++ {
		index := columnIndex - firstColumn
		minimumWidth := lipgloss.Width(sanitizeText(m.page.Columns[columnIndex])) + tableHorizontalPadding
		columnWidths[index] = minimumWidth
		desiredWidths[index] = minimumWidth

		for _, row := range m.page.Rows {
			var value any
			if columnIndex < len(row) {
				value = row[columnIndex]
			}
			desiredWidths[index] = max(desiredWidths[index], lipgloss.Width(formatCell(value))+tableHorizontalPadding)
		}
		usedWidth += minimumWidth
	}

	availableWidth := tableWidth(width) - tableOuterBorderWidth - max(0, len(columnWidths)-1)*tableColumnBorderWidth
	remainingWidth := max(0, availableWidth-usedWidth)
	for remainingWidth > 0 {
		totalNeededWidth := 0
		for index := range columnWidths {
			totalNeededWidth += desiredWidths[index] - columnWidths[index]
		}
		if totalNeededWidth == 0 {
			break
		}

		for index := range columnWidths {
			neededWidth := desiredWidths[index] - columnWidths[index]
			if neededWidth == 0 || remainingWidth == 0 {
				continue
			}
			additionalWidth := max(1, remainingWidth*neededWidth/totalNeededWidth)
			additionalWidth = min(additionalWidth, neededWidth, remainingWidth)
			columnWidths[index] += additionalWidth
			remainingWidth -= additionalWidth
		}
	}

	return columnWidths
}

func totalTableWidth(columnWidths []int) int {
	width := tableOuterBorderWidth + max(0, len(columnWidths)-1)*tableColumnBorderWidth
	for _, columnWidth := range columnWidths {
		width += columnWidth
	}
	return width
}

func tableWidth(width int) int {
	return max(20, width-4)
}
