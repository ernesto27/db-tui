package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/redis"
)

const (
	redisLoadTimeout = 2 * time.Minute
	redisKeyLimit    = 400
)

type redisKeysLoadedMsg struct {
	page      redis.KeyPage
	pageIndex int
	databases []int
	database  int
	session   uint64
	request   uint64
	err       error
}

type redisCommandFinishedMsg struct {
	result  redis.Result
	session uint64
	request uint64
	elapsed time.Duration
	err     error
}

func (m *Model) adoptRedis(client *redis.Client, settings ConnectionSettings) tea.Cmd {
	m.query.cancelExecution()
	if m.database != nil {
		m.database.Close()
		m.database = nil
	}
	if m.redis.client != nil {
		_ = m.redis.client.Close()
	}
	m.redis.reset()
	m.redis.client = client
	m.savedConnection = settings
	m.readOnly = false
	m.reconnecting = false
	m.reconnectErr = nil
	m.lastConnectionSaveErr = nil
	m.tableLoadErr = nil
	m.viewLoadErr = nil
	m.materializedViewLoadErr = nil
	m.functionLoadErr = nil
	m.schemaObjectGroups = nil
	m.schemaObjectGroupsLoading = false
	m.navigator.reset()
	m.activeRelation = activeRelation{}
	m.activeFunction = activeFunction{}
	m.activeExtensions = activeExtensions{}
	m.data.reset()
	m.query.reset(m.layout)
	m.query.editor.Placeholder = "Write a Redis command…"
	m.rawQueryDeleteModal = nil
	m.ddlModal = nil
	m.columnsModal = nil
	m.indexesModal = nil
	m.objectsModal = nil
	m.databaseExplorerModal = nil
	m.loading = false
	m.viewsLoading = false
	m.materializedViewsLoading = false
	m.functionsLoading = false
	m.session++
	return m.startRedisLoad()
}

func (m *Model) startRedisLoad() tea.Cmd {
	m.redis.pages = nil
	m.redis.cursor = redis.ScanCursor{}
	m.redis.pageIndex = 0
	return m.startRedisPageLoad(0)
}

func (m *Model) startRedisPageLoad(pageIndex int) tea.Cmd {
	if m.redis.client == nil {
		return nil
	}
	if pageIndex < 0 || pageIndex > len(m.redis.pages) {
		return nil
	}
	var cursor redis.ScanCursor
	if pageIndex > 0 {
		cursor = m.redis.cursor
	}
	if m.redis.loadCancel != nil {
		m.redis.loadCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisLoadTimeout)
	m.redis.loadCancel = cancel
	m.redis.request++
	request := m.redis.request
	database := m.redis.database
	session := m.session
	client := m.redis.client
	if pageIndex == 0 {
		m.data.beginLoad(0)
	} else {
		m.data.loading = true
		m.data.err = nil
	}
	return tea.Batch(func() tea.Msg {
		defer cancel()
		page, err := client.LoadKeyPage(ctx, database, redisKeyLimit, cursor)
		var databases []int
		if err == nil && pageIndex == 0 {
			databases, err = client.PopulatedDatabases(ctx)
		}
		return redisKeysLoadedMsg{page: page, pageIndex: pageIndex, databases: databases, database: database, session: session, request: request, err: err}
	}, m.startSpinner())
}

func redisRowPage(page redis.KeyPage) db.RowPage {
	rows := make([][]any, len(page.Keys))
	for index, item := range page.Keys {
		rows[index] = []any{item.Name, item.Value, item.Type, item.TTL}
	}
	return db.RowPage{Columns: []string{"key", "value", "type", "ttl"}, Rows: rows, HasMore: page.HasMore}
}

func (m *Model) showRedisPage(pageIndex, selectedRow int) tea.Cmd {
	if pageIndex < 0 || pageIndex >= len(m.redis.pages) {
		return nil
	}
	m.redis.pageIndex = pageIndex
	m.data.beginLoad(pageIndex * redisKeyLimit)
	m.data.finishLoad(m.redis.pages[pageIndex], selectedRow, nil, m.layout)
	return nil
}

func (m *Model) nextRedisPage() tea.Cmd {
	if m.data.loading || !m.data.page.HasMore {
		return nil
	}
	next := m.redis.pageIndex + 1
	if next < len(m.redis.pages) {
		return m.showRedisPage(next, 0)
	}
	return m.startRedisPageLoad(next)
}

