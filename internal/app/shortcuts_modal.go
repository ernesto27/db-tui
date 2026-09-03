package app

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type shortcut struct {
	key    string
	action string
}

type shortcutSection struct {
	title     string
	shortcuts []shortcut
}

var shortcutSections = []shortcutSection{
	{title: "Global", shortcuts: []shortcut{
		{key: "Ctrl+K", action: "Open or close keyboard shortcuts"},
		{key: "Ctrl+L", action: "Open connections"},
		{key: "Ctrl+N", action: "New connection / new query script"},
		{key: "Ctrl+S", action: "Open settings"},
		{key: "Ctrl+R", action: "Open raw query"},
		{key: "Ctrl+T", action: "Open table data"},
		{key: "Ctrl+O", action: "Select database objects"},
		{key: "Q / Ctrl+C", action: "Quit"},
	}},
	{title: "Navigation", shortcuts: []shortcut{
		{key: "Tab", action: "Switch focused pane"},
		{key: "Up / K", action: "Move up"},
		{key: "Down / J", action: "Move down"},
		{key: "Left / Right", action: "Switch pane or scroll columns"},
		{key: "PageUp / PageDown", action: "Move by page"},
		{key: "Home / End", action: "Jump to first or last object"},
		{key: "Enter", action: "Open the selected object"},
		{key: "Ctrl+F", action: "Search database objects"},
		{key: "Esc", action: "Clear search or close a modal"},
	}},
	{title: "Tables and data", shortcuts: []shortcut{
		{key: "R", action: "Refresh table data"},
		{key: "E", action: "Edit selected row"},
		{key: "D", action: "Delete selected row"},
		{key: "Ctrl+E", action: "Export table data"},
		{key: "Ctrl+G", action: "Open table or connection actions"},
		{key: "Ctrl+D", action: "Dump database"},
	}},
	{title: "Query editor", shortcuts: []shortcut{
		{key: "Ctrl+N", action: "Start a new query script"},
		{key: "Ctrl+P", action: "Execute query"},
		{key: "Ctrl+H", action: "Open saved scripts"},
		{key: "Ctrl+E", action: "Export query results"},
		{key: "Tab", action: "Switch editor and results"},
		{key: "Up / Down", action: "Scroll query results"},
		{key: "PageUp / PageDown", action: "Scroll results by page"},
	}},
	{title: "Dialogs", shortcuts: []shortcut{
		{key: "Up / Down or J / K", action: "Move between options"},
		{key: "Tab / Shift+Tab", action: "Move between fields"},
		{key: "Enter", action: "Select or confirm"},
		{key: "Esc", action: "Cancel or close"},
	}},
}

type shortcutsModal struct {
	offset int
}

func newShortcutsModal(layout appLayout) shortcutsModal {
	modal := shortcutsModal{}
	modal.clamp(layout)
	return modal
}

func (m *shortcutsModal) clamp(layout appLayout) {
	m.offset = min(max(m.offset, 0), max(0, len(m.lines(layout))-m.visibleRows(layout)))
}

func (m shortcutsModal) visibleRows(layout appLayout) int {
	return max(1, layout.height-8)
}

func (m *shortcutsModal) scroll(delta int, layout appLayout) {
	m.offset += delta
	m.clamp(layout)
}

func (m shortcutsModal) lines(layout appLayout) []string {
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorModalBackground)
	actionStyle := lipgloss.NewStyle().Foreground(colorText).Background(colorModalBackground)
	headingStyle := lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Background(colorModalBackground)
	mutedStyle := lipgloss.NewStyle().Foreground(colorTextMuted).Background(colorModalBackground)

	lines := make([]string, 0, 48)
	for sectionIndex, section := range shortcutSections {
		if sectionIndex > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, headingStyle.Render(section.title))
		lines = append(lines, mutedStyle.Render("Key"+strings.Repeat(" ", 24)+"Action"))
		for _, item := range section.shortcuts {
			actionWidth := max(1, shortcutsModalWidth(layout)-33)
			lines = append(lines, keyStyle.Render(padRight(item.key, 27))+actionStyle.Render(truncateLabel(item.action, actionWidth)))
		}
	}
	return lines
}

func (m shortcutsModal) view(layout appLayout) string {
	modalWidth := shortcutsModalWidth(layout)
	allLines := m.lines(layout)
	first := min(m.offset, max(0, len(allLines)-m.visibleRows(layout)))
	last := min(first+m.visibleRows(layout), len(allLines))
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Background(colorModalBackground).Render("Keyboard shortcuts"), ""}
	lines = append(lines, allLines[first:last]...)
	help := "↑/↓ or j/k scroll  •  PageUp/PageDown page  •  Esc or Ctrl+K close"
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Background(colorModalBackground).Render(help))
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func shortcutsModalWidth(layout appLayout) int {
	return min(88, max(56, layout.width-8))
}

func padRight(value string, width int) string {
	return value + strings.Repeat(" ", max(1, width-lipgloss.Width(value)))
}

func (m *Model) updateShortcutsModal(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.String() == "esc", key.Matches(msg, m.keys.shortcuts):
			m.shortcutsModal = nil
		case msg.String() == "up" || msg.String() == "k":
			m.shortcutsModal.scroll(-1, m.layout)
		case msg.String() == "down" || msg.String() == "j":
			m.shortcutsModal.scroll(1, m.layout)
		case msg.String() == "pgup":
			m.shortcutsModal.scroll(-m.shortcutsModal.visibleRows(m.layout), m.layout)
		case msg.String() == "pgdown":
			m.shortcutsModal.scroll(m.shortcutsModal.visibleRows(m.layout), m.layout)
		case msg.String() == "home":
			m.shortcutsModal.offset = 0
		case msg.String() == "end":
			m.shortcutsModal.offset = len(m.shortcutsModal.lines(m.layout))
			m.shortcutsModal.clamp(m.layout)
		}
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			m.shortcutsModal.scroll(-3, m.layout)
		} else if msg.Button == tea.MouseWheelDown {
			m.shortcutsModal.scroll(3, m.layout)
		}
	}
	return nil
}
