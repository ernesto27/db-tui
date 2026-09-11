package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type rawQueryDeleteModal struct {
	sql string
}

type rawQueryDeleteConfirmMsg struct{}
type rawQueryDeleteCancelMsg struct{}

func newRawQueryDeleteModal(sql string) rawQueryDeleteModal {
	return rawQueryDeleteModal{sql: sql}
}

func (m rawQueryDeleteModal) update(msg tea.Msg) (rawQueryDeleteModal, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "enter":
		return m, func() tea.Msg { return rawQueryDeleteConfirmMsg{} }
	case "esc":
		return m, func() tea.Msg { return rawQueryDeleteCancelMsg{} }
	default:
		return m, nil
	}
}

func (m rawQueryDeleteModal) view(layout appLayout) string {
	styles := newEditRowStyles()
	contentWidth := editRowContentWidth(layout.width)

	lines := []string{
		styles.title.Render("Confirm DELETE"),
		"",
		styles.base.Render("This query will delete data."),
		"",
		styles.dim.Render("Enter confirm  •  Esc cancel"),
	}

	return styles.container(contentWidth).Render(strings.Join(lines, "\n"))
}
