package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/version"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	view := m.baseView()
	if m.modal != nil || m.connectionsModal != nil {
		view.Content = m.renderModalOverlay(view.Content)
	}
	return view
}

func (m Model) baseView() tea.View {
	appVersion := version.Version()
	headerTitle := fmt.Sprintf("db-tui v%s  /  PostgreSQL", appVersion)
	if m.databaseName != "" {
		headerTitle = fmt.Sprintf("db-tui v%s  /  %s  /  PostgreSQL", appVersion, sanitizeText(m.databaseName))
	}
	header := lipgloss.NewStyle().Width(m.layout.width).Padding(0, 1).Bold(true).
		Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).
		Render(headerTitle)
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.navigator.view(m.navigatorStatus(), m.layout, m.focus == focusNavigator), " ",
		m.data.view(m.dataStatus(), m.layout, m.focus == focusData),
	)
	footer := lipgloss.NewStyle().Width(m.layout.width).Padding(0, 1).
		Foreground(lipgloss.Color("245")).Render(m.footerText())

	view := tea.NewView(strings.Join([]string{header, body, footer}, "\n"))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "db-tui"
	return view
}

func (m Model) navigatorStatus() navigatorStatus {
	return navigatorStatus{
		databaseName: m.databaseName,
		loading:      m.loading,
		tableLoadErr: m.tableLoadErr,
	}
}

func (m Model) dataStatus() dataStatus {
	return dataStatus{
		tableName:     m.navigator.selectedTableName(),
		disconnected:  m.database == nil,
		tablesLoading: m.loading,
		tableLoadErr:  m.tableLoadErr,
		noTables:      len(m.navigator.tables) == 0,
		spinner:       m.spinner(),
	}
}

func (m Model) renderModalOverlay(base string) string {
	var modal string
	if m.modal != nil {
		modal = m.modal.view(m.layout.width)
	} else {
		modal = m.connectionsModal.view(m.layout.width)
	}

	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(modal).
			X(max(0, (m.layout.width-lipgloss.Width(modal))/2)).
			Y(max(0, (m.layout.height-lipgloss.Height(modal))/2)).
			Z(1),
	).Render()
}

func (m Model) footerText() string {
	if m.database == nil {
		return "welcome to db-tui  •  Ctrl+N new connection  •  Ctrl+L open connections  •  q quit"
	}
	if m.loading {
		return "loading tables  •  q quit"
	}
	if m.tableLoadErr != nil {
		return "unable to load tables  •  q quit"
	}
	if len(m.navigator.tables) == 0 {
		return "no public tables  •  q quit"
	}

	firstTable := m.navigator.offset + 1
	lastTable := min(m.navigator.offset+m.layout.navigatorListRows, len(m.navigator.tables))
	focusLabel := "tables"
	if m.focus == focusData {
		focusLabel = "data"
	}
	rowStatus := ""
	if m.data.loading {
		rowStatus = "  •  " + m.spinner() + " Query executing…"
	} else if m.data.err != nil {
		rowStatus = "  •  row load failed"
	} else if len(m.data.page.Rows) > 0 {
		firstRow := m.data.offset + 1
		lastRow := m.data.offset + len(m.data.page.Rows)
		rowStatus = fmt.Sprintf("  •  rows %d–%d", firstRow, lastRow)
		if m.data.page.HasMore {
			rowStatus += "  •  PgDown next"
		}
	}
	return fmt.Sprintf("%s  •  tables %d–%d/%d  •  focus: %s%s  •  ←/→ switch  •  q quit", m.navigator.selectedTableName(), firstTable, lastTable, len(m.navigator.tables), focusLabel, rowStatus)
}

func panelStyle(width, height int, focused bool) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if focused {
		borderColor = lipgloss.Color("62")
	}
	return lipgloss.NewStyle().Width(width).Height(height).Padding(0, 1).
		Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
}
