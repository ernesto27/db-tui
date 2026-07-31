package app

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type columnsModal struct {
	tableName   string
	loading     bool
	columns     []db.Column
	err         error
	rowOffset   int
	fieldOffset int
}

type columnGridField struct {
	title string
	width int
	value func(db.Column) string
}

var columnGridFields = []columnGridField{
	{title: "Column Name", width: 18, value: func(column db.Column) string { return column.Name }},
	{title: "#", width: 3, value: func(column db.Column) string { return strconv.Itoa(column.OrdinalPosition) }},
	{title: "Data type", width: 22, value: func(column db.Column) string { return column.DataType }},
	{title: "Identity", width: 14, value: func(column db.Column) string { return column.Identity }},
	{title: "Collation", width: 18, value: func(column db.Column) string { return column.Collation }},
	{title: "Not Null", width: 8, value: func(column db.Column) string {
		if column.NotNull {
			return "[v]"
		}
		return ""
	}},
	{title: "Default", width: 24, value: func(column db.Column) string { return column.Default }},
	{title: "Comment", width: 24, value: func(column db.Column) string { return column.Comment }},
}

func newColumnsModal(tableName string) columnsModal {
	return columnsModal{tableName: tableName, loading: true}
}

func (m *columnsModal) finish(columns []db.Column, err error, layout appLayout) {
	m.loading = false
	m.columns = columns
	m.err = err
	m.rowOffset = 0
	m.fieldOffset = 0
	m.clamp(layout)
}

func (m *columnsModal) scrollRows(delta int, layout appLayout) {
	m.rowOffset += delta
	m.clamp(layout)
}

func (m *columnsModal) scrollFields(delta int) {
	m.fieldOffset = min(max(m.fieldOffset+delta, 0), len(columnGridFields)-1)
}

func (m *columnsModal) clamp(layout appLayout) {
	m.rowOffset = min(max(m.rowOffset, 0), max(0, len(m.columns)-m.visibleRows(layout)))
	m.fieldOffset = min(max(m.fieldOffset, 0), len(columnGridFields)-1)
}

func (m columnsModal) view(layout appLayout, spinner string) string {
	modalWidth := min(120, max(40, layout.width-8))
	modalHeight := max(8, layout.height-6)
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Columns · " + sanitizeText(m.tableName)),
		"",
	}

	switch {
	case m.loading:
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(spinner+" Loading columns…"))
	case m.err != nil:
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("✕ Unable to load columns"), sanitizeText(m.err.Error()))
	case len(m.columns) == 0:
		lines = append(lines, "No columns found.")
	default:
		fields := m.visibleFields(modalWidth - 6)
		lines = append(lines, m.renderHeader(fields), m.renderDivider(fields))
		last := min(m.rowOffset+m.visibleRows(layout), len(m.columns))
		for _, column := range m.columns[m.rowOffset:last] {
			lines = append(lines, m.renderRow(column, fields))
		}
	}

	lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("←/→ fields  •  ↑/↓ rows  •  PgUp/PgDn  •  Home/End  •  Esc close"))
	return lipgloss.NewStyle().Width(modalWidth).Height(modalHeight).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Background(lipgloss.Color("235")).Render(strings.Join(lines, "\n"))
}

func (m columnsModal) visibleRows(layout appLayout) int {
	return max(1, max(8, layout.height-6)-8)
}

func (m columnsModal) visibleFields(width int) []columnGridField {
	width = max(1, width)
	fields := make([]columnGridField, 0, len(columnGridFields)-m.fieldOffset)
	used := 0
	for _, field := range columnGridFields[m.fieldOffset:] {
		required := field.width
		if len(fields) > 0 {
			required += 3
		}
		if used+required > width {
			remaining := width - used
			if len(fields) > 0 {
				remaining -= 3
			}
			if remaining > 0 {
				field.width = remaining
				fields = append(fields, field)
			}
			break
		}
		fields = append(fields, field)
		used += required
	}
	return fields
}

func (m columnsModal) renderHeader(fields []columnGridField) string {
	values := make([]string, 0, len(fields))
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("235"))
	for _, field := range fields {
		values = append(values, style.Width(field.width).Render(truncateLabel(field.title, field.width)))
	}
	separator := lipgloss.NewStyle().Background(lipgloss.Color("235")).Render(" │ ")
	return strings.Join(values, separator)
}

func (m columnsModal) renderDivider(fields []columnGridField) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		values = append(values, strings.Repeat("─", field.width))
	}
	return strings.Join(values, "─┼─")
}

func (m columnsModal) renderRow(column db.Column, fields []columnGridField) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		value := truncateLabel(sanitizeText(field.value(column)), field.width)
		values = append(values, lipgloss.NewStyle().Width(field.width).Render(value))
	}
	return strings.Join(values, " │ ")
}
