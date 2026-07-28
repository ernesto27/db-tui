package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

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
