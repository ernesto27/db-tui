package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestBaseViewShowsSQLServerEngine(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "db_tui", engine: db.EngineSQLServer}

	view := model.baseView()

	assert.Contains(t, view.Content, "db_tui  /  SQL Server")
}

func TestEngineCapabilities(t *testing.T) {
	tests := []struct {
		engine                 string
		wantFunctions          bool
		wantMaterializedViews  bool
		wantSchemaObjectGroups bool
	}{
		{
			engine:                 db.EnginePostgreSQL,
			wantFunctions:          true,
			wantMaterializedViews:  true,
			wantSchemaObjectGroups: true,
		},
		{
			engine:                 db.EngineMySQL,
			wantFunctions:          true,
			wantMaterializedViews:  false,
			wantSchemaObjectGroups: false,
		},
		{
			engine:                 db.EngineOracle,
			wantFunctions:          true,
			wantMaterializedViews:  true,
			wantSchemaObjectGroups: false,
		},
		{
			engine:                 db.EngineSQLite,
			wantFunctions:          false,
			wantMaterializedViews:  false,
			wantSchemaObjectGroups: false,
		},
		{
			engine:                 db.EngineSQLServer,
			wantFunctions:          true,
			wantMaterializedViews:  false,
			wantSchemaObjectGroups: true,
		},
	}

	for _, test := range tests {
		t.Run(test.engine, func(t *testing.T) {
			assert.Equal(t, test.wantFunctions, supportsFunctions(test.engine), "supportsFunctions")
			assert.Equal(t, test.wantMaterializedViews, supportsMaterializedViews(test.engine), "supportsMaterializedViews")
			assert.Equal(t, test.wantSchemaObjectGroups, supportsSchemaObjectGroups(test.engine), "supportsSchemaObjectGroups")
		})
	}
}

func TestConnectionEnginesIncludeSQLServer(t *testing.T) {
	assert.Equal(t, []string{
		db.EnginePostgreSQL,
		db.EngineMySQL,
		db.EngineOracle,
		db.EngineSQLite,
		db.EngineSQLServer,
	}, connectionEngines)
}

func TestConnectionModalCyclesToSQLServer(t *testing.T) {
	modal := newConnectionModal(ConnectionSettings{})
	require.Equal(t, db.EnginePostgreSQL, modal.engine())

	// PostgreSQL -> MySQL -> Oracle -> SQLite -> SQL Server.
	for range 4 {
		modal.selectEngine(1)
	}

	assert.Equal(t, db.EngineSQLServer, modal.engine())
	assert.Equal(t, "1433", modal.inputs[portInput].Value())
	assert.Equal(t, "sqlserver://user:password@host:1433?database=name", modal.inputs[dsnInput].Placeholder)

	// Wrapping forward returns to the first engine.
	modal.selectEngine(1)
	assert.Equal(t, db.EnginePostgreSQL, modal.engine())
}

func TestConnectionModalKeepsFullFieldSetForSQLServer(t *testing.T) {
	modal := newConnectionModal(ConnectionSettings{Engine: db.EngineSQLServer})

	require.Equal(t, db.EngineSQLServer, modal.engine())
	assert.Equal(t, "1433", modal.inputs[portInput].Value())
	assert.Equal(t, []connectionInput{
		engineInput,
		hostInput,
		databaseNameInput,
		portInput,
		usernameInput,
		passwordInput,
		dsnInput,
	}, connectionInputsForEngine(db.EngineSQLServer))
}

func TestDefaultPortForEngine(t *testing.T) {
	assert.Equal(t, "5432", defaultPortForEngine(db.EnginePostgreSQL))
	assert.Equal(t, "3306", defaultPortForEngine(db.EngineMySQL))
	assert.Equal(t, "1521", defaultPortForEngine(db.EngineOracle))
	assert.Empty(t, defaultPortForEngine(db.EngineSQLite))
	assert.Equal(t, "1433", defaultPortForEngine(db.EngineSQLServer))
}

func TestObjectsShortcutOpensDatabaseExplorerModalForSQLServer(t *testing.T) {
	groups := []db.SchemaObjectGroup{{Schema: "dbo", Type: db.SchemaObjectTables}}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "db_tui", engine: db.EngineSQLServer}
	model.schemaObjectGroups = groups

	updated, command := updateModel(t, model, keyPress('o', "", tea.ModCtrl))

	require.NotNil(t, updated.databaseExplorerModal)
	assert.Nil(t, updated.objectsModal)
	assert.Equal(t, groups, updated.databaseExplorerModal.groups)
	assert.Nil(t, command)
}

func TestSQLServerConnectionRoundTripsThroughConfig(t *testing.T) {
	settings := ConnectionSettings{
		Engine:       db.EngineSQLServer,
		Host:         "127.0.0.1",
		Port:         1434,
		DatabaseName: "db_tui",
		Username:     "sa",
		Password:     "secret",
	}

	connection := newConfigConnection(settings)

	assert.Equal(t, db.EngineSQLServer, connection.Engine)
	assert.Equal(t, "db_tui (127.0.0.1)", connection.Name)
	assert.Equal(t, "1434", connection.Settings.Port)

	assert.Equal(t, settings, connectionSettingsFromConfig(connection))
}
