package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
)

func TestSelectConnectionRecordsActiveIndex(t *testing.T) {
	model := New(config.Config{
		Connections: []config.Connection{
			{Name: "First", Engine: "postgres"},
			{
				Name:   "Second",
				Engine: "postgres",
				Settings: config.Settings{
					Hostname: "127.0.0.1",
					Database: "chinook",
					Username: "db_tui",
					Port:     "5433",
				},
			},
			{Name: "Third", Engine: "postgres"},
		},
	}, ConnectionSettings{}, nil)
	model.session = 10

	// Open the connections modal so selectConnectionMsg is routed.
	modal := newConnectionsModal(model.config)
	model.connectionsModal = &modal

	// Select the second connection — sets pending, not active.
	model, _ = updateModel(t, model, selectConnectionMsg{
		index:      1,
		connection: model.config.Connections[1],
	})

	// Pending set, active not yet committed.
	assert.Equal(t, 1, model.pendingConnectionIndex)
	assert.Equal(t, -1, model.activeConnectionIndex)

	// Simulate successful connect to commit the pending index.
	model, command := updateModel(t, model, submitConnectionMsg{})
	require.NotNil(t, command)

	model, _ = updateModel(t, model, connectionFinishedMsg{
		database: &fakeDatabase{name: "second"},
		settings: ConnectionSettings{Engine: "postgres", DSN: "postgres://second"},
		attempt:  model.connectionAttempt,
	})

	assert.Equal(t, 1, model.activeConnectionIndex)
	assert.Equal(t, -1, model.pendingConnectionIndex)
}

func TestEditConnectionSetsActiveIndex(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "First", Engine: "postgres"},
			{
				Name:   "Second",
				Engine: "postgres",
				Settings: config.Settings{
					Hostname: "127.0.0.1",
					Database: "chinook",
					Username: "db_tui",
					Port:     "5433",
				},
			},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.activeConnectionIndex = 0
	model.session = 10

	// Edit the second connection via the connections modal.
	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal
	model, _ = updateModel(t, model, editConnectionMsg{
		index:      1,
		connection: cfg.Connections[1],
	})

	// Now the modal should be open and editingConnection set.
	assert.NotNil(t, model.modal)
	assert.Equal(t, 1, model.editingConnection)

	model, command := updateModel(t, model, submitConnectionMsg{})
	require.NotNil(t, command)

	connFinished := connectionFinishedMsg{
		database: &fakeDatabase{name: "edited"},
		settings: ConnectionSettings{Engine: "postgres", DSN: "postgres://edited"},
		attempt:  model.connectionAttempt,
	}
	model, _ = updateModel(t, model, connFinished)

	// Editing connection 1 and connecting successfully makes it the active entry.
	assert.Equal(t, 1, model.activeConnectionIndex)
}

func TestNewConnectionRecordsAppendedIndex(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Existing", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10

	// Create a new connection.
	modal := newConnectionModal(ConnectionSettings{Engine: "postgres", DSN: "postgres://new"})
	model.modal = &modal
	model.creatingConnection = true

	model, command := updateModel(t, model, submitConnectionMsg{})
	require.NotNil(t, command)

	connFinished := connectionFinishedMsg{
		database: &fakeDatabase{name: "new"},
		settings: ConnectionSettings{Engine: "postgres", DSN: "postgres://new"},
		attempt:  1,
	}
	model, _ = updateModel(t, model, connFinished)

	// New connection should be at index 1.
	assert.Equal(t, 1, model.activeConnectionIndex)
}

