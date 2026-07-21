// Package app contains the root Bubble Tea application model.
package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

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

// Model is the root Bubble Tea application model.
type Model struct {
	database          db.Database
	databaseName      string
	savedConnection   ConnectionSettings
	connect           ConnectFunc
	saveConnection    SaveConnectionFunc
	modal             *connectionModal
	connectionAttempt uint64
	session           uint64

	loading      bool
	startupErr   error
	tableLoadErr error
	navigator    navigatorModel
	data         dataModel

	spinnerFrame    int
	spinnerRunning  bool
	layout          appLayout
	keys            keyMap
	focus           focusPane
	lastWheelAt     time.Time
	lastWheelButton tea.MouseButton
}

// New creates the root Bubble Tea application model for databaseName.
func New(database db.Database, databaseName string, startupErr error, savedConnection ConnectionSettings, connect ConnectFunc, saveConnection SaveConnectionFunc) Model {
	isLoading := database != nil && startupErr == nil

	return Model{
		database:        database,
		databaseName:    databaseName,
		savedConnection: savedConnection,
		connect:         connect,
		saveConnection:  saveConnection,
		loading:         isLoading,
		spinnerRunning:  isLoading,
		startupErr:      startupErr,
		session:         1,
		layout:          newAppLayout(defaultWidth, defaultHeight),
		keys:            defaultKeyMap(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.database == nil || m.startupErr != nil {
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