func (m *Model) previousRedisPage(selectedRow int) tea.Cmd {
	if m.data.loading {
		return nil
	}
	return m.showRedisPage(m.redis.pageIndex-1, selectedRow)
}

func (m *Model) selectRedisDatabase(database int) tea.Cmd {
	if m.redis.client == nil || database < 0 || database >= m.redis.client.DatabaseCount() {
		return nil
	}
	m.redis.highlighted = database
	m.ensureRedisHighlightedVisible()
	if m.redis.database == database && !m.data.loading {
		return nil
	}
	m.redis.database = database
	m.data.resetColumnWidths()
	return m.startRedisLoad()
}

func (m *Model) moveRedisHighlight(delta int) {
	if len(m.redis.databases) == 0 {
		return
	}
	index := slices.Index(m.redis.databases, m.redis.highlighted)
	m.redis.highlighted = m.redis.databases[min(max(index+delta, 0), len(m.redis.databases)-1)]
	m.ensureRedisHighlightedVisible()
}

func (m *Model) ensureRedisHighlightedVisible() {
	if len(m.redis.databases) == 0 {
		m.redis.navigatorOffset = 0
		return
	}
	index := slices.Index(m.redis.databases, m.redis.highlighted)
	if index < 0 {
		index = 0
		m.redis.highlighted = m.redis.databases[0]
	}
	visible := max(1, m.layout.navigatorListRows)
	if index < m.redis.navigatorOffset {
		m.redis.navigatorOffset = index
	}
	if index >= m.redis.navigatorOffset+visible {
		m.redis.navigatorOffset = index - visible + 1
	}
	m.redis.navigatorOffset = min(max(m.redis.navigatorOffset, 0), max(0, len(m.redis.databases)-visible))
}

func (m Model) redisNavigatorView() string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("● Redis"),
		"", "", lipgloss.NewStyle().Bold(true).Foreground(colorTextMuted).Render("Databases"),
	}
	count := len(m.redis.databases)
	for index := m.redis.navigatorOffset; index < min(count, m.redis.navigatorOffset+m.layout.navigatorListRows); index++ {
		marker := "  "
		style := lipgloss.NewStyle().Width(max(1, m.layout.navigator.width-4))
		if m.redis.databases[index] == m.redis.highlighted {
			marker = "> "
			style = style.Bold(true).Foreground(colorSelectionForeground).Background(colorSelectionBackground)
		}
		label := fmt.Sprintf("db%d", m.redis.databases[index])
		lines = append(lines, style.Render(marker+label))
	}
	if count == 0 {
		message := "  No databases contain keys"
		if m.data.loading {
			message = "  Loading databases…"
		}
		lines = append(lines, message)
	}
	return panelStyle(m.layout.navigator.width, m.layout.navigator.height, m.focus == focusNavigator).Render(strings.Join(lines, "\n"))
}

func (m Model) redisDataStatus() dataStatus {
	return dataStatus{
		tableName:   fmt.Sprintf("db%d", m.redis.database),
		active:      true,
		spinner:     m.spinner(),
		loadingText: "Loading Redis keys…",
		errorText:   "Unable to load Redis keys:",
		emptyText:   "No keys in this Redis database.",
	}
}

func (m *Model) startRedisCommand() tea.Cmd {
	command := m.query.executableSQL()
	if m.redis.client == nil || m.query.loading || strings.TrimSpace(command) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.config.QueryTimeout())
	request := m.query.beginExecute(command)
	m.query.cancel = cancel
	client := m.redis.client
	database := m.redis.database
	session := m.session
	started := m.query.executionStartedAt
	return tea.Batch(func() tea.Msg {
		result, err := client.Execute(ctx, database, command)
		return redisCommandFinishedMsg{result: result, session: session, request: request, err: err, elapsed: time.Since(started)}
	}, queryElapsedTick(session, request, started))
}