func TestDuplicateConnectionSettingsTracksCorrectIndex(t *testing.T) {
	// Two connections with identical settings - the active index must be the one selected.
	cfg := config.Config{
		Connections: []config.Connection{
			{
				Name:   "Local A",
				Engine: "postgres",
				Settings: config.Settings{
					Hostname: "127.0.0.1",
					Database: "chinook",
					Username: "db_tui",
					Port:     "5433",
				},
			},
			{
				Name:   "Local B",
				Engine: "postgres",
				Settings: config.Settings{
					Hostname: "127.0.0.1",
					Database: "chinook",
					Username: "db_tui",
					Port:     "5433",
				},
			},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10

	// Open the connections modal so selectConnectionMsg is routed.
	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal

	// Select the second connection (index 1) — sets pending.
	model, cmd := updateModel(t, model, selectConnectionMsg{
		index:      1,
		connection: model.config.Connections[1],
	})
	require.NotNil(t, cmd)

	model, cmd = updateModel(t, model, submitConnectionMsg{})
	require.NotNil(t, cmd)

	model, _ = updateModel(t, model, connectionFinishedMsg{
		database: &fakeDatabase{name: "localb"},
		settings: ConnectionSettings{Engine: "postgres", DSN: "postgres://localb"},
		attempt:  model.connectionAttempt,
	})

	assert.Equal(t, 1, model.activeConnectionIndex)
}

func TestActiveConnectionIndexDefaultsToMinusOne(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	assert.Equal(t, -1, model.activeConnectionIndex)
}

func TestSelectingConnectionViaModalMessage(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{
				Name:   "Only",
				Engine: "postgres",
				Settings: config.Settings{
					Hostname: "127.0.0.1",
					Database: "chinook",
					Username: "db_tui",
					Port:     "5433",
				},
			},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10

	// Open connections modal so message is routed correctly.
	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal

	// Select sets pending, not active.
	model, cmd := updateModel(t, model, selectConnectionMsg{
		index:      0,
		connection: model.config.Connections[0],
	})
	require.NotNil(t, cmd)
	assert.Equal(t, 0, model.pendingConnectionIndex)
	assert.Equal(t, -1, model.activeConnectionIndex)

	// Connection success commits the pending index.
	model, cmd = updateModel(t, model, submitConnectionMsg{})
	require.NotNil(t, cmd)
	model, _ = updateModel(t, model, connectionFinishedMsg{
		database: &fakeDatabase{name: "only"},
		settings: ConnectionSettings{Engine: "postgres", DSN: "postgres://only"},
		attempt:  model.connectionAttempt,
	})

	assert.Equal(t, 0, model.activeConnectionIndex)
	assert.Equal(t, -1, model.pendingConnectionIndex)
}

func TestCancelConnectionModalPreservesIndex(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.activeConnectionIndex = 2
	model.database = &fakeDatabase{name: "connected"}
	model.session = 10

	modal := newConnectionModal(ConnectionSettings{})
	model.modal = &modal

	model, _ = updateModel(t, model, cancelConnectionMsg{})

	assert.Nil(t, model.modal)
	assert.Equal(t, 2, model.activeConnectionIndex)
}

func TestDeleteConnectionBeforeActiveDecrementsIndex(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Zero", Engine: "postgres"},
			{Name: "One", Engine: "postgres"},
			{Name: "Two", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10
	model.activeConnectionIndex = 2
	model.database = &fakeDatabase{name: "active"}

	// Open connections modal for message routing.
	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal

	// Delete connection at index 0 (before active).
	model, command := updateModel(t, model, deleteConnectionMsg{
		index:      0,
		connection: cfg.Connections[0],
	})

	assert.Nil(t, command)
	// Active index should decrement from 2 to 1.
	assert.Equal(t, 1, model.activeConnectionIndex)
	// Database session should remain open.
	assert.NotNil(t, model.database)
}

func TestDeleteActiveConnectionClearsIndexAndSession(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Zero", Engine: "postgres"},
			{Name: "One", Engine: "postgres"},
		},
	}
	db := &fakeDatabase{name: "active", engine: "postgres"}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10
	model.activeConnectionIndex = 0
	model.database = db

	// Open connections modal for message routing.
	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal

	// Delete the active connection.
	model, command := updateModel(t, model, deleteConnectionMsg{
		index:      0,
		connection: cfg.Connections[0],
	})

	assert.Nil(t, command)
	// Active index should be cleared.
	assert.Equal(t, -1, model.activeConnectionIndex)
	// Database session should be closed.
	assert.Nil(t, model.database)
	assert.Equal(t, 1, db.closeCalls)
}

func TestDeleteConnectionAfterActivePreservesIndex(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Zero", Engine: "postgres"},
			{Name: "One", Engine: "postgres"},
			{Name: "Two", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10
	model.activeConnectionIndex = 0
	model.database = &fakeDatabase{name: "active"}

	// Open connections modal for message routing.
	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal

	// Delete connection at index 2 (after active).
	model, command := updateModel(t, model, deleteConnectionMsg{
		index:      2,
		connection: cfg.Connections[2],
	})

	assert.Nil(t, command)
	assert.Equal(t, 0, model.activeConnectionIndex)
	assert.NotNil(t, model.database)
}

func TestDeleteConnectionWithoutActiveIndex(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Only", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10
	model.activeConnectionIndex = -1

	// Open connections modal for message routing.
	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal

	model, command := updateModel(t, model, deleteConnectionMsg{
		index:      0,
		connection: cfg.Connections[0],
	})

	assert.Nil(t, command)
	assert.Equal(t, -1, model.activeConnectionIndex)
}

func TestDeleteActiveIndexWithoutDatabaseDoesNotPanic(t *testing.T) {
	// Regression: activeConnectionIndex set but database is nil (failed connect
	// followed by Esc). Deleting the entry must not call Close on nil.
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Only", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10
	model.activeConnectionIndex = 0
	// database intentionally nil — simulates failed connect then cancel.

	connModal := newConnectionsModal(cfg)
	model.connectionsModal = &connModal

	// Must not panic.
	model, command := updateModel(t, model, deleteConnectionMsg{
		index:      0,
		connection: cfg.Connections[0],
	})

	assert.Nil(t, command)
	assert.Equal(t, -1, model.activeConnectionIndex)
	assert.Nil(t, model.database)
}

func TestCancelAfterFailedSelectResetsActiveIndex(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "DB1", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10
	model.pendingConnectionIndex = 0
	model.database = nil // connection was never established

	// Open connection modal (simulates selectConnectionMsg flow).
	modal := newConnectionModal(ConnectionSettings{})
	model.modal = &modal

	model, _ = updateModel(t, model, cancelConnectionMsg{})

	assert.Nil(t, model.modal)
	assert.Equal(t, -1, model.activeConnectionIndex)
	assert.Equal(t, -1, model.pendingConnectionIndex)
}

func TestCancelAfterFailedSelectWithDatabasePreservesIndex(t *testing.T) {
	// If a database IS connected, cancel should not reset the index.
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "DB1", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.session = 10
	model.activeConnectionIndex = 0
	model.database = &fakeDatabase{name: "connected"}

	modal := newConnectionModal(ConnectionSettings{})
	model.modal = &modal

	model, _ = updateModel(t, model, cancelConnectionMsg{})

	assert.Nil(t, model.modal)
	assert.Equal(t, 0, model.activeConnectionIndex)
}
