package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestFunctionsLoadedUpdatesCurrentSessionOnly(t *testing.T) {
	functions := []db.FunctionColumns{{Name: "customer_total"}}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 7
	model.functionsLoading = true
	model.navigator.setFunctionsAvailable(true)

	updated, command := updateModel(t, model, functionsLoadedMsg{functions: functions, session: 7})

	assert.Nil(t, command)
	assert.False(t, updated.functionsLoading)
	assert.Equal(t, functions, updated.navigator.functions)

	stale, command := updateModel(t, updated, functionsLoadedMsg{session: 6})
	assert.Nil(t, command)
	assert.Equal(t, functions, stale.navigator.functions)
}

func TestFunctionSchemaUsesEngineScope(t *testing.T) {
	assert.Equal(t, "public", functionSchema(&fakeDatabase{engine: db.EnginePostgreSQL}))
	assert.Equal(t, "chinook", functionSchema(&fakeDatabase{name: "chinook", engine: db.EngineMySQL}))
	assert.Empty(t, functionSchema(&fakeDatabase{engine: db.EngineOracle}))
}

func TestObjectsModalSelectsFunctionNavigatorSection(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook", engine: db.EngineMySQL}
	model.navigator.setFunctionsAvailable(true)

	opened, _ := updateModel(t, model, keyPress('o', "", tea.ModCtrl))
	moved, _ := updateModel(t, opened, keyPress(tea.KeyDown, "", 0))
	moved, _ = updateModel(t, moved, keyPress(tea.KeyDown, "", 0))
	updated, command := updateModel(t, moved, keyPress(tea.KeyEnter, "", 0))

	assert.NotNil(t, opened.objectsModal)
	assert.Nil(t, command)
	assert.Nil(t, updated.objectsModal)
	assert.Equal(t, navigatorFunctions, updated.navigator.section)
	assert.Equal(t, focusNavigator, updated.focus)
}
