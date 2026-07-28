package app

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type exportModalState uint8

const (
	exportSelecting exportModalState = iota
	exportConfirming
	exportRunning
	exportSucceeded
	exportFailed
)

type exportSource uint8

const (
	exportTableSource exportSource = iota
	exportQuerySource
)

type exportModal struct {
	state     exportModalState
	source    exportSource
	tableName string
	query     string
	format    string
	err       error
}

func newExportModal(tableName string) exportModal {
	return exportModal{
		state:     exportSelecting,
		tableName: tableName,
		format:    db.ExportTypeCSV,
	}
}

func newQueryExportModal(query string) exportModal {
	return exportModal{
		state:  exportConfirming,
		source: exportQuerySource,
		query:  query,
		format: db.ExportTypeCSV,
	}
}

func (m exportModal) isRunning() bool {
	return m.state == exportRunning
}

func (m exportModal) view(width int, spinner string) string {
	modalWidth := min(56, max(40, width-8))

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Render("Export format"),
		"",
	}

	switch m.state {
	case exportSelecting:
		lines = append(lines,
			"Export "+m.description()+" as:",
			m.formatOption(db.ExportTypeCSV),
			m.formatOption(db.ExportTypeJSON),
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("↑/↓ or j/k choose  •  Enter continue  •  Esc cancel"),
		)
	case exportConfirming:
		lines = append(lines,
			"Export "+m.description()+" to "+m.formatName()+"?",
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Enter confirm  •  Esc cancel"),
		)
	case exportRunning:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Render(spinner+" Exporting "+m.description()+" to "+m.formatName()+"…"),
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Please wait"),
		)
	case exportSucceeded:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Render("✓ "+m.formatName()+" exported successfully"),
			"",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Enter or Esc close"),
		)
	case exportFailed:
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("203")).
				Render("✕ "+m.formatName()+" export failed"),
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

func (m exportModal) formatOption(format string) string {
	prefix := "  "
	if m.format == format {
		prefix = "› "
	}
	return prefix + strings.ToUpper(format)
}

func (m exportModal) formatName() string {
	return strings.ToUpper(m.format)
}

func (m exportModal) description() string {
	if m.source == exportQuerySource {
		return "query results"
	}
	return sanitizeText(m.tableName)
}
