package app

import (
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if command, handled := m.updateLifecycle(msg); handled {
		return m, command
	}

	if m.modal != nil {
		switch msg.(type) {
		case tea.MouseClickMsg, tea.MouseWheelMsg:
			return m, nil
		default:
			return m.updateModal(msg)
		}
	}
	if m.connectionsModal != nil {
		switch msg.(type) {
		case tea.MouseClickMsg, tea.MouseWheelMsg:
			return m, nil
		default:
			return m.updateConnectionsModal(msg)
		}
	}

	if m.dumpModal != nil {
		return m, m.updateDumpModal(msg)
	}
	if m.exportModal != nil {
		return m, m.updateExportModal(msg)
	}
	if m.ddlModal != nil {
		return m, m.updateDDLModal(msg)
	}
	if m.columnsModal != nil {
		return m, m.updateColumnsModal(msg)
	}
	if m.indexesModal != nil {
		return m, m.updateIndexesModal(msg)
	}

	if m.editRowModal != nil {
		return m.updateEditRowModal(msg)
	}
	if m.actionsModal != nil {
		return m.updateActionsModal(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m, m.updateKey(msg)
	case tea.PasteMsg:
		if m.navigator.searching {
			return m, m.updateNavigatorSearch(msg)
		}
		if m.panel == panelQuery && !m.query.resultsFocused {
			editor, command := m.query.editor.Update(msg)
			m.query.editor = editor
			return m, command
		}
		return m, nil
	case tea.MouseClickMsg:
		return m, m.updateMouseClick(msg)
	case tea.MouseWheelMsg:
		return m, m.updateMouseWheel(msg)
	default:
		return m, nil
	}
}

func (m *Model) updateLifecycle(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case spinnerTickMsg:
		if m.loading || m.viewsLoading || m.materializedViewsLoading || m.data.loading || m.query.loading ||
			(m.ddlModal != nil && m.ddlModal.loading) ||
			(m.columnsModal != nil && m.columnsModal.loading) ||
			(m.dumpModal != nil && m.dumpModal.isRunning()) ||
			(m.exportModal != nil && m.exportModal.isRunning()) ||
			(m.indexesModal != nil && m.indexesModal.loading) ||
			(m.editRowModal != nil && m.editRowModal.state == editRowSaving) {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return spinnerTick(), true
		}
		m.spinnerRunning = false
		return nil, true
	case tablesLoadedMsg:
		if msg.session != m.session {
			return nil, true
		}
		m.loading = false
		m.tableLoadErr = msg.err
		m.navigator.setTables(msg.tables)
		m.selectInitialRelation()
		return nil, true
	case viewsLoadedMsg:
		if msg.session != m.session {
			return nil, true
		}
		m.viewsLoading = false
		m.viewLoadErr = msg.err
		if msg.err == nil {
			m.navigator.setViews(msg.views)
		}
		m.selectInitialRelation()
		return nil, true
	case materializedViewsLoadedMsg:
		if msg.session != m.session {
			return nil, true
		}
		m.materializedViewsLoading = false
		m.materializedViewLoadErr = msg.err
		if msg.err == nil {
			m.navigator.materializedViews = msg.materializedViews
			m.navigator.normalizeSelection(1)
		}
		m.selectInitialRelation()
		return nil, true
	case rowsLoadedMsg:
		if msg.session != m.session ||
			!m.activeRelation.set ||
			msg.relation != m.activeRelation.item ||
			msg.request != m.activeRelation.request ||
			msg.offset != m.data.offset {
			return nil, true
		}
		m.data.finishLoad(msg.page, msg.selectedRow, msg.err, m.layout)
		return nil, true
	case tableDDLLoadedMsg:
		table, ok := m.navigator.selectedTable()
		if msg.session != m.session || m.ddlModal == nil || msg.request != m.ddlRequest || !ok || table.Name != msg.tableName || m.ddlModal.tableName != msg.tableName {
			return nil, true
		}
		m.ddlModal.finish(msg.sql, msg.err, m.layout)
		return nil, true
	case columnsLoadedMsg:
		table, ok := m.navigator.selectedTable()
		if msg.session != m.session || m.columnsModal == nil || msg.request != m.columnsRequest || !ok || table.Name != msg.tableName || m.columnsModal.tableName != msg.tableName {
			return nil, true
		}
		m.columnsModal.finish(msg.columns, msg.err, m.layout)
		return nil, true
	case indexesLoadedMsg:
		table, ok := m.navigator.selectedTable()
		if msg.session != m.session ||
			m.indexesModal == nil ||
			msg.request != m.indexesRequest ||
			!ok ||
			table.Name != msg.tableName ||
			m.indexesModal.tableName != msg.tableName {
			return nil, true
		}

		m.indexesModal.finish(msg.indexes, msg.err, m.layout)
		return nil, true

	case queryFinishedMsg:
		if msg.session != m.session || msg.request != m.query.request {
			return nil, true
		}
		m.query.finishExecute(msg.result, msg.elapsed, msg.err)
		return nil, true
	case tea.WindowSizeMsg:
		m.layout = newAppLayout(msg.Width, msg.Height)
		m.navigator.resize(m.layout)
		m.navigator.ensureVisible(m.layout.navigatorListRows)
		m.data.columnOffset = min(m.data.columnOffset, m.data.maxColumnOffset())
		m.data.ensureSelectedVisible(m.layout)
		m.query.resize(m.layout)
		if m.ddlModal != nil {
			m.ddlModal.clamp(m.layout)
		}
		if m.columnsModal != nil {
			m.columnsModal.clamp(m.layout)
		}
		if m.indexesModal != nil {
			m.indexesModal.clamp(m.layout)
		}
		if m.editRowModal != nil {
			m.editRowModal.clamp(m.layout)
		}
		return nil, true
	case renameRequestMsg:
		if msg.request != m.renameRequest || msg.index != m.activeConnectionIndex {
			return nil, true
		}
		if m.actionsModal == nil || m.actionsModal.state != actionsRenameSaving {
			return nil, true
		}
		if msg.err != nil {
			m.actionsModal.state = actionsRenameFailed
			m.actionsModal.renameError = "save connection name: " + msg.err.Error()
			return nil, true
		}
		m.config = msg.config
		m.actionsModal.connName = msg.config.Connections[msg.index].Name
		m.actionsModal.state = actionsRenameSuccess
		return nil, true
	case dumpFinishedMsg:
		if msg.session != m.session || m.dumpModal == nil {
			return nil, true
		}

		if msg.err != nil {
			m.dumpModal.state = dumpFailed
			m.dumpModal.err = msg.err
			return nil, true
		}

		m.dumpModal.state = dumpSucceeded
		return nil, true
	case exportFinishedMsg:
		if msg.session != m.session || m.exportModal == nil {
			return nil, true
		}

		if msg.err != nil {
			m.exportModal.state = exportFailed
			m.exportModal.err = msg.err
			return nil, true
		}

		m.exportModal.state = exportSucceeded
		return nil, true
	case editRowColumnsLoadedMsg:
		if msg.session != m.session {
			return nil, true
		}
		if msg.err != nil {
			return nil, true
		}
		if len(msg.columns) == 0 {
			return nil, true
		}
		modal := newEditRowModal(msg.table, msg.columns, msg.row)
		m.editRowModal = &modal
		m.editRowModal.clamp(m.layout)
		return m.editRowModal.focusInitial(), true
	case editRowSavedMsg:
		if msg.session != m.session || m.editRowModal == nil {
			return nil, true
		}
		if msg.err != nil {
			m.editRowModal.state = editRowFailed
			m.editRowModal.err = msg.err
			return nil, true
		}
		m.editRowModal.state = editRowSuccess
		return m.startRowLoad(m.data.offset, m.data.selected), true
	default:
		return nil, false
	}
}

