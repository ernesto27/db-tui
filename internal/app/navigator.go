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
	navigatorMaterializedViews
	navigatorFunctions
	navigatorSectionCount
)

type navigatorCursor struct {
	selected int
	offset   int
}

type navigatorItem struct {
	name     string
	schema   string
	section  navigatorSection
	function db.FunctionColumns
}

func (i navigatorItem) rowSource() db.Table {
	return db.Table{Schema: i.schema, Name: i.name}
}

type navigatorModel struct {
	tables                     []db.Table
	views                      []db.View
	materializedViews          []db.MaterializedView
	materializedViewsAvailable bool
	functions                  []db.FunctionColumns
	functionsAvailable         bool
	schema                     string
	section                    navigatorSection
	cursors                    [navigatorSectionCount]navigatorCursor

	filter    textinput.Model
	searching bool
}

type navigatorStatus struct {
	databaseConneced         bool
	databaseName             string
	tablesLoading            bool
	tableLoadErr             error
	viewsLoading             bool
	viewLoadErr              error
	materializedViewsLoading bool
	materializedViewLoadErr  error
	functionsLoading         bool
	functionLoadErr          error
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

func (m *navigatorModel) setMaterializedViewsAvailable(available bool) {
	m.materializedViewsAvailable = available
	if !available && m.section == navigatorMaterializedViews {
		m.section = navigatorViews
	}
	m.normalizeSelection(1)
}

func (m *navigatorModel) setFunctionsAvailable(available bool) {
	m.functionsAvailable = available
	if !available && m.section == navigatorFunctions {
		m.section = navigatorTables
	}
	m.normalizeSelection(1)
}

func (m *navigatorModel) setFunctions(functions []db.FunctionColumns) {
	m.functions = functions
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
	return item.rowSource(), true
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

func (m navigatorModel) selectedIsFunction() bool {
	item, ok := m.selectedItem()
	return ok && item.section == navigatorFunctions
}

func (m navigatorModel) hasSelection() bool {
	_, ok := m.selectedItem()
	return ok
}

func (m navigatorModel) hasRelations() bool {
	return len(m.tables) > 0 || len(m.views) > 0 || len(m.materializedViews) > 0
}

func (m navigatorModel) hasObjects() bool {
	return m.hasRelations() || len(m.functions) > 0
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

func (m *navigatorModel) selectSection(section navigatorSection, visibleRows int) bool {
	if !m.sectionAvailable(section) || m.section == section {
		return false
	}
	m.section = section
	m.ensureVisible(visibleRows)
	return true
}

func (m navigatorModel) sectionAvailable(section navigatorSection) bool {
	switch section {
	case navigatorTables, navigatorViews:
		return true
	case navigatorMaterializedViews:
		return m.materializedViewsAvailable
	case navigatorFunctions:
		return m.functionsAvailable
	default:
		return false
	}
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
	lines := []string{}
	if status.databaseConneced {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("● "+sanitizeText(status.databaseName)), "")
		lines = append(lines, m.filter.View())
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colorTextMuted).Render(m.sectionTitle()))
	}

	items := m.visibleItems()
	switch {
	case m.sectionLoading(status):
		lines = append(lines, "Loading "+strings.ToLower(m.sectionTitle())+"…")
	case m.sectionError(status) != nil:
		lines = append(lines, "Unable to load "+strings.ToLower(m.sectionTitle()), truncateLabel(m.sectionError(status).Error(), max(1, layout.navigator.width-4)))
	case len(items) == 0:
		if m.hasFilter() {
			lines = append(lines, "No matching "+strings.ToLower(m.sectionTitle())+".")
		}
	default:
		cursor := m.cursor()
		lastVisible := min(cursor.offset+layout.navigatorListRows, len(items))
		for index := cursor.offset; index < lastVisible; index++ {
			itemWidth := max(1, layout.navigator.width-4)
			marker := "  "
			style := lipgloss.NewStyle().Width(itemWidth)
			if index == cursor.selected {
				marker = "> "
				style = style.Bold(true).Foreground(colorSelectionForeground).Background(colorSelectionBackground)
			}
			lines = append(lines, style.Render(marker+truncateLabel(items[index].name, max(0, itemWidth-len(marker)))))
		}
	}
	return panelStyle(layout.navigator.width, layout.navigator.height, focused).Render(strings.Join(lines, "\n"))
}

func (m navigatorModel) sectionTitle() string {
	var title string
	switch m.section {
	case navigatorViews:
		title = "Views"
	case navigatorMaterializedViews:
		title = "Materialized views"
	case navigatorFunctions:
		title = "Functions"
	default:
		title = "Tables"
	}
	if m.schema == "" {
		return title
	}
	return sanitizeText(m.schema) + " — " + title
}

func (m navigatorModel) sectionLoading(status navigatorStatus) bool {
	switch m.section {
	case navigatorViews:
		return status.viewsLoading
	case navigatorMaterializedViews:
		return status.materializedViewsLoading
	case navigatorFunctions:
		return status.functionsLoading
	default:
		return status.tablesLoading
	}
}

func (m navigatorModel) sectionError(status navigatorStatus) error {
	switch m.section {
	case navigatorViews:
		return status.viewLoadErr
	case navigatorMaterializedViews:
		return status.materializedViewLoadErr
	case navigatorFunctions:
		return status.functionLoadErr
	default:
		return status.tableLoadErr
	}
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
	switch m.section {
	case navigatorViews:
		views := m.visibleViews()
		items := make([]navigatorItem, len(views))
		for index, view := range views {
			items[index] = navigatorItem{
				name:    view.Name,
				schema:  m.schema,
				section: navigatorViews,
			}
		}
		return items

	case navigatorMaterializedViews:
		views := m.visibleMaterializedViews()
		items := make([]navigatorItem, len(views))
		for index, view := range views {
			items[index] = navigatorItem{
				name:    view.Name,
				schema:  m.schema,
				section: navigatorMaterializedViews,
			}
		}
		return items

	case navigatorFunctions:
		functions := m.visibleFunctions()
		items := make([]navigatorItem, len(functions))
		for index, function := range functions {
			items[index] = navigatorItem{
				name:     function.Name,
				schema:   m.schema,
				section:  navigatorFunctions,
				function: function,
			}
		}
		return items

	default:
		tables := m.visibleTables()
		items := make([]navigatorItem, len(tables))
		for index, table := range tables {
			items[index] = navigatorItem{
				name:    table.Name,
				schema:  table.Schema,
				section: navigatorTables,
			}
		}
		return items
	}
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
	if !m.sectionAvailable(m.section) {
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

func (m navigatorModel) visibleMaterializedViews() []db.MaterializedView {
	query := m.filterQuery()
	if query == "" {
		return m.materializedViews
	}

	views := make([]db.MaterializedView, 0, len(m.materializedViews))
	for _, view := range m.materializedViews {
		if strings.Contains(strings.ToLower(view.Name), query) {
			views = append(views, view)
		}
	}
	return views
}

func (m navigatorModel) visibleFunctions() []db.FunctionColumns {
	query := m.filterQuery()
	if query == "" {
		return m.functions
	}

	functions := make([]db.FunctionColumns, 0, len(m.functions))
	for _, function := range m.functions {
		if strings.Contains(strings.ToLower(function.Name), query) {
			functions = append(functions, function)
		}
	}
	return functions
}
