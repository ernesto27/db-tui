package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/version"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	view := m.baseView()
	if m.modal != nil || m.connectionsModal != nil || m.dumpModal != nil {
		view.Content = m.renderModalOverlay(view.Content)
	}
	return view
}

func (m Model) baseView() tea.View {
	appVersion := version.Version()
	headerTitle := fmt.Sprintf("db-tui v%s", appVersion)
	if m.databaseName != "" {
		headerTitle = fmt.Sprintf(
			"db-tui v%s  /  %s  /  %s",
			appVersion,
			sanitizeText(m.databaseName),
			engineDisplayName(m.databaseEngine),
		)
	}
	header := lipgloss.NewStyle().Width(m.layout.width).Padding(0, 1).Bold(true).
		Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).
		Render(headerTitle)
	rightPanel := m.data.view(m.dataStatus(), m.layout, m.focus == focusData)
	if m.panel == panelQuery {
		rightPanel = m.query.view(m.layout, m.focus == focusData, m.database != nil, m.spinner())
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.navigator.view(m.navigatorStatus(), m.layout, m.focus == focusNavigator), " ",
		rightPanel,
	)
	footer := lipgloss.NewStyle().Width(m.layout.width).Padding(0, 1).
		Foreground(lipgloss.Color("245")).Render(m.footerText())

	view := tea.NewView(strings.Join([]string{header, body, footer}, "\n"))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "db-tui"
	return view
}

func engineDisplayName(engine string) string {
	switch engine {
	case db.EngineMySQL:
		return "MySQL"
	case db.EnginePostgreSQL, "":
		return "PostgreSQL"
	default:
		return sanitizeText(engine)
	}
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
	switch {
	case m.modal != nil:
		modal = m.modal.view(m.layout.width)
	case m.connectionsModal != nil:
		modal = m.connectionsModal.view(m.layout.width)
	default:
		modal = m.dumpModal.view(m.layout.width, m.spinner())
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
		if m.panel == panelQuery {
			return "raw query  •  connection required  •  Ctrl+T table data  •  Ctrl+N new connection  •  Ctrl+L open connections  •  q quit"
		}
		return "Ctrl+N new connection  •  Ctrl+L open connections  •  Ctrl+R raw query  •  q quit"
	}
	if m.panel == panelQuery {
		return "raw query  •  Ctrl+P execute  •  Tab editor/results  •  ↑/↓, j/k, or wheel scroll results  •  Ctrl+T table data  •  Ctrl+L connections  •  q quit"
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
	visibleTables := m.navigator.visibleTables()
	if len(visibleTables) == 0 {
		return "no matching tables  •  Ctrl+F edit search  •  Esc clear search  •  q quit"
	}

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

	return fmt.Sprintf("focus: %s%s  •  Ctrl+F search tables  •  Ctrl+D dump database  •  Ctrl+R raw query  •  Tab table/search/data  •  q quit",
		focusLabel, rowStatus)
}

func panelStyle(width, height int, focused bool) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if focused {
		borderColor = lipgloss.Color("62")
	}
	return lipgloss.NewStyle().Width(width).Height(height).Padding(0, 1).
		Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
}