func (m *Model) selectInitialRelation() {
	if m.loading || m.viewsLoading || m.materializedViewsLoading || m.navigator.hasSelection() {
		return
	}
	if len(m.navigator.visibleViews()) > 0 {
		m.navigator.section = navigatorViews
	} else if m.navigator.materializedViewsAvailable && len(m.navigator.visibleMaterializedViews()) > 0 {
		m.navigator.section = navigatorMaterializedViews
	} else {
		return
	}
	m.navigator.ensureVisible(m.layout.navigatorListRows)
}

func (m Model) updateModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case submitConnectionMsg:
		settings, err := m.modal.connectionSettings()
		if err != nil {
			m.modal.errorText = err.Error()
			return m, nil
		}
		m.modal.errorText = ""
		m.modal.connecting = true
		m.connectionAttempt++
		return m, connectConnection(m.connect, settings, m.connectionAttempt)
	case cancelConnectionMsg:
		m.modal = nil
		m.editingConnection = -1
		m.creatingConnection = false
		m.pendingConnectionIndex = -1
		if m.database == nil {
			m.activeConnectionIndex = -1
		}
		return m, nil
	case connectionFinishedMsg:
		if msg.attempt != m.connectionAttempt {
			if msg.database != nil {
				msg.database.Close()
			}
			return m, nil
		}
		m.modal.connecting = false
		if msg.err != nil {
			m.modal.errorText = msg.err.Error()
			return m, nil
		}
		if m.editingConnection >= 0 || m.creatingConnection {
			updatedConfig := m.config
			updatedConfig.Connections = slices.Clone(m.config.Connections)
			if m.editingConnection >= len(m.config.Connections) {
				msg.database.Close()
				m.modal.errorText = "selected connection no longer exists"
				return m, nil
			}
			var nextActiveIndex int
			if m.editingConnection >= 0 {
				updatedConnection := updatedConfig.Connections[m.editingConnection]
				updatedConnection.Engine = msg.settings.Engine
				updatedConnection.Settings = configSettingsFromConnectionSettings(msg.settings)
				updatedConfig.Connections[m.editingConnection] = updatedConnection
				nextActiveIndex = m.editingConnection
			} else {
				updatedConfig.Connections = append(updatedConfig.Connections, newConfigConnection(msg.settings))
				nextActiveIndex = len(updatedConfig.Connections) - 1
			}
			if err := updatedConfig.Save(); err != nil {
				msg.database.Close()
				m.modal.errorText = "save connection: " + err.Error()
				return m, nil
			}
			m.config = updatedConfig
			m.activeConnectionIndex = nextActiveIndex
			m.editingConnection = -1
			m.creatingConnection = false
		}

		// Commit the pending index for a plain select (non-edit, non-create).
		if m.pendingConnectionIndex >= 0 {
			m.activeConnectionIndex = m.pendingConnectionIndex
			m.pendingConnectionIndex = -1
		}

		if m.database != nil {
			m.database.Close()
		}
		m.database = msg.database
		m.savedConnection = msg.settings
		m.tableLoadErr = nil
		m.viewLoadErr = nil
		m.materializedViewLoadErr = nil
		m.navigator.reset()
		m.activeRelation = activeRelation{}
		m.lastNavigatorClick = navigatorClick{}
		m.navigator.setMaterializedViewsAvailable(supportsMaterializedViews(msg.database.Engine()))
		m.data.reset()
		m.query.reset(m.layout)
		m.ddlModal = nil
		m.columnsModal = nil
		m.indexesModal = nil
		m.loading = true
		m.viewsLoading = true
		m.materializedViewsLoading = true
		m.session++
		m.modal = nil
		return m, tea.Batch(m.loadDatabaseObjects(), m.startSpinner())
	default:
		modal, command := m.modal.update(msg)
		m.modal = &modal
		return m, command
	}
}

