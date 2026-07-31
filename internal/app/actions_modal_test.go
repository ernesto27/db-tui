package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestActionsModalShowsDDLAndRenameActions(t *testing.T) {
	modal := newActionsModal("Album", "Local")
	view := modal.view(80)

	assert.Contains(t, view, "View DDL for Album")
	assert.Contains(t, view, "Inspect columns for Album")
	assert.Contains(t, view, "Rename connection \"Local\"")
}

func TestActionsModalRenameOnlyWhenNoTableSelected(t *testing.T) {
	modal := newActionsModal("", "Local")
	view := modal.view(80)

	assert.NotContains(t, view, "View DDL")
	assert.NotContains(t, view, "Inspect columns")
	assert.Contains(t, view, "Rename connection \"Local\"")
}

func TestActionsModalKeyboardNavigation(t *testing.T) {
	modal := newActionsModal("Album", "Local")

	// Down moves to second action.
	updated, _ := modal.update(keyPress(tea.KeyDown, "", 0))
	assert.Equal(t, 1, updated.selected)

	// Up moves back to first.
	updated, _ = updated.update(keyPress(tea.KeyUp, "", 0))
	assert.Equal(t, 0, updated.selected)

	// j moves down.
	updated, _ = updated.update(keyPress('j', "j", 0))
	assert.Equal(t, 1, updated.selected)

	// k moves up.
	updated, _ = updated.update(keyPress('k', "k", 0))
	assert.Equal(t, 0, updated.selected)
}

func TestActionsModalKeyboardNavigationRenameOnly(t *testing.T) {
	modal := newActionsModal("", "Local")

	// Only one action, selection stays at 0.
	updated, _ := modal.update(keyPress(tea.KeyDown, "", 0))
	assert.Equal(t, 0, updated.selected)

	updated, _ = updated.update(keyPress(tea.KeyUp, "", 0))
	assert.Equal(t, 0, updated.selected)
}

func TestActionsModalEnterSelectsAction(t *testing.T) {
	modal := newActionsModal("Album", "Local")

	_, command := modal.update(keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)

	msg := command()
	_, ok := msg.(selectDDLActionMsg)
	assert.True(t, ok)
}

func TestActionsModalEnterSelectsRenameWhenDDLNotAvailable(t *testing.T) {
	modal := newActionsModal("", "Local")

	_, command := modal.update(keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)

	msg := command()
	_, ok := msg.(selectRenameActionMsg)
	assert.True(t, ok)
}

func TestActionsModalEnterSelectsColumnsAction(t *testing.T) {
	modal := newActionsModal("Album", "Local")
	modal.selected = 1

	_, command := modal.update(keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)

	msg := command()
	_, ok := msg.(selectColumnsActionMsg)
	assert.True(t, ok)
}

func TestActionsModalEnterSelectsRenameAfterColumnsAction(t *testing.T) {
	modal := newActionsModal("Album", "Local")
	modal.selected = 2

	_, command := modal.update(keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)

	_, ok := command().(selectRenameActionMsg)
	assert.True(t, ok)
}

func TestActionsModalEscCloses(t *testing.T) {
	modal := newActionsModal("Album", "Local")

	_, command := modal.update(keyPress(tea.KeyEscape, "", 0))
	require.NotNil(t, command)

	msg := command()
	_, ok := msg.(cancelActionsMsg)
	assert.True(t, ok)
}

func TestActionsModalEscFromRenameReturnsToSelection(t *testing.T) {
	modal := newActionsModal("Album", "Local")
	modal.state = actionsRenameEditing

	updated, _ := modal.update(keyPress(tea.KeyEscape, "", 0))
	assert.Equal(t, actionsSelecting, updated.state)
}

func TestCtrlGOpensActionsModalWithSelectedTable(t *testing.T) {
	model := New(config.Config{
		Connections: []config.Connection{
			{Name: "MyDB", Engine: "postgres"},
		},
	}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}
	model.activeConnectionIndex = 0

	updated, _ := updateModel(t, model, keyPress('g', "", tea.ModCtrl))

	assert.NotNil(t, updated.actionsModal)
	assert.NotEmpty(t, updated.actionsModal.tableName)
}

func TestCtrlGOpensActionsModalWithoutTable(t *testing.T) {
	model := New(config.Config{
		Connections: []config.Connection{
			{Name: "MyDB", Engine: "postgres"},
		},
	}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0

	updated, _ := updateModel(t, model, keyPress('g', "", tea.ModCtrl))

	assert.NotNil(t, updated.actionsModal)
	assert.Empty(t, updated.actionsModal.tableName)
}

func TestCtrlGNoOpWithoutActiveConnectionAndNoTable(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.activeConnectionIndex = -1

	updated, _ := updateModel(t, model, keyPress('g', "", tea.ModCtrl))

	assert.Nil(t, updated.actionsModal)
}

func TestCtrlGDDLOnlyWithoutActiveConnection(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}
	model.activeConnectionIndex = -1

	updated, _ := updateModel(t, model, keyPress('g', "", tea.ModCtrl))

	// DDL is available even without active connection.
	assert.NotNil(t, updated.actionsModal)
	assert.True(t, updated.actionsModal.ddlAvailable)
	assert.True(t, updated.actionsModal.columnsAvailable)
	assert.False(t, updated.actionsModal.renameAvailable)
}