func (m *Model) updateRedisKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.shortcuts):
		modal := newShortcutsModal(m.layout, true)
		m.shortcutsModal = &modal
	case key.Matches(msg, m.keys.settings):
		modal := newSettingsModal(m.config.PageSize(), m.config.QueryTimeoutText())
		m.settingsModal = &modal
		return m.settingsModal.focusInput(0)
	case key.Matches(msg, m.keys.connections):
		modal := newConnectionsModal(m.config)
		m.connectionsModal = &modal
	case m.panel == panelQuery && key.Matches(msg, m.keys.newConnection):
		m.query.reset(m.layout)
		m.query.editor.Placeholder = "Write a Redis command…"
		return m.query.focusEditor()
	case key.Matches(msg, m.keys.newConnection):
		modal := newConnectionModal(ConnectionSettings{})
		m.modal = &modal
		m.editingConnection = -1
		m.creatingConnection = true
		return m.modal.focus(0)
	case key.Matches(msg, m.keys.query):
		m.panel = panelQuery
		m.focus = focusData
		return m.query.focusEditor()
	case key.Matches(msg, m.keys.tableData):
		m.panel = panelData
		m.focus = focusData
	case key.Matches(msg, m.keys.quit) && !(m.panel == panelQuery && !m.query.resultsFocused && msg.String() == "q"):
		return tea.Quit
	case m.panel == panelQuery && key.Matches(msg, m.keys.executeQuery):
		return m.startRedisCommand()
	case m.panel == panelQuery && key.Matches(msg, m.keys.queryFocus):
		return m.query.toggleFocus()
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.up):
		m.query.scrollResults(-1, m.layout)
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.down):
		m.query.scrollResults(1, m.layout)
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.pageUp):
		m.query.scrollResults(-m.query.resultHeight(m.layout), m.layout)
	case m.panel == panelQuery && m.query.resultsFocused && key.Matches(msg, m.keys.pageDown):
		m.query.scrollResults(m.query.resultHeight(m.layout), m.layout)
	case m.panel == panelQuery && m.query.resultsFocused:
		return nil
	case m.panel == panelQuery:
		return m.updateQueryEditor(msg)
	case key.Matches(msg, m.keys.queryFocus):
		if m.focus == focusData {
			m.focus = focusNavigator
		} else {
			m.focus = focusData
		}
	case m.focus == focusNavigator && key.Matches(msg, m.keys.reconnect):
		return m.startReconnect()
	case m.focus == focusNavigator && key.Matches(msg, m.keys.activate):
		return m.selectRedisDatabase(m.redis.highlighted)
	case m.focus == focusNavigator && key.Matches(msg, m.keys.up):
		m.moveRedisHighlight(-1)
	case m.focus == focusNavigator && key.Matches(msg, m.keys.down):
		m.moveRedisHighlight(1)
	case m.focus == focusNavigator && key.Matches(msg, m.keys.pageUp):
		m.moveRedisHighlight(-m.layout.navigatorListRows)
	case m.focus == focusNavigator && key.Matches(msg, m.keys.pageDown):
		m.moveRedisHighlight(m.layout.navigatorListRows)
	case m.focus == focusNavigator && key.Matches(msg, m.keys.home):
		if len(m.redis.databases) > 0 {
			m.redis.highlighted = m.redis.databases[0]
			m.ensureRedisHighlightedVisible()
		}
	case m.focus == focusNavigator && key.Matches(msg, m.keys.end):
		if len(m.redis.databases) > 0 {
			m.redis.highlighted = m.redis.databases[len(m.redis.databases)-1]
			m.ensureRedisHighlightedVisible()
		}
	case key.Matches(msg, m.keys.focusLeft):
		if m.focus == focusData && m.data.columnOffset > 0 {
			m.data.scrollColumns(-1, m.layout)
		} else {
			m.focus = focusNavigator
		}
	case key.Matches(msg, m.keys.focusRight):
		if m.focus == focusNavigator {
			m.focus = focusData
		} else {
			m.data.scrollColumns(1, m.layout)
		}
	case key.Matches(msg, m.keys.up):
		_, load := m.data.moveUp(m.layout, redisKeyLimit)
		if load {
			return m.previousRedisPage(redisKeyLimit - 1)
		}
	case key.Matches(msg, m.keys.down):
		_, load := m.data.moveDown(m.layout, redisKeyLimit)
		if load {
			return m.nextRedisPage()
		}
	case key.Matches(msg, m.keys.pageUp):
		return m.previousRedisPage(0)
	case key.Matches(msg, m.keys.pageDown):
		return m.nextRedisPage()
	case key.Matches(msg, m.keys.refreshTable):
		return m.startRedisLoad()
	}
	return nil
}