func supportsMaterializedViews(engine string) bool {
	return engine == db.EnginePostgreSQL || engine == db.EngineOracle
}

func (m Model) updateConnectionsModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cancelConnectionsMsg:
		m.connectionsModal = nil
		return m, nil
	case selectConnectionMsg:
		modal := newConnectionModal(connectionSettingsFromConfig(msg.connection))
		m.modal = &modal
		m.connectionsModal = nil
		m.editingConnection = -1
		m.creatingConnection = false
		m.pendingConnectionIndex = msg.index
		return m, func() tea.Msg { return submitConnectionMsg{} }
	case editConnectionMsg:
		modal := newConnectionModal(connectionSettingsFromConfig(msg.connection))
		m.modal = &modal
		m.connectionsModal = nil
		m.editingConnection = msg.index
		m.creatingConnection = false
		return m, m.modal.focus(0)
	case deleteConnectionMsg:
		if msg.index < 0 || msg.index >= len(m.config.Connections) {
			m.connectionsModal.deletionError = "selected connection no longer exists"
			return m, nil
		}

		updatedConfig := m.config
		updatedConfig.Connections = slices.Delete(slices.Clone(m.config.Connections), msg.index, msg.index+1)
		if err := updatedConfig.Save(); err != nil {
			m.connectionsModal.deletionError = "remove connection: " + err.Error()
			return m, nil
		}

		m.config = updatedConfig
		m.connectionsModal = nil
		if msg.index == m.activeConnectionIndex {
			if m.database != nil {
				m.database.Close()
				m.database = nil
			}
			m.savedConnection = ConnectionSettings{}
			m.activeConnectionIndex = -1
			m.loading = false
			m.tableLoadErr = nil
			m.viewsLoading = false
			m.viewLoadErr = nil
			m.materializedViewsLoading = false
			m.materializedViewLoadErr = nil
			m.navigator.reset()
			m.activeRelation = activeRelation{}
			m.lastNavigatorClick = navigatorClick{}
			m.data.reset()
			m.query.reset(m.layout)
			m.ddlModal = nil
			m.columnsModal = nil
			m.indexesModal = nil
			m.session++
		} else if msg.index < m.activeConnectionIndex {
			m.activeConnectionIndex--
		}
		return m, nil
	default:
		modal, command := m.connectionsModal.update(msg)
		m.connectionsModal = &modal
		return m, command
	}
}

