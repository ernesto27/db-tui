package app

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

const helpModalTwoColumnWidth = 50

type helpShortcut struct {
	key         string
	description string
}

type helpModal struct {
	general  []helpShortcut
	specific []helpShortcut
}

func newHelpModal(keys keyMap) helpModal {
	return helpModal{
		general: shortcutsFor(
			keys.help,
			keys.newConnection,
			keys.connections,
			keys.query,
			keys.tableData,
			keys.quit,
		),
		specific: append(shortcutsFor(
			keys.tableSearch,
			keys.executeQuery,
			keys.queryFocus,
			keys.focusLeft,
			keys.focusRight,
			keys.up,
			keys.down,
			keys.pageUp,
			keys.pageDown,
			keys.home,
			keys.end,
			keys.dump,
			keys.export,
		),
			helpShortcut{key: "enter", description: "apply / connect / confirm"},
			helpShortcut{key: "shift+tab", description: "previous connection field"},
			helpShortcut{key: "d", description: "remove saved connection"},
			helpShortcut{key: "y/n", description: "confirm / cancel removal"},
			helpShortcut{key: "esc", description: "clear / close / cancel"},
		),
	}
}

func shortcutsFor(bindings ...key.Binding) []helpShortcut {
	shortcuts := make([]helpShortcut, 0, len(bindings))
	for _, binding := range bindings {
		help := binding.Help()
		shortcuts = append(shortcuts, helpShortcut{
			key:         help.Key,
			description: help.Desc,
		})
	}
	return shortcuts
}

func (m helpModal) view(width int) string {
	modalWidth := min(88, max(52, width-8))
	contentWidth := modalWidth - 6
	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Render("Keyboard shortcuts"),
		"",
	}

	lines = append(lines, renderShortcutSection("General", m.general, contentWidth)...)
	lines = append(lines, "")
	lines = append(lines, renderShortcutSection("Specific", m.specific, contentWidth)...)
	lines = append(lines,
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("Esc close"),
	)

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}

func renderShortcutSection(title string, shortcuts []helpShortcut, width int) []string {
	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			Render(title),
	}
	if width < helpModalTwoColumnWidth {
		for _, shortcut := range shortcuts {
			lines = append(lines, renderShortcut(shortcut, width))
		}
		return lines
	}

	columnWidth := (width - 2) / 2
	rowCount := (len(shortcuts) + 1) / 2
	leftColumn := lipgloss.NewStyle().Width(columnWidth)
	for row := range rowCount {
		left := renderShortcut(shortcuts[row], columnWidth)
		rightIndex := row + rowCount
		if rightIndex >= len(shortcuts) {
			lines = append(lines, left)
			continue
		}
		lines = append(lines,
			leftColumn.Render(left)+"  "+renderShortcut(shortcuts[rightIndex], columnWidth),
		)
	}
	return lines
}

func renderShortcut(shortcut helpShortcut, width int) string {
	keyWidth := min(12, max(10, width/3))
	keyStyle := lipgloss.NewStyle().
		Width(keyWidth).
		Bold(true).
		Foreground(lipgloss.Color("230"))
	descriptionWidth := max(1, width-keyWidth-1)

	return keyStyle.Render(shortcut.key) + " " +
		truncateLabel(shortcut.description, descriptionWidth)
}
