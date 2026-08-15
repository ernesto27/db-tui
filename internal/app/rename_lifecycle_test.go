package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestRenameChangesOnlyConnectionName(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{
				Name:   "OldName",
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
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0
	model.session = 5

	// Set up the actions modal in saving state.
	modal := newActionsModal("", "OldName")
	modal.state = actionsRenameSaving
	model.actionsModal = &modal

	// Simulate rename submission.
	model.renameRequest++
	renameReq := model.renameRequest
	cloned := model.config
	cloned.Connections = append([]config.Connection(nil), cloned.Connections...)
	cloned.Connections[0].Name = "NewName"

	model, _ = updateModel(t, model, renameRequestMsg{
		request: renameReq,
		index:   0,
		config:  cloned,
		err:     nil,
	})

	assert.Equal(t, "NewName", model.config.Connections[0].Name)
	// Other fields unchanged.
	assert.Equal(t, "postgres", model.config.Connections[0].Engine)
	assert.Equal(t, "127.0.0.1", model.config.Connections[0].Settings.Hostname)
	assert.Equal(t, "chinook", model.config.Connections[0].Settings.Database)
}

func TestRenameSaveFailurePreservesInMemoryState(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Original", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0
	model.session = 5

	// Set up the actions modal in saving state.
	modal := newActionsModal("", "Original")
	modal.state = actionsRenameSaving
	model.actionsModal = &modal

	model.renameRequest++
	renameReq := model.renameRequest

	model, _ = updateModel(t, model, renameRequestMsg{
		request: renameReq,
		index:   0,
		config:  cfg,
		err:     assert.AnError,
	})

	// Config should be unchanged in memory.
	assert.Equal(t, "Original", model.config.Connections[0].Name)
	// Modal should be in failed state.
	assert.Equal(t, actionsRenameFailed, model.actionsModal.state)
	assert.NotEmpty(t, model.actionsModal.renameError)
}

func TestRenameStaleCompletionIsIgnored(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Original", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0
	model.session = 5

	// Set up the actions modal in saving state.
	modal := newActionsModal("", "Original")
	modal.state = actionsRenameSaving
	model.actionsModal = &modal

	model.renameRequest = 5
	cloned := model.config
	cloned.Connections = append([]config.Connection(nil), cloned.Connections...)
	cloned.Connections[0].Name = "Stale"

	// Stale request number.
	model, _ = updateModel(t, model, renameRequestMsg{
		request: 3,
		index:   0,
		config:  cloned,
		err:     nil,
	})

	assert.Equal(t, "Original", model.config.Connections[0].Name)
}

func TestRenameDoesNotCloseOrReplaceDatabase(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "MyDB", Engine: "postgres"},
		},
	}
	fakeDB := &fakeDatabase{name: "chinook", engine: db.EnginePostgreSQL}
	model := New(cfg, ConnectionSettings{}, nil)
	model.database = fakeDB
	model.activeConnectionIndex = 0
	model.session = 5
	model.navigator.tables = []db.Table{{Name: "Album"}}

	// Set up the actions modal in saving state.
	modal := newActionsModal("", "MyDB")
	modal.state = actionsRenameSaving
	model.actionsModal = &modal

	model.renameRequest++
	renameReq := model.renameRequest
	cloned := model.config
	cloned.Connections = append([]config.Connection(nil), cloned.Connections...)
	cloned.Connections[0].Name = "NewName"

	model, _ = updateModel(t, model, renameRequestMsg{
		request: renameReq,
		index:   0,
		config:  cloned,
		err:     nil,
	})

	// Database session unchanged.
	assert.Equal(t, 0, fakeDB.closeCalls)
	assert.NotNil(t, model.database)
	assert.Equal(t, "chinook", model.database.Name())
	assert.Equal(t, db.EnginePostgreSQL, model.database.Engine())
	assert.Equal(t, uint64(5), model.session)
	assert.Equal(t, []db.Table{{Name: "Album"}}, model.navigator.tables)
}