func (m *Model) updateKey(msg tea.KeyPressMsg) tea.Cmd {
	m.lastNavigatorClick = navigatorClick{}
	switch {
	case key.Matches(msg, m.keys.connections):
		modal := newConnectionsModal(m.config)
		m.connectionsModal = &modal
		return nil
	case key.Matches(msg, m.keys.newConnection):
		modal := newConnectionModal(ConnectionSettings{})
		m.modal = &modal
		m.editingConnection = -1
		m.creatingConnection = true
		return m.modal.focus(0)
	case key.Matches(msg, m.keys.query):
		m.panel = panelQuery
		m.focus = focusData
		m.query.resize(m.layout)
		return m.query.focusEditor()
	case key.Matches(msg, m.keys.tableData):
		m.panel = panelData
		m.focus = focusData
		return nil
	case key.Matches(msg, m.keys.tableDDL):
		if m.navigator.selectedIsView() {
			return nil
		}
		tableName := ""
		if table, ok := m.navigator.selectedTable(); ok {
			tableName = table.Name
		}
		connName := ""
		if m.activeConnectionIndex >= 0 && m.activeConnectionIndex < len(m.config.Connections) {
			connName = m.config.Connections[m.activeConnectionIndex].Name
		}
		if tableName == "" && connName == "" {
			return nil
		}
		modal := newActionsModal(tableName, connName)
		modal.indexesAvailable = tableName != ""
		m.actionsModal = &modal
		return nil
	case key.Matches(msg, m.keys.tableSearch):
		m.focus = focusNavigator
		return m.navigator.startSearch()
	case m.panel == panelData && key.Matches(msg, m.keys.queryFocus):
		switch {
		case m.navigator.searching:
			m.navigator.finishSearch()
			m.focus = focusData
			return nil
		case m.focus == focusData:
			m.focus = focusNavigator
			return nil
		default:
			m.focus = focusData
			return nil
		}
	case m.navigator.searching:
		return m.updateNavigatorSearch(msg)
	case m.panel == panelData && m.focus == focusNavigator && key.Matches(msg, m.keys.activate):
		return m.activateHighlightedRelation()
	case key.Matches(msg, m.keys.quit) && !(m.panel == panelQuery && !m.query.resultsFocused && msg.String() == "q"):
		return tea.Quit
	case key.Matches(msg, m.keys.export) && m.panel == panelQuery:
		if m.database == nil || m.query.loading || m.query.err != nil || len(m.query.result.Columns) == 0 || strings.TrimSpace(m.query.lastExecutedSQL) == "" {
			return nil
		}
		modal := newQueryExportModal(m.query.lastExecutedSQL)
		m.exportModal = &modal
		return nil
	case m.panel == panelQuery && key.Matches(msg, m.keys.executeQuery):
		return m.startQuery()
	case m.panel == panelQuery && key.Matches(msg, m.keys.queryFocus):
		return m.query.toggleFocus()
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.up):
		m.query.scrollResults(-1)
		return nil
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.down):
		m.query.scrollResults(1)
		return nil
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.pageUp):
		m.query.scrollResults(-m.query.resultHeight(m.layout))
		return nil
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.pageDown):
		m.query.scrollResults(m.query.resultHeight(m.layout))
		return nil
	case m.panel == panelQuery && m.query.resultsFocused:
		return nil
	case m.panel == panelQuery:
		editor, command := m.query.editor.Update(msg)
		m.query.editor = editor
		return command
	case key.Matches(msg, m.keys.focusLeft):
		if m.focus == focusNavigator && m.navigator.switchSection(-1, m.layout.navigatorListRows) {
			return nil
		}
		if m.focus == focusData && m.data.columnOffset > 0 {
			m.data.scrollColumns(-1, m.layout)
			return nil
		}
		m.focus = focusNavigator
		return nil
	case key.Matches(msg, m.keys.focusRight):
		if m.focus == focusNavigator {
			if m.navigator.hasViewsSection() {
				if m.navigator.switchSection(1, m.layout.navigatorListRows) {
					return nil
				}
				return nil
			}
			m.focus = focusData
		} else {
			m.data.scrollColumns(1, m.layout)
		}
		return nil
	case m.panel == panelData && m.focus == focusData && m.activeRelation.set && m.activeRelation.item.section == navigatorTables && len(m.data.page.Rows) > 0 && !m.data.loading && key.Matches(msg, m.keys.editRow):
		return m.openEditRowModal()
	case key.Matches(msg, m.keys.up):
		if m.focus == focusNavigator {
			if m.navigator.move(-1, m.layout.navigatorListRows) {
				return nil
			}
			return nil
		}
		request, load := m.data.moveUp(m.layout)
		if load {
			return m.startRowLoad(request.offset, request.selectedRow)
		}
		return nil
	case key.Matches(msg, m.keys.down):
		if m.focus == focusNavigator {
			if m.navigator.move(1, m.layout.navigatorListRows) {
				return nil
			}
			return nil
		}
		request, load := m.data.moveDown(m.layout)
		if load {
			return m.startRowLoad(request.offset, request.selectedRow)
		}
		return nil
	case key.Matches(msg, m.keys.pageUp):
		if m.focus == focusNavigator {
			if m.navigator.move(-m.layout.navigatorListRows, m.layout.navigatorListRows) {
				return nil
			}
			return nil
		}
		if m.data.offset > 0 && !m.data.loading {
			return m.startRowLoad(max(0, m.data.offset-rowPageSize), 0)
		}
		return nil
	case key.Matches(msg, m.keys.pageDown):
		if m.focus == focusNavigator {
			if m.navigator.move(m.layout.navigatorListRows, m.layout.navigatorListRows) {
				return nil
			}
			return nil
		}
		if m.data.page.HasMore && !m.data.loading {
			return m.startRowLoad(m.data.offset+rowPageSize, 0)
		}
		return nil
	case key.Matches(msg, m.keys.home):
		if m.focus == focusNavigator && m.navigator.selectIndex(0, m.layout.navigatorListRows) {
			return nil
		}
		return nil
	case key.Matches(msg, m.keys.end):
		if m.focus == focusNavigator && m.navigator.selectIndex(len(m.navigator.visibleItems())-1, m.layout.navigatorListRows) {
			return nil
		}
		return nil
	case key.Matches(msg, m.keys.dump):
		if m.database == nil || m.loading || m.data.loading || m.query.loading {
			return nil
		}
		modal := newDumpModal(m.database.Name())
		m.dumpModal = &modal
		return nil
	case key.Matches(msg, m.keys.export):
		table, ok := m.navigator.selectedTable()
		if m.database == nil || !ok || m.loading || m.data.loading || m.query.loading {
			return nil
		}
		modal := newExportModal(table.Name)
		m.exportModal = &modal
		return nil
	default:
		return nil
	}
}

