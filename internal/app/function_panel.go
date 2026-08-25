package app

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type activeFunction struct {
	function db.FunctionColumns
	offset   int
	set      bool
}

func (m *activeFunction) activate(function db.FunctionColumns, layout appLayout) {
	m.function = function
	m.offset = 0
	m.set = true
	m.clamp(layout)
}

func (m *activeFunction) scroll(delta int, layout appLayout) {
	m.offset += delta
	m.clamp(layout)
}

func (m *activeFunction) clamp(layout appLayout) {
	m.offset = min(max(m.offset, 0), max(0, len(m.lines(layout))-m.visibleRows(layout)))
}

func (m activeFunction) visibleRows(layout appLayout) int {
	return max(1, layout.data.height-4)
}

func (m activeFunction) lines(layout appLayout) []string {
	contentWidth := max(1, layout.data.width-4)
	widths := functionColumnWidths(contentWidth)
	fields := []struct {
		label string
		value string
	}{
		{label: "Arguments", value: m.function.Arguments},
		{label: "Return type", value: m.function.ReturnType},
		{label: "Definition", value: m.function.Definition},
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render(sanitizeText(m.function.Name)),
		"",
		m.tableHeader(fields, widths),
		m.tableDivider(widths),
	}

	values := make([][]string, len(fields))
	rowCount := 0
	for index, field := range fields {
		value := sanitizeMultilineText(field.value)
		if value == "" {
			value = "—"
		}
		values[index] = strings.Split(ansi.Wrap(value, widths[index], ""), "\n")
		rowCount = max(rowCount, len(values[index]))
	}
	for row := range rowCount {
		lines = append(lines, m.tableRow(values, widths, row))
	}
	return lines
}

func functionColumnWidths(contentWidth int) []int {
	available := max(3, contentWidth-6) // two " │ " separators
	arguments := max(1, available/4)
	returnType := max(1, available/5)
	definition := max(1, available-arguments-returnType)
	return []int{arguments, returnType, definition}
}

func (m activeFunction) tableHeader(fields []struct {
	label string
	value string
}, widths []int) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
	values := make([]string, len(fields))
	for index, field := range fields {
		values[index] = style.Width(widths[index]).Render(truncateLabel(field.label, widths[index]))
	}
	return strings.Join(values, " │ ")
}

func (m activeFunction) tableDivider(widths []int) string {
	values := make([]string, len(widths))
	for index, width := range widths {
		values[index] = strings.Repeat("─", width)
	}
	return strings.Join(values, "─┼─")
}

func (m activeFunction) tableRow(values [][]string, widths []int, row int) string {
	columns := make([]string, len(widths))
	style := lipgloss.NewStyle()
	for index, width := range widths {
		value := ""
		if row < len(values[index]) {
			value = values[index][row]
		}
		columns[index] = style.Width(width).Render(value)
	}
	return strings.Join(columns, " │ ")
}

func (m activeFunction) view(layout appLayout, focused bool) string {
	lines := m.lines(layout)
	last := min(m.offset+m.visibleRows(layout), len(lines))
	content := strings.Join(lines[m.offset:last], "\n")
	return panelStyle(layout.data.width, layout.data.height, focused).Render(content)
}
