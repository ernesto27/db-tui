package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type navigatorSection uint8

const (
	navigatorTables navigatorSection = iota
	navigatorViews
	navigatorSectionCount
)

type navigatorCursor struct {
	selected int
	offset   int
}

type navigatorItem struct {
	name    string
	section navigatorSection
}

type navigatorModel struct {
	tables  []db.Table
	views   []db.View
	section navigatorSection
	cursors [navigatorSectionCount]navigatorCursor

	filter    textinput.Model
	searching bool
}

type navigatorStatus struct {
	databaseName  string
	tablesLoading bool
	tableLoadErr  error
	viewsLoading  bool
	viewLoadErr   error
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
	m.tables = tables
	m.normalizeSelection(1)
}

func (m *navigatorModel) setViews(views []db.View) {
	m.views = views
	m.normalizeSelection(1)
}

func (m *navigatorModel) resize(layout appLayout) {
	contentWidth := layout.navigator.width - 4 // borders and horizontal padding
	m.filter.SetWidth(max(1, contentWidth-lipgloss.Width(m.filter.Prompt)))
}

func (m navigatorModel) selectedItem() (navigatorItem, bool) {
	items := m.visibleItems()
	cursor := m.cursor()
	if len(items) == 0 || cursor.selected < 0 || cursor.selected >= len(items) {
		return navigatorItem{}, false
	}
	return items[cursor.selected], true
}

func (m navigatorModel) selectedTable() (db.Table, bool) {
	item, ok := m.selectedItem()
	if !ok || item.section != navigatorTables {
		return db.Table{}, false
	}
	return db.Table{Name: item.name}, true
}

func (m navigatorModel) selectedRowSource() (db.Table, bool) {
	item, ok := m.selectedItem()
	if !ok {
		return db.Table{}, false
	}
	return db.Table{Name: item.name}, true
}

func (m navigatorModel) selectedName() string {
	item, ok := m.selectedItem()
	if !ok {
		return ""
	}
	return sanitizeText(item.name)
}

func (m navigatorModel) selectedIsView() bool {
	item, ok := m.selectedItem()
	return ok && item.section == navigatorViews
}

func (m navigatorModel) hasSelection() bool {
	_, ok := m.selectedItem()
	return ok
}

func (m *navigatorModel) move(delta, visibleRows int) bool {
	return m.selectIndex(m.cursor().selected+delta, visibleRows)
}

func (m *navigatorModel) selectIndex(index, visibleRows int) bool {
	items := m.visibleItems()
	cursor := m.cursorPointer()
	if len(items) == 0 {
		cursor.selected = 0
		cursor.offset = 0
		return false
	}
	previous := cursor.selected
	cursor.selected = min(max(index, 0), len(items)-1)
	m.ensureVisible(visibleRows)
	return cursor.selected != previous
}

func (m *navigatorModel) switchSection(delta, visibleRows int) bool {
	if !m.hasViewsSection() {
		return false
	}

	previous := m.section
	if delta < 0 {
		m.section = navigatorTables
	} else {
		m.section = navigatorViews
	}
	m.ensureVisible(visibleRows)
	return previous != m.section
}

func (m *navigatorModel) ensureVisible(visibleRows int) {
	visibleRows = max(1, visibleRows)
	items := m.visibleItems()
	cursor := m.cursorPointer()
	if len(items) == 0 {
		cursor.selected = 0
		cursor.offset = 0
		return
	}
	cursor.selected = min(max(cursor.selected, 0), len(items)-1)
	if cursor.selected < cursor.offset {
		cursor.offset = cursor.selected
	}
	if cursor.selected >= cursor.offset+visibleRows {
		cursor.offset = cursor.selected - visibleRows + 1
	}
	cursor.offset = min(max(cursor.offset, 0), max(0, len(items)-visibleRows))
}

func (m navigatorModel) itemAtMouse(msg tea.MouseClickMsg, layout appLayout) (int, bool) {
	items := m.visibleItems()
	visibleIndex := msg.Y - layout.navigatorListY
	index := m.cursor().offset + visibleIndex
	valid := msg.Button == tea.MouseLeft &&
		layout.clickableNavigatorX(msg.X) &&
		visibleIndex >= 0 && visibleIndex < layout.navigatorListRows &&
		index >= 0 && index < len(items)
	return index, valid
}