func (m *Model) updateNavigatorSearch(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "enter":
			m.navigator.finishSearch()
			return m.activateHighlightedRelation()
		case "esc":
			m.navigator.cancelSearch(m.layout.navigatorListRows)
			return nil
		}
	}

	_, command := m.navigator.updateFilter(msg, m.layout.navigatorListRows)
	return command
}

func (m *Model) updateMouseClick(msg tea.MouseClickMsg) tea.Cmd {
	if index, ok := m.navigator.itemAtMouse(msg, m.layout); ok {
		item := m.navigator.visibleItems()[index]
		doubleClick := m.lastNavigatorClick.recorded &&
			m.lastNavigatorClick.item == item &&
			time.Since(m.lastNavigatorClick.at) <= doubleClickWindow
		m.focus = focusNavigator
		m.navigator.selectIndex(index, m.layout.navigatorListRows)
		if doubleClick {
			m.lastNavigatorClick = navigatorClick{}
			if m.panel != panelData {
				return nil
			}
			return m.activateHighlightedRelation()
		}
		m.lastNavigatorClick = navigatorClick{item: item, at: time.Now(), recorded: true}
		return nil
	}
	m.lastNavigatorClick = navigatorClick{}
	if msg.Button == tea.MouseLeft && msg.X >= m.layout.navigator.width {
		m.focus = focusData
	}
	return nil
}

