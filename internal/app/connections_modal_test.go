package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestConnectionSettingsFromConfigPreservesEngine(t *testing.T) {
	connection := config.Connection{
		Engine: "mysql",
		Settings: config.Settings{
			Hostname: "127.0.0.1",
			Database: "chinook",
			Username: "db_tui",
			Password: "secret",
			Port:     "3306",
		},
	}

	settings := connectionSettingsFromConfig(connection)

	assert.Equal(t, ConnectionSettings{
		Engine:       db.EngineMySQL,
		Host:         "127.0.0.1",
		Port:         3306,
		DatabaseName: "chinook",
		Username:     "db_tui",
		Password:     "secret",
	}, settings)
}

func TestConnectionSettingsFromConfigPreservesMissingEngine(t *testing.T) {
	settings := connectionSettingsFromConfig(config.Connection{})

	assert.Empty(t, settings.Engine)
}

func TestNewConfigConnectionUsesMySQLEngine(t *testing.T) {
	connection := newConfigConnection(ConnectionSettings{
		Engine: "mysql",
		DSN:    "db_tui@tcp(localhost:3306)/chinook",
	})

	assert.Equal(t, "MySQL connection", connection.Name)
	assert.Equal(t, db.EngineMySQL, connection.Engine)
	assert.Equal(t, "db_tui@tcp(localhost:3306)/chinook", connection.Settings.DSN)
}

func TestConnectionModalKeepsEngineWithExplicitDSN(t *testing.T) {
	modal := newConnectionModal(ConnectionSettings{
		Engine: db.EngineMySQL,
		DSN:    "db_tui@tcp(localhost:3306)/chinook",
	})

	settings, err := modal.connectionSettings()

	assert.NoError(t, err)
	assert.Equal(t, db.EngineMySQL, settings.Engine)
	assert.Equal(t, "db_tui@tcp(localhost:3306)/chinook", settings.DSN)
}

func TestConnectionModalSelectsDatabaseEngine(t *testing.T) {
	modal := newConnectionModal(ConnectionSettings{})

	assert.Equal(t, db.EnginePostgreSQL, modal.engine())
	assert.Equal(t, "5432", modal.inputs[portInput].Value())

	updated, command := modal.update(keyPress(tea.KeyRight, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, db.EngineMySQL, updated.engine())
	assert.Equal(t, "3306", updated.inputs[portInput].Value())
	assert.Contains(t, updated.view(80), "MySQL")

	updated, command = updated.update(keyPress(tea.KeyRight, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, db.EngineOracle, updated.engine())
	assert.Equal(t, "1521", updated.inputs[portInput].Value())
	assert.Contains(t, updated.view(80), "Oracle")
	assert.Contains(t, updated.view(80), "Database name")
	assert.Equal(t, "oracle://user:password@host:1521/service", updated.inputs[dsnInput].Placeholder)

	updated, command = updated.update(keyPress(tea.KeyRight, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, "sqlite", updated.engine())
	view := updated.view(80)
	assert.Contains(t, view, "SQLite")
	assert.Contains(t, view, "Database file")
	assert.Equal(t, "path/to/database.db", updated.inputs[dsnInput].Placeholder)
	assert.NotContains(t, view, "Host")
	assert.NotContains(t, view, "Username")
}

func TestSQLiteConnectionSettingsRoundTripThroughDSN(t *testing.T) {
	settings := ConnectionSettings{Engine: "sqlite", DSN: "/data/reporting.db"}

	connection := newConfigConnection(settings)
	restored := connectionSettingsFromConfig(connection)

	assert.Equal(t, "sqlite", connection.Engine)
	assert.Regexp(t, `^reporting\.db-\d+$`, connection.Name)
	assert.Equal(t, "/data/reporting.db", connection.Settings.DSN)
	assert.Equal(t, settings, restored)
}

func TestConnectionsModalSearchSelectsOriginalConnection(t *testing.T) {
	modal := newConnectionsModal(config.Config{Connections: []config.Connection{
		{Name: "Analytics", Engine: db.EnginePostgreSQL},
		{Name: "Chinook", Engine: db.EnginePostgreSQL},
		{Name: "Archive", Engine: db.EnginePostgreSQL},
	}})

	assert.True(t, modal.search.Focused())
	assert.True(t, modal.searchFocused)
	assert.Equal(t, "Search", modal.search.Placeholder)

	for _, character := range "chin" {
		modal, _ = modal.update(keyPress(character, string(character), 0))
	}
	assert.Equal(t, "chin", modal.search.Value())
	assert.Len(t, modal.visibleConnections(), 1)
	assert.Contains(t, modal.view(80), "Chinook")
	assert.NotContains(t, modal.view(80), "Analytics")

	_, command := modal.update(keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)
	selected, ok := command().(selectConnectionMsg)
	require.True(t, ok)
	assert.Equal(t, 1, selected.index)
	assert.Equal(t, "Chinook", selected.connection.Name)

	modal, _ = modal.update(keyPress(tea.KeyDown, "", 0))
	assert.False(t, modal.searchFocused)
	assert.False(t, modal.search.Focused())

	modal, _ = modal.update(keyPress('f', "f", 0))
	assert.True(t, modal.searchFocused)
	assert.True(t, modal.search.Focused())
	assert.Equal(t, "chin", modal.search.Value())

	modal, _ = modal.update(keyPress(tea.KeyDown, "", 0))
	_, command = modal.update(keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)
	selected, ok = command().(selectConnectionMsg)
	require.True(t, ok)
	assert.Equal(t, 1, selected.index)
	assert.Equal(t, "Chinook", selected.connection.Name)
}

func TestConnectionsModalSearchHandlesNoMatches(t *testing.T) {
	modal := newConnectionsModal(config.Config{Connections: []config.Connection{
		{Name: "Chinook", Engine: db.EnginePostgreSQL},
	}})

	for _, character := range "missing" {
		modal, _ = modal.update(keyPress(character, string(character), 0))
	}
	assert.Empty(t, modal.visibleConnections())
	assert.Contains(t, modal.view(80), "No matching connections")

	updated, command := modal.update(keyPress(tea.KeyDown, "", 0))
	assert.True(t, updated.searchFocused)
	assert.Nil(t, command)

	_, command = updated.update(keyPress(tea.KeyEnter, "", 0))
	assert.Nil(t, command)
}
