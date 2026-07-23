package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type navigatorModel struct {
	tables    []db.Table
	filter    textinput.Model
	searching bool
	selected  int
	offset    int
}

type navigatorStatus struct {
	databaseName string
	loading      bool
	tableLoadErr error
}

func newNavigatorModel() navigatorModel {
	filter := textinput.New()
	filter.Prompt = "Filter: "
	filter.Placeholder = "Ctrl+F"
	filter.SetWidth(14)
	return navigatorModel{filter: filter}
}

func (m *navigatorModel) reset() {
	filter := m.filter
	filter.SetValue("")
	filter.Blur()
	*m = navigatorModel{filter: filter}
}

func (m *navigatorModel) setTables(tables []db.Table) {
	m.reset()
	m.tables = tables
}

func (m *navigatorModel) resize(layout appLayout) {
	contentWidth := layout.navigator.width - 4 // borders and horizontal padding
	m.filter.SetWidth(max(1, contentWidth-lipgloss.Width(m.filter.Prompt)))
}

func (m navigatorModel) selectedTable() (db.Table, bool) {
	tables := m.visibleTables()
	if len(tables) == 0 || m.selected < 0 || m.selected >= len(tables) {
		return db.Table{}, false
	}
	return tables[m.selected], true
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
	tables := m.visibleTables()
	if len(tables) == 0 {
		m.selected = 0
		m.offset = 0
		return false
	}
	previous := m.selected
	m.selected = min(max(index, 0), len(tables)-1)
	m.ensureVisible(visibleRows)
	return m.selected != previous
}

func (m *navigatorModel) ensureVisible(visibleRows int) {
	visibleRows = max(1, visibleRows)
	tables := m.visibleTables()
	if len(tables) == 0 {
		m.selected = 0
		m.offset = 0
		return
	}
	m.selected = min(max(m.selected, 0), len(tables)-1)
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visibleRows {
		m.offset = m.selected - visibleRows + 1
	}
	m.offset = min(max(m.offset, 0), max(0, len(tables)-visibleRows))
}

func (m navigatorModel) tableAtMouse(msg tea.MouseClickMsg, layout appLayout) (int, bool) {
	tables := m.visibleTables()
	visibleIndex := msg.Y - layout.navigatorListY
	index := m.offset + visibleIndex
	valid := msg.Button == tea.MouseLeft &&
		layout.clickableNavigatorX(msg.X) &&
		visibleIndex >= 0 && visibleIndex < layout.navigatorListRows &&
		index >= 0 && index < len(tables)
	return index, valid
}

func (m navigatorModel) view(status navigatorStatus, layout appLayout, focused bool) string {
	tables := m.visibleTables()
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("● " + sanitizeText(status.databaseName)), ""}
	lines = append(lines, m.filter.View())
	switch {
	case status.loading:
		lines = append(lines, "Loading tables…")
	case status.tableLoadErr != nil:
		lines = append(lines, "Unable to load tables", truncateLabel(status.tableLoadErr.Error(), max(1, layout.navigator.width-4)))
	case len(m.tables) == 0:
		lines = append(lines, "No public tables.")
	case len(tables) == 0:
		lines = append(lines, "No matching tables.")
	default:
		title := fmt.Sprintf("TABLES (%d)", len(tables))
		if m.hasFilter() {
			title = fmt.Sprintf("TABLES (%d/%d)", len(tables), len(m.tables))
		}
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Render(title))
		lastVisible := min(m.offset+layout.navigatorListRows, len(tables))
		for index := m.offset; index < lastVisible; index++ {
			itemWidth := max(1, layout.navigator.width-4)
			marker := "  "
			style := lipgloss.NewStyle().Width(itemWidth)
			if index == m.selected {
				marker = "> "
				style = style.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
			}
			lines = append(lines, style.Render(marker+truncateLabel(tables[index].Name, max(0, itemWidth-len(marker)))))
		}
	}
	return panelStyle(layout.navigator.width, layout.navigator.height, focused).Render(strings.Join(lines, "\n"))
}

func (m navigatorModel) visibleTables() []db.Table {
	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if query == "" {
		return m.tables
	}

	tables := make([]db.Table, 0, len(m.tables))
	for _, table := range m.tables {
		if strings.Contains(strings.ToLower(table.Name), query) {
			tables = append(tables, table)
		}
	}
	return tables
}

func (m navigatorModel) hasFilter() bool {
	return strings.TrimSpace(m.filter.Value()) != ""
}

func (m *navigatorModel) startSearch() tea.Cmd {
	m.searching = true
	return m.filter.Focus()
}

func (m *navigatorModel) finishSearch() {
	m.searching = false
	m.filter.Blur()
}

func (m *navigatorModel) cancelSearch(visibleRows int) bool {
	selected, hasSelected := m.selectedTable()
	m.filter.SetValue("")
	m.finishSearch()
	return m.normalizeSelection(selected, hasSelected, visibleRows)
}

func (m *navigatorModel) updateFilter(msg tea.Msg, visibleRows int) (bool, tea.Cmd) {
	selected, hasSelected := m.selectedTable()
	previousQuery := m.filter.Value()
	var command tea.Cmd
	m.filter, command = m.filter.Update(msg)
	if m.filter.Value() == previousQuery {
		return false, command
	}
	return m.normalizeSelection(selected, hasSelected, visibleRows), command
}

func (m *navigatorModel) normalizeSelection(selected db.Table, hasSelected bool, visibleRows int) bool {
	tables := m.visibleTables()
	if len(tables) == 0 {
		changed := hasSelected
		m.selected = 0
		m.offset = 0
		return changed
	}

	if hasSelected {
		for index, table := range tables {
			if table.Name == selected.Name {
				m.selected = index
				m.ensureVisible(visibleRows)
				return false
			}
		}
	}

	m.selected = 0
	m.offset = 0
	m.ensureVisible(visibleRows)
	return true
}