func (m *Model) updateMouseWheel(msg tea.MouseWheelMsg) tea.Cmd {
	m.lastNavigatorClick = navigatorClick{}
	if !m.acceptWheel(msg.Button) {
		return nil
	}
	if m.layout.mouseInNavigator(msg.X) {
		m.focus = focusNavigator
		switch msg.Button {
		case tea.MouseWheelUp:
			m.navigator.move(-1, m.layout.navigatorListRows)
		case tea.MouseWheelDown:
			m.navigator.move(1, m.layout.navigatorListRows)
		}
		return nil
	}

	if m.panel == panelQuery {
		m.focus = focusData
		switch msg.Button {
		case tea.MouseWheelUp:
			m.query.scrollResults(-1)
		case tea.MouseWheelDown:
			m.query.scrollResults(1)
		}
		return nil
	}

	m.focus = focusData
	switch msg.Button {
	case tea.MouseWheelUp:
		request, load := m.data.moveUp(m.layout)
		if load {
			return m.startRowLoad(request.offset, request.selectedRow)
		}
	case tea.MouseWheelDown:
		request, load := m.data.moveDown(m.layout)
		if load {
			return m.startRowLoad(request.offset, request.selectedRow)
		}
	}
	return nil
}

func (m *Model) activateHighlightedRelation() tea.Cmd {
	item, ok := m.navigator.selectedItem()
	if m.database == nil || !ok || (m.activeRelation.set && m.activeRelation.item == item) {
		return nil
	}
	m.activeRelation.item = item
	m.activeRelation.set = true
	return m.startRowLoad(0, 0)
}

func (m *Model) startRowLoad(offset, selectedRow int) tea.Cmd {
	if m.database == nil || !m.activeRelation.set {
		return nil
	}
	m.activeRelation.request++
	m.data.beginLoad(offset)
	return tea.Batch(loadRows(m.database, m.activeRelation.item, offset, selectedRow, m.session, m.activeRelation.request), m.startSpinner())
}

func (m *Model) startQuery() tea.Cmd {
	if m.database == nil || m.query.loading || strings.TrimSpace(m.query.editor.Value()) == "" {
		return nil
	}
	request := m.query.beginExecute(m.query.editor.Value())
	return tea.Batch(executeQuery(m.database, m.query.editor.Value(), m.session, request), m.startSpinner())
}

func (m *Model) acceptWheel(button tea.MouseButton) bool {
	now := time.Now()
	if button == m.lastWheelButton && now.Sub(m.lastWheelAt) < wheelDebounce {
		return false
	}
	m.lastWheelAt = now
	m.lastWheelButton = button
	return true
}

