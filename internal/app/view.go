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
	if m.modal != nil || m.connectionsModal != nil || m.settingsModal != nil || m.dumpModal != nil || m.exportModal != nil || m.ddlModal != nil || m.columnsModal != nil || m.indexesModal != nil || m.actionsModal != nil || m.editRowModal != nil || m.deleteRowModal != nil || m.sqlScriptsModal != nil || m.objectsModal != nil || m.databaseExplorerModal != nil {
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
		Foreground(colorTitle).Background(colorHeaderBackground).
		Render(headerTitle)
	rightPanel := m.data.view(m.dataStatus(), m.layout, m.focus == focusData)
	if m.activeFunction.set {
		rightPanel = m.activeFunction.view(m.layout, m.focus == focusData)
	}
	if m.panel == panelQuery {
		rightPanel = m.query.view(m.layout, m.focus == focusData, m.database != nil, m.spinner())
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.navigator.view(m.navigatorStatus(), m.layout, m.focus == focusNavigator), " ",
		rightPanel,
	)
	footer := lipgloss.NewStyle().Width(m.layout.width).Padding(0, 1).
		Foreground(colorTextMuted).Render(m.footerText())

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
	case db.EngineOracle:
		return "Oracle"
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
		databaseName:             m.connectedDatabaseName(),
		tablesLoading:            m.loading,
		tableLoadErr:             m.tableLoadErr,
		viewsLoading:             m.viewsLoading,
		viewLoadErr:              m.viewLoadErr,
		materializedViewsLoading: m.materializedViewsLoading,
		materializedViewLoadErr:  m.materializedViewLoadErr,
		functionsLoading:         m.functionsLoading,
		functionLoadErr:          m.functionLoadErr,
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
	if !m.navigator.hasRelations() {
		tableLoadErr = m.tableLoadErr
	}
	tableName := ""
	if m.activeRelation.set {
		tableName = sanitizeText(m.activeRelation.item.name)
	}
	return dataStatus{
		tableName:       tableName,
		highlightedName: m.navigator.selectedName(),
		active:          m.activeRelation.set,
		disconnected:    m.database == nil,
		tablesLoading:   (m.loading || m.viewsLoading || m.materializedViewsLoading || m.functionsLoading) && !m.navigator.hasObjects(),
		tableLoadErr:    tableLoadErr,
		noTables:        !m.loading && !m.viewsLoading && !m.materializedViewsLoading && !m.functionsLoading && !m.navigator.hasObjects(),
		spinner:         m.spinner(),
	}
}

func (m Model) dataGridTop() int {
	status := m.dataStatus()
	return m.data.gridTop(m.data.title(status, m.layout), m.layout)
}

func (m Model) renderModalOverlay(base string) string {
	var modal string
	switch {
	case m.modal != nil:
		modal = m.modal.view(m.layout.width)
	case m.connectionsModal != nil:
		modal = m.connectionsModal.view(m.layout.width)
	case m.settingsModal != nil:
		modal = m.settingsModal.view(m.layout.width)
	case m.dumpModal != nil:
		modal = m.dumpModal.view(m.layout.width, m.spinner())
	case m.exportModal != nil:
		modal = m.exportModal.view(m.layout.width, m.spinner())
	case m.editRowModal != nil:
		modal = m.editRowModal.view(m.layout)
	case m.deleteRowModal != nil:
		modal = m.deleteRowModal.view(m.layout)
	case m.actionsModal != nil:
		modal = m.actionsModal.view(m.layout.width)
	case m.ddlModal != nil:
		modal = m.ddlModal.view(m.layout, m.spinner())
	case m.columnsModal != nil:
		modal = m.columnsModal.view(m.layout, m.spinner())
	case m.indexesModal != nil:
		modal = m.indexesModal.view(m.layout, m.spinner())
	case m.sqlScriptsModal != nil:
		modal = m.sqlScriptsModal.view(m.layout)
	case m.databaseExplorerModal != nil:
		modal = m.databaseExplorerModal.view(m.layout)
	case m.objectsModal != nil:
		modal = m.objectsModal.view(m.layout.width)
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
			return "raw query  •  connection required  •  Ctrl+S settings  •  Ctrl+T table data  •  Ctrl+N new connection  •  Ctrl+L open connections  •  q quit"
		}
		return "Ctrl+S settings  •  Ctrl+N new connection  •  Ctrl+L open connections  •  Ctrl+R raw query  •  q quit"
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
		return "raw query  •  Ctrl+N new script  •  Ctrl+P execute  •  Ctrl+H saved scripts" + exportHelp + ddlHelp + "  •  Tab editor/results  •  ↑/↓, j/k, or wheel scroll results  •  Ctrl+S settings  •  Ctrl+T table data  •  Ctrl+L connections  •  q quit"
	}
	if (m.loading || m.viewsLoading || m.materializedViewsLoading || m.functionsLoading) && !m.navigator.hasObjects() {
		return "loading database objects  •  q quit"
	}
	if m.tableLoadErr != nil && !m.navigator.hasObjects() {
		return "unable to load tables  •  q quit"
	}
	if !m.navigator.hasObjects() {
		return "no database objects found  •  Ctrl+O select objects  •  q quit"
	}
	if len(m.navigator.visibleItems()) == 0 {
		return "no matching database objects  •  Ctrl+F edit search  •  Esc clear search  •  q quit"
	}

	rowStatus := ""
	if m.data.loading {
		rowStatus = "  •  " + m.spinner() + " Query executing…"
	} else if m.data.err != nil {
		rowStatus = "  •  row load failed"
	}

	tableSelected := false
	if _, ok := m.navigator.selectedTable(); ok {
		tableSelected = true
	}
	tableHelp := ""
	if tableSelected {
		tableHelp = "  •  Ctrl+E export  •  Ctrl+G actions"
	}
	editHelp := ""
	if m.activeRelation.set && m.activeRelation.item.section == navigatorTables && len(m.data.page.Rows) > 0 && !m.data.loading && m.editRowModal == nil {
		editHelp = "  •  e edit row"
	}
	activationHelp := ""
	if m.focus == focusNavigator {
		activationHelp = "  •  Enter "
		if m.navigator.selectedIsFunction() {
			activationHelp += "view function"
		} else {
			activationHelp += "load rows"
		}
	}
	refreshHelp := ""
	if m.panel == panelData && m.focus == focusData && !m.activeFunction.set {
		refreshHelp = "  •  r refresh"
	}
	functionHelp := ""
	if m.activeFunction.set && m.panel == panelData && m.focus == focusData {
		functionHelp = "  •  ↑/↓ or j/k scroll function"
	}
	return fmt.Sprintf("Ctrl+O objects  •  Ctrl+F search%s%s%s%s%s%s  •  Ctrl+S settings  •  Ctrl+D dump database  •  Ctrl+R raw query  •  Tab navigator/data  •  q quit",
		rowStatus, activationHelp, refreshHelp, functionHelp, tableHelp, editHelp)
}

func panelStyle(width, height int, focused bool) lipgloss.Style {
	borderColor := colorBorderInactive
	if focused {
		borderColor = colorBorderActive
	}
	return lipgloss.NewStyle().Width(width).Height(height).Padding(0, 1).
		Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
}