func (m navigatorModel) view(status navigatorStatus, layout appLayout, focused bool) string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("● " + sanitizeText(status.databaseName)), ""}
	lines = append(lines, m.filter.View())
	lines = append(lines, m.sectionTabs(status, layout))

	items := m.visibleItems()
	switch {
	case m.section == navigatorTables && status.tablesLoading:
		lines = append(lines, "Loading tables…")
	case m.section == navigatorTables && status.tableLoadErr != nil:
		lines = append(lines, "Unable to load tables", truncateLabel(status.tableLoadErr.Error(), max(1, layout.navigator.width-4)))
	case len(items) == 0 && m.section == navigatorTables:
		if m.hasFilter() {
			lines = append(lines, "No matching tables.")
		} else {
			lines = append(lines, "No public tables.")
		}
	case len(items) == 0:
		lines = append(lines, "No matching views.")
	default:
		cursor := m.cursor()
		lastVisible := min(cursor.offset+layout.navigatorListRows, len(items))
		for index := cursor.offset; index < lastVisible; index++ {
			itemWidth := max(1, layout.navigator.width-4)
			marker := "  "
			style := lipgloss.NewStyle().Width(itemWidth)
			if index == cursor.selected {
				marker = "> "
				style = style.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
			}
			lines = append(lines, style.Render(marker+truncateLabel(items[index].name, max(0, itemWidth-len(marker)))))
		}
	}
	return panelStyle(layout.navigator.width, layout.navigator.height, focused).Render(strings.Join(lines, "\n"))
}

func (m navigatorModel) sectionTabs(status navigatorStatus, layout appLayout) string {
	tabs := []string{m.renderSectionTab(navigatorTables)}
	if m.hasViewsSection() {
		tabs = append(tabs, m.renderSectionTab(navigatorViews))
	}
	if status.viewsLoading {
		tabs = append(tabs, "Loading views…")
	} else if status.viewLoadErr != nil {
		tabs = append(tabs, "Views unavailable")
	}
	return truncateLabel(strings.Join(tabs, " "), max(1, layout.navigator.width-4))
}

func (m navigatorModel) renderSectionTab(section navigatorSection) string {
	name := "TABLES"
	if section == navigatorViews {
		name = "VIEWS"
	}
	if m.section == section {
		return "[" + name + "]"
	}
	return name
}

func (m navigatorModel) visibleTables() []db.Table {
	query := m.filterQuery()
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

func (m navigatorModel) visibleViews() []db.View {
	query := m.filterQuery()
	if query == "" {
		return m.views
	}

	views := make([]db.View, 0, len(m.views))
	for _, view := range m.views {
		if strings.Contains(strings.ToLower(view.Name), query) {
			views = append(views, view)
		}
	}
	return views
}

func (m navigatorModel) visibleItems() []navigatorItem {
	if m.section == navigatorViews {
		views := m.visibleViews()
		items := make([]navigatorItem, len(views))
		for index, view := range views {
			items[index] = navigatorItem{name: view.Name, section: navigatorViews}
		}
		return items
	}

	tables := m.visibleTables()
	items := make([]navigatorItem, len(tables))
	for index, table := range tables {
		items[index] = navigatorItem{name: table.Name, section: navigatorTables}
	}
	return items
}

func (m navigatorModel) hasViewsSection() bool {
	return true
}

func (m navigatorModel) cursor() navigatorCursor {
	return m.cursors[m.section]
}

func (m *navigatorModel) cursorPointer() *navigatorCursor {
	return &m.cursors[m.section]
}

func (m navigatorModel) hasFilter() bool {
	return m.filterQuery() != ""
}

func (m navigatorModel) filterQuery() string {
	return strings.ToLower(strings.TrimSpace(m.filter.Value()))
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
	previous, hadPrevious := m.selectedItem()
	m.filter.SetValue("")
	m.finishSearch()
	return m.normalizeSelectionWithPrevious(previous, hadPrevious, visibleRows)
}

func (m *navigatorModel) updateFilter(msg tea.Msg, visibleRows int) (bool, tea.Cmd) {
	previousQuery := m.filter.Value()
	previous, hadPrevious := m.selectedItem()
	var command tea.Cmd
	m.filter, command = m.filter.Update(msg)
	if m.filter.Value() == previousQuery {
		return false, command
	}
	return m.normalizeSelectionWithPrevious(previous, hadPrevious, visibleRows), command
}

func (m *navigatorModel) normalizeSelection(visibleRows int) bool {
	previous, hadPrevious := m.selectedItem()
	return m.normalizeSelectionWithPrevious(previous, hadPrevious, visibleRows)
}

func (m *navigatorModel) normalizeSelectionWithPrevious(previous navigatorItem, hadPrevious bool, visibleRows int) bool {
	if m.section == navigatorViews && !m.hasViewsSection() {
		m.section = navigatorTables
	}

	items := m.visibleItems()
	cursor := m.cursorPointer()
	if len(items) == 0 {
		cursor.selected = 0
		cursor.offset = 0
		return hadPrevious
	}

	if hadPrevious && previous.section == m.section {
		for index, item := range items {
			if item.name == previous.name {
				cursor.selected = index
				m.ensureVisible(visibleRows)
				return false
			}
		}
	}

	cursor.selected = 0
	cursor.offset = 0
	m.ensureVisible(visibleRows)
	return true
}