func (m *Model) updateDumpModal(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch m.dumpModal.state {
	case dumpConfirming:
		switch keyMsg.String() {
		case "enter":
			m.dumpModal.state = dumpRunning
			return tea.Batch(
				dumpDatabase(m.database, m.session),
				m.startSpinner(),
			)
		case "esc":
			m.dumpModal = nil
		}
	case dumpRunning:
		return nil
	case dumpSucceeded, dumpFailed:
		switch keyMsg.String() {
		case "enter", "esc":
			m.dumpModal = nil
		}
	}

	return nil
}

func (m *Model) updateExportModal(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch m.exportModal.state {
	case exportSelecting:
		switch keyMsg.String() {
		case "up", "k":
			m.exportModal.format = db.ExportTypeCSV
		case "down", "j":
			m.exportModal.format = db.ExportTypeJSON
		case "enter":
			m.exportModal.state = exportConfirming
		case "esc":
			m.exportModal = nil
		}
	case exportConfirming:
		switch keyMsg.String() {
		case "enter":
			m.exportModal.state = exportRunning
			command := exportTable(m.database, db.Table{Name: m.exportModal.tableName}, m.exportModal.format, m.session)
			if m.exportModal.source == exportQuerySource {
				command = exportQuery(m.database, m.exportModal.query, m.session)
			}
			return tea.Batch(
				command,
				m.startSpinner(),
			)
		case "esc":
			m.exportModal = nil
		}
	case exportRunning:
		return nil
	case exportSucceeded, exportFailed:
		switch keyMsg.String() {
		case "enter", "esc":
			m.exportModal = nil
		}
	}

	return nil
}

func (m *Model) updateActionsModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case submitRenameMsg:
		if m.activeConnectionIndex < 0 || m.activeConnectionIndex >= len(m.config.Connections) {
			m.actionsModal = nil
			return *m, nil
		}
		m.renameRequest++
		cloned := m.config
		cloned.Connections = slices.Clone(m.config.Connections)
		cloned.Connections[m.activeConnectionIndex].Name = msg.newName
		m.actionsModal.state = actionsRenameSaving
		return *m, saveConnectionName(cloned, m.renameRequest, m.activeConnectionIndex)
	case cancelActionsMsg:
		m.actionsModal = nil
		return *m, nil
	case selectDDLActionMsg:
		table, ok := m.navigator.selectedTable()
		if m.database == nil || !ok {
			m.actionsModal = nil
			return *m, nil
		}
		m.actionsModal = nil
		m.ddlRequest++
		modal := newDDLModal(table.Name)
		m.ddlModal = &modal
		return *m, tea.Batch(loadTableDDL(m.database, table, m.session, m.ddlRequest), m.startSpinner())
	case selectColumnsActionMsg:
		table, ok := m.navigator.selectedTable()
		if m.database == nil || !ok {
			m.actionsModal = nil
			return *m, nil
		}
		m.actionsModal = nil
		m.columnsRequest++
		modal := newColumnsModal(table.Name)
		m.columnsModal = &modal
		return *m, tea.Batch(loadColumns(m.database, table, m.session, m.columnsRequest), m.startSpinner())
	case selectIndexesActionMsg:
		table, ok := m.navigator.selectedTable()
		if m.database == nil || !ok {
			m.actionsModal = nil
			return *m, nil
		}
		m.actionsModal = nil
		m.indexesRequest++
		modal := newIndexesModal(table.Name)
		m.indexesModal = &modal
		return *m, tea.Batch(loadIndexes(m.database, table, m.session, m.indexesRequest), m.startSpinner())
	case selectRenameActionMsg:
		if m.activeConnectionIndex < 0 || m.activeConnectionIndex >= len(m.config.Connections) {
			m.actionsModal = nil
			return *m, nil
		}
		m.actionsModal.state = actionsRenameEditing
		m.actionsModal.renameInput.SetValue(m.config.Connections[m.activeConnectionIndex].Name)
		m.actionsModal.renameInput.CursorEnd()
		return *m, m.actionsModal.renameInput.Focus()
	default:
		modal, command := m.actionsModal.update(msg)
		m.actionsModal = &modal
		return *m, command
	}
}