func TestSelectDDLClosesActionsAndOpensDDLModal(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook", ddl: "CREATE TABLE"}
	model.navigator.tables = []db.Table{{Name: "Album"}}
	model.session = 5

	modal := newActionsModal("Album", "Local")
	model.actionsModal = &modal

	updated, command := updateModel(t, model, selectDDLActionMsg{})
	require.NotNil(t, command)

	assert.Nil(t, updated.actionsModal)
	assert.NotNil(t, updated.ddlModal)
	assert.Equal(t, "Album", updated.ddlModal.tableName)
	assert.True(t, updated.ddlModal.loading)
}

func TestSelectColumnsClosesActionsAndOpensColumnsModal(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook", columns: []db.Column{{Name: "AlbumId"}}}
	model.navigator.tables = []db.Table{{Name: "Album"}}
	model.session = 5

	modal := newActionsModal("Album", "Local")
	model.actionsModal = &modal

	updated, command := updateModel(t, model, selectColumnsActionMsg{})
	require.NotNil(t, command)

	assert.Nil(t, updated.actionsModal)
	require.NotNil(t, updated.columnsModal)
	assert.Equal(t, "Album", updated.columnsModal.tableName)
	assert.True(t, updated.columnsModal.loading)
}

func TestActionsModalRendersOverlay(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)

	// Without actions modal, overlay should show the base view.
	view := model.renderModalOverlay("base")
	assert.Equal(t, "base", view)

	// With actions modal, overlay should show the modal.
	modal := newActionsModal("Album", "Local")
	model.actionsModal = &modal
	view = model.renderModalOverlay("base")
	assert.Contains(t, view, "View DDL for Album")
}

func TestActionsModalClosesWhenEscReceivedInRootUpdate(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newActionsModal("Album", "Local")
	model.actionsModal = &modal

	updated, command := updateModel(t, model, keyPress(tea.KeyEscape, "", 0))

	// Esc in the actions modal emits cancelActionsMsg which closes it.
	assert.NotNil(t, command)
	msg := command()
	_, ok := msg.(cancelActionsMsg)
	assert.True(t, ok)
	assert.NotNil(t, updated.actionsModal)
}

func TestRenameInputPrefilledWithCurrentName(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "MyDatabase", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.activeConnectionIndex = 0

	modal := newActionsModal("", "MyDatabase")
	model.actionsModal = &modal

	model, _ = updateModel(t, model, selectRenameActionMsg{})

	assert.NotNil(t, model.actionsModal)
	assert.Equal(t, actionsRenameEditing, model.actionsModal.state)
	assert.Equal(t, "MyDatabase", model.actionsModal.renameInput.Value())
}

func TestRenameEmptyNameValidation(t *testing.T) {
	cfg := config.Config{
		Connections: []config.Connection{
			{Name: "MyDB", Engine: "postgres"},
		},
	}
	model := New(cfg, ConnectionSettings{}, nil)
	model.activeConnectionIndex = 0

	modal := newActionsModal("", "MyDB")
	model.actionsModal = &modal
	model, _ = updateModel(t, model, selectRenameActionMsg{})

	// Set whitespace-only value and try to submit.
	model.actionsModal.renameInput.SetValue("   ")
	updatedModal, cmd := model.actionsModal.submitRename()

	// Should return nil command (validation error) and the returned modal has the error.
	assert.Nil(t, cmd)
	assert.NotEmpty(t, updatedModal.renameError)
}

func TestRenameCancellationReturnsToSelection(t *testing.T) {
	modal := newActionsModal("", "MyDB")
	modal.state = actionsRenameEditing
	modal.renameInput.SetValue("NewName")

	updated, _ := modal.update(keyPress(tea.KeyEscape, "", 0))

	assert.Equal(t, actionsSelecting, updated.state)
	assert.Empty(t, updated.renameError)
}

func TestRenameSuccessState(t *testing.T) {
	modal := newActionsModal("", "MyDB")
	modal.state = actionsRenameSuccess

	view := modal.view(80)
	assert.Contains(t, view, "✓ Connection renamed")
}

func TestRenameFailedState(t *testing.T) {
	modal := newActionsModal("", "MyDB")
	modal.state = actionsRenameFailed
	modal.renameError = "disk full"

	view := modal.view(80)
	assert.Contains(t, view, "disk full")
}

func TestRenameSuccessDismissesWithEnterOrEsc(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{
		keyPress(tea.KeyEnter, "", 0),
		keyPress(tea.KeyEscape, "", 0),
	} {
		modal := newActionsModal("", "MyDB")
		modal.state = actionsRenameSuccess

		updated, _ := modal.update(key)
		assert.Equal(t, actionsSelecting, updated.state)
	}
}

func TestRenameFailedDismissesWithEnterOrEsc(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{
		keyPress(tea.KeyEnter, "", 0),
		keyPress(tea.KeyEscape, "", 0),
	} {
		modal := newActionsModal("", "MyDB")
		modal.state = actionsRenameFailed
		modal.renameError = "save failed"

		updated, _ := modal.update(key)
		assert.Equal(t, actionsSelecting, updated.state)
	}
}
