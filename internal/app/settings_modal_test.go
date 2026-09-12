package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestSettingsModalSavesMaxPageSizeWithoutReloadingRows(t *testing.T) {
	database := &fakeDatabase{}
	model := New(config.Config{MaxPageSize: 100, QueryTimeoutSeconds: 1200}, ConnectionSettings{}, nil)
	model.database = database
	model.activeRelation = activeRelation{item: navigatorItem{name: "Album", section: navigatorTables}, set: true}
	model.data = dataModel{page: db.RowPage{Rows: [][]any{{1}, {2}}}, offset: 100, selected: 1}

	opened, command := updateModel(t, model, keyPress('s', "", tea.ModCtrl))
	require.NotNil(t, opened.settingsModal)
	assert.Equal(t, "100", opened.settingsModal.maxPageSize.Value())
	require.NotNil(t, command)

	opened.settingsModal.maxPageSize.SetValue("250")
	opened.settingsModal.queryTimeout.SetValue("600")
	submitted, command := updateModel(t, opened, keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)

	saving, command := updateModel(t, submitted, command())
	require.NotNil(t, saving.settingsModal)
	assert.True(t, saving.settingsModal.saving)
	require.NotNil(t, command)

	saved, command := updateModel(t, saving, command())
	assert.Nil(t, command)
	assert.Nil(t, saved.settingsModal)
	assert.Equal(t, 250, saved.config.MaxPageSize)
	assert.Equal(t, 600, saved.config.QueryTimeoutSeconds)
	assert.Equal(t, 100, saved.data.offset)
	assert.Equal(t, [][]any{{1}, {2}}, saved.data.page.Rows)
	assert.Equal(t, 1, saved.data.selected)
	assert.Zero(t, database.getRowsCalls)
}

func TestSettingsModalRejectsInvalidMaxPageSize(t *testing.T) {
	model := New(config.Config{MaxPageSize: 100}, ConnectionSettings{}, nil)
	modal := newSettingsModal(model.config.MaxPageSize, model.config.QueryTimeoutSeconds)
	modal.maxPageSize.SetValue("0")
	model.settingsModal = &modal

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	assert.Nil(t, command)
	require.NotNil(t, updated.settingsModal)
	assert.Equal(t, "max page size must be a positive whole number", updated.settingsModal.errorText)
	assert.Equal(t, 100, updated.config.MaxPageSize)
}

func TestSettingsModalCancelsWithoutChangingMaxPageSize(t *testing.T) {
	model := New(config.Config{MaxPageSize: 100}, ConnectionSettings{}, nil)
	modal := newSettingsModal(model.config.MaxPageSize, model.config.QueryTimeoutSeconds)
	modal.maxPageSize.SetValue("250")
	model.settingsModal = &modal

	cancelling, command := updateModel(t, model, keyPress(tea.KeyEscape, "", 0))
	require.NotNil(t, command)
	cancelled, command := updateModel(t, cancelling, command())

	assert.Nil(t, command)
	assert.Nil(t, cancelled.settingsModal)
	assert.Equal(t, 100, cancelled.config.MaxPageSize)
}

func TestSettingsModalRejectsInvalidQueryTimeout(t *testing.T) {
	model := New(config.Config{MaxPageSize: 100, QueryTimeoutSeconds: 1200}, ConnectionSettings{}, nil)
	modal := newSettingsModal(model.config.MaxPageSize, model.config.QueryTimeoutSeconds)
	modal.queryTimeout.SetValue("0")
	model.settingsModal = &modal

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	assert.Nil(t, command)
	require.NotNil(t, updated.settingsModal)
	assert.Equal(t, "query timeout must be a positive whole number of seconds", updated.settingsModal.errorText)
}

func TestSettingsModalView(t *testing.T) {
	modal := newSettingsModal(250, 1200)

	view := modal.view(80)

	assert.Contains(t, view, "Settings")
	assert.Contains(t, view, "Max page size")
	assert.Contains(t, view, "Query timeout (sec)")
	assert.Contains(t, view, "Enter save")
}
