package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type sqlScriptsLoadedMsg struct {
	connectionName string
	request        uint64
	scripts        []SqlScript
	err            error
}

type sqlScriptSavedMsg struct {
	session uint64
	request uint64
	err     error
}

type sqlScriptsModal struct {
	connectionName string
	request        uint64
	scripts        []SqlScript
	selected       int
	offset         int
	loading        bool
	loadErr        error
}

func newSQLScriptsModal(connectionName string, request uint64) sqlScriptsModal {
	return sqlScriptsModal{connectionName: connectionName, request: request, loading: true}
}

func (m *sqlScriptsModal) finish(scripts []SqlScript, err error, layout appLayout) {
	m.loading = false
	m.scripts = scripts
	m.loadErr = err
	m.clamp(layout)
}

func (m *sqlScriptsModal) move(delta int, layout appLayout) {
	m.selected += delta
	m.clamp(layout)
}

func (m *sqlScriptsModal) clamp(layout appLayout) {
	m.selected = min(max(m.selected, 0), max(0, len(m.scripts)-1))
	visibleRows := m.visibleRows(layout)
	m.offset = min(max(m.offset, 0), max(0, len(m.scripts)-visibleRows))
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visibleRows {
		m.offset = m.selected - visibleRows + 1
	}
}

func (m sqlScriptsModal) visibleRows(layout appLayout) int {
	return min(max(1, len(m.scripts)), m.maxVisibleRows(layout))
}

func (m sqlScriptsModal) maxVisibleRows(layout appLayout) int {
	return min(8, max(1, layout.height-12))
}

func (m sqlScriptsModal) view(layout appLayout) string {
	modalWidth := min(56, max(40, layout.width-8))
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Saved SQL scripts"), ""}
	switch {
	case m.loading:
		lines = append(lines, "Loading saved scripts…")
	case m.loadErr != nil:
		lines = append(lines, lipgloss.NewStyle().Foreground(colorError).Render("✕ "+sanitizeText(m.loadErr.Error())))
	case len(m.scripts) == 0:
		lines = append(lines, "No saved scripts")
	default:
		labelWidth := modalWidth - 8
		last := min(m.offset+m.visibleRows(layout), len(m.scripts))
		for index := m.offset; index < last; index++ {
			script := m.scripts[index]
			label := "(empty script)"
			for _, line := range strings.Split(script.content, "\n") {
				if strings.TrimSpace(line) != "" {
					label = strings.TrimSpace(line)
					break
				}
			}
			prefix := "  "
			style := lipgloss.NewStyle()
			if index == m.selected {
				prefix = "> "
				style = style.Bold(true).Foreground(colorTitle)
			}
			lines = append(lines, prefix+style.Render(truncateLabel(label, labelWidth)))
		}
	}
	footer := "Esc close"
	if !m.loading && m.loadErr == nil && len(m.scripts) > 0 {
		footer = "↑/↓ or j/k move  •  Enter load  •  Esc close"
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render(footer))
	style := lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive)
	if !m.loading && m.loadErr == nil && len(m.scripts) > m.maxVisibleRows(layout) {
		style = style.Height(m.maxVisibleRows(layout) + 4)
	}
	return style.Render(strings.Join(lines, "\n"))
}
