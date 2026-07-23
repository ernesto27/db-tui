package app

import (
	"testing"

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

func TestConnectionSettingsFromConfigDefaultsLegacyEngineToPostgreSQL(t *testing.T) {
	settings := connectionSettingsFromConfig(config.Connection{})

	assert.Equal(t, db.EnginePostgreSQL, settings.Engine)
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
