package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestSQLScriptsHistoryLoadsMostRecentlyUpdatedScript(t *testing.T) {
	const connectionName = "history modal test"
	model := New(config.Config{Connections: []config.Connection{{Name: connectionName}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0
	model.panel = panelQuery

	require.NoError(t, model.sqlScripts.createByConnection(connectionName, "SELECT older;"))
	require.NoError(t, model.sqlScripts.createByConnection(connectionName, "\n  SELECT newest;"))
	directory, err := sqlScriptsDirectory(connectionName)
	require.NoError(t, err)
	scripts, err := model.sqlScripts.getList(connectionName)
	require.NoError(t, err)
	require.Len(t, scripts, 2)
	for _, script := range scripts {
		modifiedAt := time.Now()
		if script.content == "SELECT older;" {
			modifiedAt = modifiedAt.Add(-time.Hour)
		}
		require.NoError(t, os.Chtimes(filepath.Join(directory, script.name), modifiedAt, modifiedAt))
	}

	updated, command := updateModel(t, model, keyPress('h', "", tea.ModCtrl))
	require.NotNil(t, updated.sqlScriptsModal)
	loaded, ok := command().(sqlScriptsLoadedMsg)
	require.True(t, ok)
	updated, _ = updateModel(t, updated, loaded)
	require.NotNil(t, updated.sqlScriptsModal)
	assert.Contains(t, updated.sqlScriptsModal.view(updated.layout), "SELECT newest;")

	updated, command = updateModel(t, updated, keyPress(tea.KeyEnter, "", 0))
	assert.NotNil(t, command)
	assert.Nil(t, updated.sqlScriptsModal)
	assert.Equal(t, "\n  SELECT newest;", updated.query.editor.Value())
	assert.NotEmpty(t, updated.query.loadedScriptName)
}

func TestSQLScriptsHistoryIsScopedToRawQueryPanel(t *testing.T) {
	model := New(config.Config{Connections: []config.Connection{{Name: "scoped history"}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0

	updated, command := updateModel(t, model, keyPress('h', "", tea.ModCtrl))
	assert.Nil(t, command)
	assert.Nil(t, updated.sqlScriptsModal)
}

func TestSQLScriptsModalKeepsSelectedScriptVisible(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.layout = newAppLayout(100, 16)
	modal := newSQLScriptsModal("small terminal", 1)
	scripts := make([]SqlScript, 8)
	for index := range scripts {
		scripts[index] = SqlScript{content: fmt.Sprintf("SELECT %d;", index)}
	}
	modal.finish(scripts, nil, model.layout)
	model.sqlScriptsModal = &modal

	for range 5 {
		var command tea.Cmd
		model, command = updateModel(t, model, keyPress(tea.KeyDown, "", 0))
		assert.Nil(t, command)
	}

	require.NotNil(t, model.sqlScriptsModal)
	assert.Equal(t, 5, model.sqlScriptsModal.selected)
	assert.Greater(t, model.sqlScriptsModal.offset, 0)
	view := model.sqlScriptsModal.view(model.layout)
	assert.Contains(t, view, "SELECT 5;")
	assert.NotContains(t, view, "SELECT 0;")
	assert.Contains(t, view, "Enter load")
}

func TestSQLScriptsModalIgnoresStaleLoadResult(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newSQLScriptsModal("same connection", 2)
	model.sqlScriptsModal = &modal

	updated, command := updateModel(t, model, sqlScriptsLoadedMsg{
		connectionName: "same connection",
		request:        1,
		scripts:        []SqlScript{{content: "SELECT stale;"}},
	})
	assert.Nil(t, command)
	require.NotNil(t, updated.sqlScriptsModal)
	assert.True(t, updated.sqlScriptsModal.loading)
	assert.Empty(t, updated.sqlScriptsModal.scripts)

	updated, command = updateModel(t, updated, sqlScriptsLoadedMsg{
		connectionName: "same connection",
		request:        2,
		scripts:        []SqlScript{{content: "SELECT current;"}},
	})
	assert.Nil(t, command)
	require.NotNil(t, updated.sqlScriptsModal)
	assert.False(t, updated.sqlScriptsModal.loading)
	assert.Equal(t, "SELECT current;", updated.sqlScriptsModal.scripts[0].content)
}

func TestRawQuerySavesNewScriptAndUpdatesLoadedScript(t *testing.T) {
	const connectionName = "save and update test"
	model := New(config.Config{Connections: []config.Connection{{Name: connectionName}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0
	model.query.editor.SetValue("SELECT original;")

	command := model.startQuery()
	require.NotNil(t, command)
	batch, ok := command().(tea.BatchMsg)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(batch), 2)
	saved, ok := batch[len(batch)-1]().(sqlScriptSavedMsg)
	require.True(t, ok)
	require.NoError(t, saved.err)

	scripts, err := model.sqlScripts.getList(connectionName)
	require.NoError(t, err)
	require.Len(t, scripts, 1)
	assert.Equal(t, "SELECT original;", scripts[0].content)

	model.query.loading = false
	model.query.loadedScriptName = scripts[0].name
	model.query.editor.SetValue("SELECT revised;")
	command = model.startQuery()
	require.NotNil(t, command)
	batch, ok = command().(tea.BatchMsg)
	require.True(t, ok)
	saved, ok = batch[len(batch)-1]().(sqlScriptSavedMsg)
	require.True(t, ok)
	require.NoError(t, saved.err)

	scripts, err = model.sqlScripts.getList(connectionName)
	require.NoError(t, err)
	require.Len(t, scripts, 1)
	assert.Equal(t, "SELECT revised;", scripts[0].content)
}

func TestRawQueryScriptSaveFailureDoesNotBlockExecution(t *testing.T) {
	database := &fakeDatabase{name: "chinook", queryResult: db.QueryResult{CommandTag: "SELECT 1"}}
	model := New(config.Config{Connections: []config.Connection{{Name: "../not-a-directory"}}}, ConnectionSettings{}, nil)
	model.database = database
	model.activeConnectionIndex = 0
	model.panel = panelQuery
	model.query.editor.SetValue("SELECT 1;")

	command := model.startQuery()
	require.NotNil(t, command)
	batch, ok := command().(tea.BatchMsg)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(batch), 2)
	finished, ok := batch[0]().(queryFinishedMsg)
	require.True(t, ok)
	saved, ok := batch[len(batch)-1]().(sqlScriptSavedMsg)
	require.True(t, ok)
	require.Error(t, saved.err)

	updated, _ := updateModel(t, model, saved)
	updated, _ = updateModel(t, updated, finished)
	assert.Equal(t, 1, database.executeCalls)
	assert.Equal(t, "SELECT 1;", database.executedSQL)
	assert.Equal(t, "SELECT 1", updated.query.result.CommandTag)
	assert.Contains(t, updated.query.view(updated.layout, true, true, ""), "SQL script was not saved")
}

func TestConnectionRenameMovesSQLScriptsLibrary(t *testing.T) {
	const (
		oldName = "rename scripts old"
		newName = "rename scripts new"
	)
	model := New(config.Config{Connections: []config.Connection{{Name: oldName}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0
	modal := newActionsModal("", oldName)
	model.actionsModal = &modal
	require.NoError(t, model.sqlScripts.createByConnection(oldName, "SELECT moved;"))

	next, command := model.updateActionsModal(submitRenameMsg{newName: newName})
	updated, ok := next.(Model)
	require.True(t, ok)
	require.NotNil(t, command)
	message, ok := command().(renameRequestMsg)
	require.True(t, ok)
	require.NoError(t, message.err)
	updated, _ = updateModel(t, updated, message)
	assert.Equal(t, newName, updated.config.Connections[0].Name)

	scripts, err := updated.sqlScripts.getList(newName)
	require.NoError(t, err)
	require.Len(t, scripts, 1)
	assert.Equal(t, "SELECT moved;", scripts[0].content)
	_, err = updated.sqlScripts.getList(oldName)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}