func (m *Model) updateDDLModal(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch {
	case keyMsg.String() == "esc":
		m.ddlModal = nil
	case keyMsg.String() == "c" && !m.ddlModal.loading && m.ddlModal.err == nil && m.ddlModal.sql != "":
		m.ddlModal.copied = true
		return tea.SetClipboard(m.ddlModal.sql)
	case key.Matches(keyMsg, m.keys.up):
		m.ddlModal.scroll(-1, m.layout)
	case key.Matches(keyMsg, m.keys.down):
		m.ddlModal.scroll(1, m.layout)
	case key.Matches(keyMsg, m.keys.pageUp):
		m.ddlModal.scroll(-m.ddlModal.visibleRows(m.layout), m.layout)
	case key.Matches(keyMsg, m.keys.pageDown):
		m.ddlModal.scroll(m.ddlModal.visibleRows(m.layout), m.layout)
	case key.Matches(keyMsg, m.keys.home):
		m.ddlModal.offset = 0
	case key.Matches(keyMsg, m.keys.end):
		m.ddlModal.offset = len(m.ddlModal.lines(m.layout))
		m.ddlModal.clamp(m.layout)
	}
	return nil
}

func (m *Model) updateColumnsModal(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch {
	case keyMsg.String() == "esc":
		m.columnsModal = nil
	case key.Matches(keyMsg, m.keys.focusLeft):
		m.columnsModal.scrollFields(-1)
	case key.Matches(keyMsg, m.keys.focusRight):
		m.columnsModal.scrollFields(1)
	case key.Matches(keyMsg, m.keys.up):
		m.columnsModal.scrollRows(-1, m.layout)
	case key.Matches(keyMsg, m.keys.down):
		m.columnsModal.scrollRows(1, m.layout)
	case key.Matches(keyMsg, m.keys.pageUp):
		m.columnsModal.scrollRows(-m.columnsModal.visibleRows(m.layout), m.layout)
	case key.Matches(keyMsg, m.keys.pageDown):
		m.columnsModal.scrollRows(m.columnsModal.visibleRows(m.layout), m.layout)
	case key.Matches(keyMsg, m.keys.home):
		m.columnsModal.rowOffset = 0
	case key.Matches(keyMsg, m.keys.end):
		m.columnsModal.rowOffset = len(m.columnsModal.columns)
		m.columnsModal.clamp(m.layout)
	}
	return nil
}

func (m *Model) updateIndexesModal(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}

	switch {
	case keyMsg.String() == "esc":
		m.indexesModal = nil
	case key.Matches(keyMsg, m.keys.focusLeft):
		m.indexesModal.scrollFields(-1)
	case key.Matches(keyMsg, m.keys.focusRight):
		m.indexesModal.scrollFields(1)
	case key.Matches(keyMsg, m.keys.up):
		m.indexesModal.scrollRows(-1, m.layout)
	case key.Matches(keyMsg, m.keys.down):
		m.indexesModal.scrollRows(1, m.layout)
	case key.Matches(keyMsg, m.keys.pageUp):
		m.indexesModal.scrollRows(-m.indexesModal.visibleRows(m.layout), m.layout)
	case key.Matches(keyMsg, m.keys.pageDown):
		m.indexesModal.scrollRows(m.indexesModal.visibleRows(m.layout), m.layout)
	case key.Matches(keyMsg, m.keys.home):
		m.indexesModal.rowOffset = 0
	case key.Matches(keyMsg, m.keys.end):
		m.indexesModal.rowOffset = len(m.indexesModal.indexes)
		m.indexesModal.clamp(m.layout)
	}

	return nil
}

func (m *Model) openEditRowModal() tea.Cmd {
	if !m.activeRelation.set || m.activeRelation.item.section != navigatorTables || m.database == nil || m.data.selected >= len(m.data.page.Rows) {
		return nil
	}
	row := m.data.page.Rows[m.data.selected]
	return loadEditRowColumns(m.database, db.Table{Name: m.activeRelation.item.name}, row, m.session)
}

func (m *Model) updateEditRowModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editRowCancelMsg:
		m.editRowModal = nil
		return m, nil
	case editRowSaveMsg:
		m.editRowModal.state = editRowSaving
		return m, tea.Batch(
			saveRowEdit(m.database, msg.table, msg.setColumns, msg.whereColumns, m.session),
			m.startSpinner(),
		)
	default:
		modal, cmd := m.editRowModal.update(msg)
		m.editRowModal = &modal
		return m, cmd
	}
}
