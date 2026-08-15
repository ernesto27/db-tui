package app

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type indexesModal struct {
	tableName   string
	loading     bool
	indexes     []db.IndexColumns
	err         error
	rowOffset   int
	fieldOffset int
}

type indexGridField struct {
	title string
	width int
	value func(db.IndexColumns) string
}

var indexGridFields = []indexGridField{
	{title: "Index Name", width: 34, value: func(index db.IndexColumns) string { return index.Name }},
	{title: "Column", width: 24, value: func(index db.IndexColumns) string { return index.Column }},
	{title: "Table", width: 24, value: func(index db.IndexColumns) string { return index.Table }},
	{title: "Access Method", width: 16, value: func(index db.IndexColumns) string { return index.AccessMethod }},
}

func newIndexesModal(tableName string) indexesModal {
	return indexesModal{
		tableName: tableName,
		loading:   true,
	}
}

func (m *indexesModal) finish(indexes []db.IndexColumns, err error, layout appLayout) {
	m.loading = false
	m.indexes = indexes
	m.err = err
	m.rowOffset = 0
	m.fieldOffset = 0
	m.clamp(layout)
}

func (m *indexesModal) scrollRows(delta int, layout appLayout) {
	m.rowOffset += delta
	m.clamp(layout)
}

func (m *indexesModal) scrollFields(delta int) {
	m.fieldOffset = min(max(m.fieldOffset+delta, 0), len(indexGridFields)-1)
}

func (m *indexesModal) clamp(layout appLayout) {
	m.rowOffset = min(max(m.rowOffset, 0), max(0, len(m.indexes)-m.visibleRows(layout)))
	m.fieldOffset = min(max(m.fieldOffset, 0), len(indexGridFields)-1)
}

func (m indexesModal) view(layout appLayout, spinner string) string {
	modalWidth := min(120, max(40, layout.width-8))
	modalHeight := max(8, layout.height-6)

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTitle).
			Render("Indexes · " + sanitizeText(m.tableName)),
		"",
	}

	switch {
	case m.loading:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(colorAccent).
				Render(spinner+" Loading indexes…"),
		)
	case m.err != nil:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(colorError).
				Render("✕ Unable to load indexes"),
			sanitizeText(m.err.Error()),
		)
	case len(m.indexes) == 0:
		lines = append(lines, "No indexes found.")
	default:
		fields := m.visibleFields(modalWidth - 6)
		lines = append(lines, m.renderHeader(fields), m.renderDivider(fields))

		last := min(m.rowOffset+m.visibleRows(layout), len(m.indexes))
		for _, index := range m.indexes[m.rowOffset:last] {
			lines = append(lines, m.renderRow(index, fields))
		}
	}

	lines = append(lines, "",
		lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Render("←/→ fields  •  ↑/↓ rows  •  PgUp/PgDn  •  Home/End  •  Esc close"),
	)

	return lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m indexesModal) visibleRows(layout appLayout) int {
	return max(1, max(8, layout.height-6)-8)
}

func (m indexesModal) visibleFields(width int) []indexGridField {
	width = max(1, width)

	fields := make([]indexGridField, 0, len(indexGridFields)-m.fieldOffset)
	used := 0
	for _, field := range indexGridFields[m.fieldOffset:] {
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

func (m indexesModal) renderHeader(fields []indexGridField) string {
	values := make([]string, 0, len(fields))
	style := lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Background(colorModalBackground)
	for _, field := range fields {
		values = append(values, style.Width(field.width).Render(truncateLabel(field.title, field.width)))
	}
	separator := lipgloss.NewStyle().Background(colorModalBackground).Render(" │ ")
	return strings.Join(values, separator)
}

func (m indexesModal) renderDivider(fields []indexGridField) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		values = append(values, strings.Repeat("─", field.width))
	}
	return strings.Join(values, "─┼─")
}

func (m indexesModal) renderRow(index db.IndexColumns, fields []indexGridField) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		value := truncateLabel(sanitizeText(field.value(index)), field.width)
		values = append(values, lipgloss.NewStyle().Width(field.width).Render(value))
	}
	return strings.Join(values, " │ ")
}
