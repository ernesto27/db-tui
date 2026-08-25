package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type objectsModal struct {
	sections []navigatorSection
	selected int
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
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Objects"),
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
