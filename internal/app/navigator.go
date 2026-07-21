package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type navigatorModel struct {
	tables   []db.Table
	selected int
	offset   int
}

type navigatorStatus struct {
	databaseName string
	startupErr   error
	loading      bool
	tableLoadErr error
}

func (m *navigatorModel) reset() {
	m.tables = nil
	m.selected = 0
	m.offset = 0
}

func (m *navigatorModel) setTables(tables []db.Table) {
	m.tables = tables
	m.selected = 0
	m.offset = 0
}

func (m navigatorModel) selectedTable() (db.Table, bool) {
	if len(m.tables) == 0 || m.selected < 0 || m.selected >= len(m.tables) {
		return db.Table{}, false
	}
	return m.tables[m.selected], true
}

func (m navigatorModel) selectedTableName() string {
	table, ok := m.selectedTable()
	if !ok {
		return ""
	}
	return sanitizeText(table.Name)
}

func (m *navigatorModel) move(delta, visibleRows int) bool {
	return m.selectIndex(m.selected+delta, visibleRows)
}

func (m *navigatorModel) selectIndex(index, visibleRows int) bool {
	if len(m.tables) == 0 {
		m.selected = 0
		m.offset = 0
		return false
	}
	previous := m.selected
	m.selected = min(max(index, 0), len(m.tables)-1)
	m.ensureVisible(visibleRows)
	return m.selected != previous
}

func (m *navigatorModel) ensureVisible(visibleRows int) {
	visibleRows = max(1, visibleRows)
	if len(m.tables) == 0 {
		m.selected = 0
		m.offset = 0
		return
	}
	m.selected = min(max(m.selected, 0), len(m.tables)-1)
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visibleRows {
		m.offset = m.selected - visibleRows + 1
	}
	m.offset = min(max(m.offset, 0), max(0, len(m.tables)-visibleRows))
}

func (m navigatorModel) tableAtMouse(msg tea.MouseClickMsg, layout appLayout) (int, bool) {
	visibleIndex := msg.Y - layout.navigatorListY
	index := m.offset + visibleIndex
	valid := msg.Button == tea.MouseLeft &&
		layout.clickableNavigatorX(msg.X) &&
		visibleIndex >= 0 && visibleIndex < layout.navigatorListRows &&
		index >= 0 && index < len(m.tables)
	return index, valid
}

func (m navigatorModel) view(status navigatorStatus, layout appLayout, focused bool) string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("● " + sanitizeText(status.databaseName)), ""}
	switch {
	case status.startupErr != nil:
		lines = append(lines, "Unable to start database session", truncateLabel(status.startupErr.Error(), max(1, layout.navigator.width-4)))
	case status.loading:
		lines = append(lines, "Loading tables…")
	case status.tableLoadErr != nil:
		lines = append(lines, "Unable to load tables", truncateLabel(status.tableLoadErr.Error(), max(1, layout.navigator.width-4)))
	case len(m.tables) == 0:
		lines = append(lines, "No public tables.")
	default:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Render(fmt.Sprintf("TABLES (%d)", len(m.tables))))
		lastVisible := min(m.offset+layout.navigatorListRows, len(m.tables))
		for index := m.offset; index < lastVisible; index++ {
			itemWidth := max(1, layout.navigator.width-4)
			marker := "  "
			style := lipgloss.NewStyle().Width(itemWidth)
			if index == m.selected {
				marker = "> "
				style = style.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
			}
			lines = append(lines, style.Render(marker+truncateLabel(m.tables[index].Name, max(0, itemWidth-len(marker)))))
		}
	}
	return panelStyle(layout.navigator.width, layout.navigator.height, focused).Render(strings.Join(lines, "\n"))
}
