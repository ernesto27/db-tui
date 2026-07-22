// Package app contains the root Bubble Tea application model.
package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	defaultWidth  = 100
	defaultHeight = 24
	wheelDebounce = 50 * time.Millisecond
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

// Model is the root Bubble Tea application model.
type Model struct {
	database           db.Database
	databaseName       string
	savedConnection    ConnectionSettings
	connect            ConnectFunc
	modal              *connectionModal
	connectionsModal   *connectionsModal
	editingConnection  int
	creatingConnection bool
	connectionAttempt  uint64
	session            uint64

	loading      bool
	tableLoadErr error
	navigator    navigatorModel
	data         dataModel
	panel        rightPanel
	query        queryModel

	spinnerFrame    int
	spinnerRunning  bool
	layout          appLayout
	keys            keyMap
	focus           focusPane
	lastWheelAt     time.Time
	lastWheelButton tea.MouseButton

	config config.Config
}

// New creates the root Bubble Tea application model.
func New(config config.Config, savedConnection ConnectionSettings, connect ConnectFunc) Model {
	return Model{
		savedConnection:   savedConnection,
		connect:           connect,
		config:            config,
		editingConnection: -1,
		session:           1,
		layout:            newAppLayout(defaultWidth, defaultHeight),
		keys:              defaultKeyMap(),
		query:             newQueryModel(newAppLayout(defaultWidth, defaultHeight)),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.database == nil {
		return nil
	}
	return tea.Batch(loadTables(m.database, m.session), spinnerTick())
}

// Close releases the current database session, if any.
func (m Model) Close() {
	if m.database != nil {
		m.database.Close()
	}
}

var _ tea.Model = Model{}
