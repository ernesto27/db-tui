package app

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type objectsModal struct {
	sections []navigatorSection
	selected int
}

type databaseExplorerModal struct {
	groups   []db.SchemaObjectGroup
	selected int
	offset   int
}

func newObjectsModal(navigator navigatorModel) objectsModal {
	sections := []navigatorSection{navigatorTables, navigatorViews}
	if navigator.materializedViewsAvailable {
		sections = append(sections, navigatorMaterializedViews)
	}
	if navigator.functionsAvailable {
		sections = append(sections, navigatorFunctions)
	}

	selected := 0
	for index, section := range sections {
		if section == navigator.section {
			selected = index
			break
		}
	}
	return objectsModal{sections: sections, selected: selected}
}

func newDatabaseExplorerModal(groups []db.SchemaObjectGroup) databaseExplorerModal {
	return databaseExplorerModal{groups: groups}
}

func (m *databaseExplorerModal) move(delta int, layout appLayout) {
	if len(m.groups) == 0 {
		m.selected = 0
		m.offset = 0
		return
	}
	m.selected = min(max(m.selected+delta, 0), len(m.groups)-1)
	m.ensureVisible(layout)
}

func (m *databaseExplorerModal) clamp(layout appLayout) {
	if len(m.groups) == 0 {
		m.selected = 0
		m.offset = 0
		return
	}
	m.selected = min(max(m.selected, 0), len(m.groups)-1)
	m.offset = min(max(m.offset, 0), max(0, len(m.groups)-m.visibleRows(layout)))
	m.ensureVisible(layout)
}

func (m *databaseExplorerModal) ensureVisible(layout appLayout) {
	visibleRows := m.visibleRows(layout)
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visibleRows {
		m.offset = m.selected - visibleRows + 1
	}
	m.offset = min(max(m.offset, 0), max(0, len(m.groups)-visibleRows))
}

func (m databaseExplorerModal) visibleRows(layout appLayout) int {
	return max(1, layout.height-8)
}

func (m databaseExplorerModal) selectedGroup() db.SchemaObjectGroup {
	if len(m.groups) == 0 {
		return db.SchemaObjectGroup{}
	}
	return m.groups[m.selected]
}

func (m *objectsModal) move(delta int) {
	m.selected = min(max(m.selected+delta, 0), len(m.sections)-1)
}

func (m objectsModal) selectedSection() navigatorSection {
	if len(m.sections) == 0 {
		return navigatorTables
	}
	return m.sections[m.selected]
}

func (m objectsModal) view(width int) string {
	modalWidth := min(40, max(28, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Database"),
		"",
	}
	for index, section := range m.sections {
		prefix := "  "
		style := lipgloss.NewStyle()
		if index == m.selected {
			prefix = "> "
			style = style.Bold(true).Foreground(colorTitle)
		}
		lines = append(lines, prefix+style.Render(section.displayName()))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("↑/↓ or j/k move  •  Enter select  •  Esc close"))
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m databaseExplorerModal) view(layout appLayout) string {
	modalWidth := min(48, max(32, layout.width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Database explorer"),
		"",
	}
	first := min(max(m.offset, 0), max(0, len(m.groups)-m.visibleRows(layout)))
	last := min(first+m.visibleRows(layout), len(m.groups))
	for index := first; index < last; index++ {
		group := m.groups[index]
		prefix := "  "
		style := lipgloss.NewStyle()
		if index == m.selected {
			prefix = "> "
			style = style.Bold(true).Foreground(colorTitle)
		}
		label := truncateLabel(sanitizeText(group.Schema)+" — "+schemaObjectTypeDisplayName(group.Type), max(1, modalWidth-6))
		lines = append(lines, prefix+style.Render(label))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("↑/↓ or j/k move  •  Enter select  •  Esc close"))
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func schemaObjectTypeDisplayName(objectType db.SchemaObjectType) string {
	switch objectType {
	case db.SchemaObjectViews:
		return "Views"
	case db.SchemaObjectMaterializedViews:
		return "Materialized views"
	case db.SchemaObjectFunctions:
		return "Functions"
	default:
		return "Tables"
	}
}

func (s navigatorSection) displayName() string {
	switch s {
	case navigatorViews:
		return "Views"
	case navigatorMaterializedViews:
		return "Materialized views"
	case navigatorFunctions:
		return "Functions"
	default:
		return "Tables"
	}
}
