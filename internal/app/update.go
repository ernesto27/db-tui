package app

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
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

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m, m.updateKey(msg)
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
		if m.loading || m.data.loading {
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
		m.data.reset()
		if msg.err == nil {
			if _, ok := m.navigator.selectedTable(); ok {
				return m.startRowLoad(0, 0), true
			}
		}
		return nil, true
	case rowsLoadedMsg:
		table, ok := m.navigator.selectedTable()
		if msg.session != m.session || !ok || msg.tableName != table.Name || msg.offset != m.data.offset {
			return nil, true
		}
		m.data.finishLoad(msg.page, msg.selectedRow, msg.err, m.layout)
		return nil, true
	case tea.WindowSizeMsg:
		m.layout = newAppLayout(msg.Width, msg.Height)
		m.navigator.ensureVisible(m.layout.navigatorListRows)
		m.data.columnOffset = min(m.data.columnOffset, m.data.maxColumnOffset())
		m.data.ensureSelectedVisible(m.layout)
		return nil, true
	default:
		return nil, false
	}
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
		return m, connectAndSave(m.connect, m.saveConnection, settings, m.connectionAttempt)
	case cancelConnectionMsg:
		m.modal = nil
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

		if m.database != nil {
			m.database.Close()
		}
		m.database = msg.database
		m.databaseName = msg.database.Name()
		m.savedConnection = msg.settings
		m.startupErr = nil
		m.tableLoadErr = nil
		m.navigator.reset()
		m.data.reset()
		m.loading = true
		m.session++
		m.modal = nil
		return m, tea.Batch(loadTables(m.database, m.session), m.startSpinner())
	default:
		modal, command := m.modal.update(msg)
		m.modal = &modal
		return m, command
	}
}

func (m *Model) updateKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.connect):
		modal := newConnectionModal(m.savedConnection)
		m.modal = &modal
		return m.modal.focus(0)
	case key.Matches(msg, m.keys.quit):
		return tea.Quit
	case key.Matches(msg, m.keys.focusLeft):
		if m.focus == focusData && m.data.columnOffset > 0 {
			m.data.scrollColumns(-1, m.layout)
			return nil
		}
		m.focus = focusNavigator
		return nil
	case key.Matches(msg, m.keys.focusRight):
		if m.focus == focusNavigator {
			m.focus = focusData
		} else {
			m.data.scrollColumns(1, m.layout)
		}
		return nil
	case key.Matches(msg, m.keys.up):
		if m.focus == focusNavigator {
			if m.navigator.move(-1, m.layout.navigatorListRows) {
				return m.startRowLoad(0, 0)
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
				return m.startRowLoad(0, 0)
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
				return m.startRowLoad(0, 0)
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
				return m.startRowLoad(0, 0)
			}
			return nil
		}
		if m.data.page.HasMore && !m.data.loading {
			return m.startRowLoad(m.data.offset+rowPageSize, 0)
		}
		return nil
	case key.Matches(msg, m.keys.home):
		if m.focus == focusNavigator && m.navigator.selectIndex(0, m.layout.navigatorListRows) {
			return m.startRowLoad(0, 0)
		}
		return nil
	case key.Matches(msg, m.keys.end):
		if m.focus == focusNavigator && m.navigator.selectIndex(len(m.navigator.tables)-1, m.layout.navigatorListRows) {
			return m.startRowLoad(0, 0)
		}
		return nil
	default:
		return nil
	}
}

func (m *Model) updateMouseClick(msg tea.MouseClickMsg) tea.Cmd {
	if index, ok := m.navigator.tableAtMouse(msg, m.layout); ok {
		m.focus = focusNavigator
		if m.navigator.selectIndex(index, m.layout.navigatorListRows) {
			return m.startRowLoad(0, 0)
		}
		return nil
	}
	if msg.Button == tea.MouseLeft && msg.X >= m.layout.navigator.width {
		m.focus = focusData
	}
	return nil
}

func (m *Model) updateMouseWheel(msg tea.MouseWheelMsg) tea.Cmd {
	if !m.acceptWheel(msg.Button) {
		return nil
	}
	if m.layout.mouseInNavigator(msg.X) {
		m.focus = focusNavigator
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.navigator.move(-1, m.layout.navigatorListRows) {
				return m.startRowLoad(0, 0)
			}
		case tea.MouseWheelDown:
			if m.navigator.move(1, m.layout.navigatorListRows) {
				return m.startRowLoad(0, 0)
			}
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

func (m *Model) startRowLoad(offset, selectedRow int) tea.Cmd {
	table, ok := m.navigator.selectedTable()
	if m.database == nil || !ok {
		return nil
	}
	m.data.beginLoad(offset)
	return tea.Batch(loadRows(m.database, table, offset, selectedRow, m.session), m.startSpinner())
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
