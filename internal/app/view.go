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
	if m.modal != nil || m.connectionsModal != nil || m.dumpModal != nil || m.exportModal != nil || m.ddlModal != nil || m.columnsModal != nil || m.indexesModal != nil || m.actionsModal != nil {
		view.Content = m.renderModalOverlay(view.Content)
	}
	return view
}

func (m Model) baseView() tea.View {
	appVersion := version.Version()
	headerTitle := fmt.Sprintf("db-tui v%s", appVersion)
	if databaseName := m.connectedDatabaseName(); databaseName != "" {
		segments := []string{
			headerTitle,
			sanitizeText(databaseName),
			engineDisplayName(m.database.Engine()),
		}
		if host := m.database.Host(); host != "" {
			segments = append(segments, host)
		}
		headerTitle = strings.Join(segments, "  /  ")
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
	case db.EngineSQLite:
		return "SQLite"
	default:
		return sanitizeText(engine)
	}
}

func (m Model) navigatorStatus() navigatorStatus {
	return navigatorStatus{
		databaseName:  m.connectedDatabaseName(),
		tablesLoading: m.loading,
		tableLoadErr:  m.tableLoadErr,
		viewsLoading:  m.viewsLoading,
		viewLoadErr:   m.viewLoadErr,
	}
}

func (m Model) connectedDatabaseName() string {
	if m.database == nil {
		return ""
	}
	return m.database.Name()
}

func (m Model) dataStatus() dataStatus {
	tableLoadErr := error(nil)
	if !m.navigator.hasSelection() {
		tableLoadErr = m.tableLoadErr
	}
	return dataStatus{
		tableName:     m.navigator.selectedName(),
		disconnected:  m.database == nil,
		tablesLoading: (m.loading || m.viewsLoading) && !m.navigator.hasSelection(),
		tableLoadErr:  tableLoadErr,
		noTables:      !m.loading && !m.viewsLoading && !m.navigator.hasSelection(),
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
	case m.dumpModal != nil:
		modal = m.dumpModal.view(m.layout.width, m.spinner())
	case m.exportModal != nil:
		modal = m.exportModal.view(m.layout.width, m.spinner())
	case m.actionsModal != nil:
		modal = m.actionsModal.view(m.layout.width)
	case m.ddlModal != nil:
		modal = m.ddlModal.view(m.layout, m.spinner())
	case m.columnsModal != nil:
		modal = m.columnsModal.view(m.layout, m.spinner())
	case m.indexesModal != nil:
		modal = m.indexesModal.view(m.layout, m.spinner())
	default:
		return base
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
		exportHelp := ""
		if !m.query.loading && m.query.err == nil && len(m.query.result.Columns) > 0 && strings.TrimSpace(m.query.lastExecutedSQL) != "" {
			exportHelp = "  •  Ctrl+E export results"
		}
		ddlHelp := ""
		_, tableSelected := m.navigator.selectedTable()
		if !m.navigator.selectedIsView() && (tableSelected || (m.activeConnectionIndex >= 0 && m.activeConnectionIndex < len(m.config.Connections))) {
			ddlHelp = "  •  Ctrl+G actions"
		}
		return "raw query  •  Ctrl+P execute" + exportHelp + ddlHelp + "  •  Tab editor/results  •  ↑/↓, j/k, or wheel scroll results  •  Ctrl+T table data  •  Ctrl+L connections  •  q quit"
	}
	if (m.loading || m.viewsLoading) && !m.navigator.hasSelection() {
		return "loading database objects  •  q quit"
	}
	if m.tableLoadErr != nil && !m.navigator.hasSelection() {
		return "unable to load tables  •  q quit"
	}
	if !m.navigator.hasSelection() {
		return "no public tables or views  •  q quit"
	}
	if len(m.navigator.visibleItems()) == 0 {
		return "no matching database objects  •  Ctrl+F edit search  •  Esc clear search  •  q quit"
	}

	focusLabel := "navigator"
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

	tableSelected := false
	if _, ok := m.navigator.selectedTable(); ok {
		tableSelected = true
	}
	tableHelp := ""
	if tableSelected {
		tableHelp = "  •  Ctrl+E export  •  Ctrl+G actions"
	}
	sectionHelp := ""
	if m.navigator.hasViewsSection() {
		sectionHelp = "  •  ←/→ switch section"
	}
	return fmt.Sprintf("focus: %s%s  •  Ctrl+F search relations%s%s  •  Ctrl+D dump database  •  Ctrl+R raw query  •  Tab navigator/data  •  q quit",
		focusLabel, rowStatus, sectionHelp, tableHelp)
}

func panelStyle(width, height int, focused bool) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if focused {
		borderColor = lipgloss.Color("62")
	}
	return lipgloss.NewStyle().Width(width).Height(height).Padding(0, 1).
		Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
}
