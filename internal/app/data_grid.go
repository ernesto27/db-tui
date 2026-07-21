package app

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

const (
	tableHorizontalPadding = 2
	tableOuterBorderWidth  = 2
	tableColumnBorderWidth = 1
)

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
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, column int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1).Width(columnWidths[column])
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(lipgloss.Color("86"))
			}
			if row == m.selected-firstRow {
				return style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
			}
			if row%2 == 1 {
				return style.Foreground(lipgloss.Color("252"))
			}
			return style.Foreground(lipgloss.Color("250"))
		})
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
