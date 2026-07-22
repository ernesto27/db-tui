package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type dumpModalState uint8

const (
	dumpConfirming dumpModalState = iota
	dumpRunning
	dumpSucceeded
	dumpFailed
)

type dumpModal struct {
	state        dumpModalState
	databaseName string
	err          error
}

func newDumpModal(databaseName string) dumpModal {
	return dumpModal{
		state:        dumpConfirming,
		databaseName: databaseName,
	}
}

func (m dumpModal) isRunning() bool {
	return m.state == dumpRunning
}

func (m dumpModal) view(width int, spinner string) string {
	modalWidth := min(56, max(40, width-8))

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Render("Database dump"),
		"",
	}

	switch m.state {
	case dumpConfirming:
		lines = append(lines,
			"Create an SQL dump of "+sanitizeText(m.databaseName)+"?",
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Enter confirm  •  Esc cancel"),
		)
	case dumpRunning:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Render(spinner+" Creating SQL dump…"),
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Please wait"),
		)
	case dumpSucceeded:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Render("✓ SQL dump created successfully"),
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Enter or Esc close"),
		)
	case dumpFailed:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("203")).
				Render("✕ Dump failed"),
			sanitizeText(m.err.Error()),
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Enter or Esc close"),
		)
	}

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}
