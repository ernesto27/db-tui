package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type deleteRowState uint8

const (
	deleteRowConfirming deleteRowState = iota
	deleteRowDeleting
	deleteRowSuccess
	deleteRowFailed
)

type deleteRowModal struct {
	table        db.Table
	whereColumns map[string]any
	state        deleteRowState
	err          error
}

func newDeleteRowModal(table db.Table, whereColumns map[string]any) deleteRowModal {
	return deleteRowModal{
		table:        table,
		whereColumns: whereColumns,
		state:        deleteRowConfirming,
	}
}

func (m deleteRowModal) update(msg tea.Msg) (deleteRowModal, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch m.state {
	case deleteRowConfirming:
		switch key.String() {
		case "enter":
			return m, func() tea.Msg {
				return deleteRowConfirmMsg{
					table:        m.table,
					whereColumns: m.whereColumns,
				}
			}
		case "esc":
			return m, func() tea.Msg { return deleteRowCancelMsg{} }
		}

	case deleteRowSuccess, deleteRowFailed:
		switch key.String() {
		case "enter", "esc":
			return m, func() tea.Msg { return deleteRowCancelMsg{} }
		}
	}

	return m, nil
}

func (m deleteRowModal) view(layout appLayout) string {
	styles := newEditRowStyles()
	contentWidth := editRowContentWidth(layout.width)

	title := styles.title.Render("Delete row") +
		styles.dim.Render("  ·  ") +
		styles.accent.Render(sanitizeText(m.table.Name))

	lines := []string{title, ""}

	switch m.state {
	case deleteRowConfirming:
		lines = append(lines,
			styles.base.Render("Delete this row?"),
			"",
			styles.dim.Render("Enter confirm  •  Esc cancel"),
		)

	case deleteRowDeleting:
		lines = append(lines,
			styles.base.Foreground(colorAccent).Render("Deleting…"),
		)

	case deleteRowSuccess:
		lines = append(lines,
			styles.base.Foreground(colorAccent).Render("✓ Row deleted"),
			"",
			styles.dim.Render("Enter or Esc continue"),
		)

	case deleteRowFailed:
		message := lipgloss.Wrap(sanitizeText(m.err.Error()), contentWidth, "")
		lines = append(lines,
			styles.base.Foreground(colorError).Render("✕ "+message),
			"",
			styles.dim.Render("Enter or Esc continue"),
		)
	}

	return styles.container(contentWidth).Render(strings.Join(lines, "\n"))
}
