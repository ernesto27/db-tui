package app

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type ddlModal struct {
	table   db.Table
	loading bool
	sql     string
	err     error
	offset  int
	copied  bool
}

func newDDLModal(table db.Table) ddlModal {
	return ddlModal{table: table, loading: true}
}

func (m *ddlModal) finish(sql string, err error, layout appLayout) {
	m.loading = false
	m.sql = sql
	m.err = err
	m.offset = 0
	m.copied = false
	m.clamp(layout)
}

func (m *ddlModal) scroll(delta int, layout appLayout) {
	m.offset += delta
	m.clamp(layout)
}

func (m *ddlModal) clamp(layout appLayout) {
	m.offset = min(max(m.offset, 0), max(0, len(m.lines(layout))-m.visibleRows(layout)))
}

func (m ddlModal) view(layout appLayout, spinner string) string {
	modalWidth := min(100, max(40, layout.width-8))
	modalHeight := max(8, layout.height-6)
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("DDL · " + sanitizeText(m.table.Schema) + "." + sanitizeText(m.table.Name)), ""}
	switch {
	case m.loading:
		lines = append(lines, lipgloss.NewStyle().Foreground(colorAccent).Render(spinner+" Loading table DDL…"))
	case m.err != nil:
		lines = append(lines, lipgloss.NewStyle().Foreground(colorError).Render("✕ Unable to load DDL"), sanitizeText(m.err.Error()))
	default:
		ddlLines := m.lines(layout)
		last := min(m.offset+m.visibleRows(layout), len(ddlLines))
		lines = append(lines, ddlLines[m.offset:last]...)
	}
	footer := "↑/↓ scroll  •  PgUp/PgDn  •  Home/End  •  c copy  •  Esc close"
	if m.copied {
		footer = "↑/↓ scroll  •  PgUp/PgDn  •  Home/End  •  Copied DDL  •  Esc close"
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render(footer))
	return lipgloss.NewStyle().Width(modalWidth).Height(modalHeight).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(colorBorderActive).Background(colorModalBackground).Render(strings.Join(lines, "\n"))
}

func (m ddlModal) visibleRows(layout appLayout) int {
	return max(1, max(8, layout.height-6)-6)
}

func (m ddlModal) lines(layout appLayout) []string {
	width := max(1, min(100, max(40, layout.width-8))-6)
	source := strings.Split(sanitizeMultilineText(m.sql), "\n")
	lines := make([]string, 0, len(source))
	for _, line := range source {
		lines = append(lines, wrapDDLLine(line, width)...)
	}
	return lines
}

func wrapDDLLine(line string, width int) []string {
	if line == "" || width < 1 {
		return []string{line}
	}
	runes := []rune(line)
	lines := make([]string, 0, len(runes)/width+1)
	for len(runes) > width {
		lines = append(lines, string(runes[:width]))
		runes = runes[width:]
	}
	return append(lines, string(runes))
}