func TestRenameUpdatedNameVisibleInConnectionsModal(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{
				Name:   "OldName",
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
	fakeDB := &fakeDatabase{name: "chinook", engine: db.EnginePostgreSQL}
	model := New(cfg, ConnectionSettings{}, nil)
	model.database = fakeDB
	model.activeConnectionIndex = 0
	model.session = 5
	model.navigator.tables = []db.Table{{Name: "Album"}}
	_ = model.navigator.selectIndex(0, 10)

	// Perform rename.
	modal := newActionsModal("", "OldName")
	modal.state = actionsRenameSaving
	model.actionsModal = &modal
	model.renameRequest++
	renameReq := model.renameRequest
	cloned := model.config
	cloned.Connections = append([]config.Connection(nil), cloned.Connections...)
	cloned.Connections[0].Name = "NewName"

	model, _ = updateModel(t, model, renameRequestMsg{
		request: renameReq,
		index:   0,
		config:  cloned,
		err:     nil,
	})

	// Config updated.
	assert.Equal(t, "NewName", model.config.Connections[0].Name)

	// Open connections modal to verify updated name.
	connModal := newConnectionsModal(model.config)
	model.connectionsModal = &connModal

	// Verify the modal shows the new name.
	view := model.connectionsModal.view(80)
	assert.Contains(t, view, "NewName")
	assert.NotContains(t, view, "OldName")

	// Engine and settings preserved.
	assert.Equal(t, "postgres", model.config.Connections[0].Engine)
	assert.Equal(t, "127.0.0.1", model.config.Connections[0].Settings.Hostname)
	assert.Equal(t, "chinook", model.config.Connections[0].Settings.Database)

	// Database session unchanged.
	assert.NotNil(t, model.database)
	assert.Equal(t, "chinook", model.database.Name())
	assert.Equal(t, uint64(5), model.session)
}

func TestRenameSuccessUpdatesModalConnName(t *testing.T) {
	// After a successful rename, the modal's connName must reflect the new name
	// so the selection menu label updates and reverting to the original name
	// is not treated as "no change".
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "Local", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0
	model.session = 5

	modal := newActionsModal("", "Local")
	modal.state = actionsRenameSaving
	model.actionsModal = &modal

	model.renameRequest++
	renameReq := model.renameRequest
	cloned := model.config
	cloned.Connections = append([]config.Connection(nil), cloned.Connections...)
	cloned.Connections[0].Name = "Prod"

	model, _ = updateModel(t, model, renameRequestMsg{
		request: renameReq,
		index:   0,
		config:  cloned,
		err:     nil,
	})

	// Success handler updated both config and modal connName.
	assert.Equal(t, actionsRenameSuccess, model.actionsModal.state)
	assert.Equal(t, "Prod", model.actionsModal.connName)

	// Dismiss success → back to selection.
	updatedModal, _ := model.actionsModal.update(keyPress(tea.KeyEnter, "", 0))
	assert.Equal(t, actionsSelecting, updatedModal.state)

	// Selection view must show the new name, not the old one.
	view := updatedModal.view(80)
	assert.Contains(t, view, "Prod")
	assert.NotContains(t, view, `"Local"`)

	// Entering rename again must compare against "Prod", not "Local".
	updatedModal.state = actionsRenameEditing
	updatedModal.renameInput.SetValue("Local")
	modalAfter, cmd := updatedModal.submitRename()
	// "Local" != "Prod" — should not be a no-op.
	assert.NotNil(t, cmd)
	assert.Empty(t, modalAfter.renameError)
}

func TestRenameCommandCallsConfigSave(t *testing.T) {
	cfg := config.Config{
		MaxPageSize: db.MaxPageSize,
		Connections: []config.Connection{
			{Name: "NewName", Engine: "postgres"},
		},
	}

	msg := saveConnectionName(cfg, 1, 0)()

	renameMsg, ok := msg.(renameRequestMsg)
	require.True(t, ok)
	assert.Equal(t, uint64(1), renameMsg.request)
	assert.Equal(t, 0, renameMsg.index)
	// Verify the command is structured correctly.
	assert.Equal(t, cfg, renameMsg.config)
	require.NoError(t, renameMsg.err)

	contents, err := os.ReadFile(filepath.Join(appTestHome, ".config", "db-tui", "config.json"))
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"maxPageSize":%d,"connections":[{"name":"NewName","engine":"postgres","settings":{"hostname":"","database":"","username":"","password":"","port":"","dsn":""},"status":false}]}`, db.MaxPageSize), string(contents))
}
