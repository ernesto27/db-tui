// Package app contains the root Bubble Tea application model.
package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	defaultWidth      = 100
	defaultHeight     = 24
	wheelDebounce     = 50 * time.Millisecond
	doubleClickWindow = 500 * time.Millisecond
)

type focusPane uint8

const (
	focusNavigator focusPane = iota
	focusData
)

type rightPanel uint8

const (
	panelData rightPanel = iota
	panelQuery
)

type activeRelation struct {
	item    navigatorItem
	request uint64
	set     bool
}

type navigatorClick struct {
	item     navigatorItem
	at       time.Time
	recorded bool
}

// Model is the root Bubble Tea application model.
type Model struct {
	database               db.Database
	savedConnection        ConnectionSettings
	sqlScripts             ListSqlScript
	activeConnectionIndex  int
	pendingConnectionIndex int
	connect                ConnectFunc
	modal                  *connectionModal
	connectionsModal       *connectionsModal
	settingsModal          *settingsModal
	editingConnection      int
	creatingConnection     bool
	connectionAttempt      uint64
	session                uint64

	loading                  bool
	tableLoadErr             error
	viewsLoading             bool
	viewLoadErr              error
	materializedViewsLoading bool
	materializedViewLoadErr  error
	navigator                navigatorModel
	activeRelation           activeRelation
	data                     dataModel
	panel                    rightPanel
	query                    queryModel
	sqlScriptsModal          *sqlScriptsModal
	sqlScriptsRequest        uint64

	spinnerFrame       int
	spinnerRunning     bool
	layout             appLayout
	keys               keyMap
	focus              focusPane
	lastWheelAt        time.Time
	lastWheelButton    tea.MouseButton
	lastNavigatorClick navigatorClick

	config config.Config

	dumpModal      *dumpModal
	exportModal    *exportModal
	ddlModal       *ddlModal
	ddlRequest     uint64
	columnsModal   *columnsModal
	columnsRequest uint64
	actionsModal   *actionsModal
	renameRequest  uint64
	indexesModal   *indexesModal
	indexesRequest uint64

	editRowModal   *editRowModal
	deleteRowModal *deleteRowModal
}

// New creates the root Bubble Tea application model.
func New(config config.Config, savedConnection ConnectionSettings, connect ConnectFunc) Model {
	layout := newAppLayout(defaultWidth, defaultHeight)
	navigator := newNavigatorModel()
	navigator.resize(layout)
	return Model{
		savedConnection:        savedConnection,
		sqlScripts:             ListSqlScript{},
		activeConnectionIndex:  -1,
		pendingConnectionIndex: -1,
		connect:                connect,
		config:                 config,
		editingConnection:      -1,
		session:                1,
		layout:                 layout,
		keys:                   defaultKeyMap(),
		navigator:              navigator,
		query:                  newQueryModel(layout),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.database == nil {
		return nil
	}
	return tea.Batch(m.loadDatabaseObjects(), spinnerTick())
}

func (m Model) loadDatabaseObjects() tea.Cmd {
	return tea.Batch(
		loadTables(m.database, m.session),
		loadViews(m.database, m.session),
		loadMaterializedViews(m.database, m.session),
	)
}

// Close releases the current database session, if any.
func (m Model) Close() {
	if m.database != nil {
		m.database.Close()
	}
}

var _ tea.Model = Model{}
