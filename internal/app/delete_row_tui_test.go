package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteRowShortcutOpensVisibleConfirmation(t *testing.T) {
	model := deleteRowTestModel()

	updated, command := updateModel(t, model, keyPress('d', "d", 0))

	assert.Nil(t, command)
	require.NotNil(t, updated.deleteRowModal)
	assert.Equal(t, db.Table{Name: "Artist"}, updated.deleteRowModal.table)
	assert.Equal(t, map[string]any{
		"ArtistId": 42,
		"Name":     "AC/DC",
	}, updated.deleteRowModal.whereColumns)

	view := ansi.Strip(updated.View().Content)
	assert.Contains(t, view, "Delete row")
	assert.Contains(t, view, "Delete this row?")
	assert.Contains(t, view, "Enter confirm")
	assert.Contains(t, view, "Esc cancel")

	next, closeCommand := updated.Update(keyPress(tea.KeyEscape, "", 0))
	confirming, ok := next.(*Model)
	require.True(t, ok, "Update returned %T instead of *app.Model", next)
	require.NotNil(t, closeCommand)
	assert.IsType(t, deleteRowCancelMsg{}, closeCommand())
	assert.NotNil(t, confirming.deleteRowModal)
}

func TestDeleteRowShortcutRequiresActiveTableRow(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Model)
	}{
		{
			name: "view",
			mutate: func(model *Model) {
				model.activeRelation.item.section = navigatorViews
			},
		},
		{
			name: "navigator focus",
			mutate: func(model *Model) {
				model.focus = focusNavigator
			},
		},
		{
			name: "empty row page",
			mutate: func(model *Model) {
				model.data.page.Rows = nil
			},
		},
		{
			name: "loading rows",
			mutate: func(model *Model) {
				model.data.loading = true
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := deleteRowTestModel()
			test.mutate(&model)

			updated, command := updateModel(t, model, keyPress('d', "d", 0))

			assert.Nil(t, command)
			assert.Nil(t, updated.deleteRowModal)
		})
	}
}

func deleteRowTestModel() Model {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.panel = panelData
	model.focus = focusData
	model.activeRelation = activeRelation{
		item: navigatorItem{name: "Artist", section: navigatorTables},
		set:  true,
	}
	model.data = dataModel{
		page: db.RowPage{
			Columns: []string{"ArtistId", "Name"},
			Rows:    [][]any{{42, "AC/DC"}},
		},
	}
	return model
}
