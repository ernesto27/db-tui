package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestNavigatorNavigationDoesNotLoadRows(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Model)
		msg         tea.Msg
		wantCommand bool
	}{
		{name: "up", setup: func(model *Model) { model.navigator.selectIndex(1, model.layout.navigatorListRows) }, msg: keyPress(tea.KeyUp, "", 0)},
		{name: "down", msg: keyPress(tea.KeyDown, "", 0)},
		{name: "page up", setup: func(model *Model) { model.navigator.selectIndex(18, model.layout.navigatorListRows) }, msg: keyPress(tea.KeyPgUp, "", 0)},
		{name: "page down", msg: keyPress(tea.KeyPgDown, "", 0)},
		{name: "home", setup: func(model *Model) { model.navigator.selectIndex(1, model.layout.navigatorListRows) }, msg: keyPress(tea.KeyHome, "", 0)},
		{name: "end", msg: keyPress(tea.KeyEnd, "", 0)},
		{name: "switch section", msg: keyPress(tea.KeyRight, "", 0)},
		{name: "mouse click", msg: tea.MouseClickMsg{X: 1, Y: newAppLayout(defaultWidth, defaultHeight).navigatorListY + 1, Button: tea.MouseLeft}},
		{name: "mouse wheel", msg: tea.MouseWheelMsg{X: 1, Button: tea.MouseWheelDown}},
		{name: "filter edit", setup: func(model *Model) { model.navigator.searching = true; model.navigator.filter.Focus() }, msg: keyPress('a', "a", 0), wantCommand: true},
		{name: "filter cancel", setup: func(model *Model) {
			model.navigator.filter.SetValue("artist")
			model.navigator.searching = true
			model.navigator.filter.Focus()
		}, msg: keyPress(tea.KeyEscape, "", 0)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = &fakeDatabase{name: "chinook"}
			for index := range 20 {
				model.navigator.tables = append(model.navigator.tables, db.Table{Name: string(rune('A' + index))})
			}
			model.navigator.views = []db.View{{Name: "AlbumView"}}
			if test.setup != nil {
				test.setup(&model)
			}

			updated, command := updateModel(t, model, test.msg)

			if test.wantCommand {
				assert.NotNil(t, command)
			} else {
				assert.Nil(t, command)
			}
			assert.False(t, updated.activeRelation.set)
			assert.False(t, updated.data.loading)
		})
	}
}

func TestEnterActivatesEachRelationType(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Model)
		expected navigatorItem
	}{
		{
			name: "table",
			setup: func(model *Model) {
				model.navigator.tables = []db.Table{{Name: "Album"}}
			},
			expected: navigatorItem{name: "Album", section: navigatorTables},
		},
		{
			name: "view",
			setup: func(model *Model) {
				model.navigator.section = navigatorViews
				model.navigator.views = []db.View{{Name: "AlbumView"}}
			},
			expected: navigatorItem{name: "AlbumView", section: navigatorViews},
		},
		{
			name: "materialized view",
			setup: func(model *Model) {
				model.navigator.section = navigatorMaterializedViews
				model.navigator.setMaterializedViewsAvailable(true)
				model.navigator.materializedViews = []db.MaterializedView{{Name: "SalesByCountry"}}
			},
			expected: navigatorItem{name: "SalesByCountry", section: navigatorMaterializedViews},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = &fakeDatabase{name: "chinook"}
			test.setup(&model)

			updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

			require.NotNil(t, command)
			assert.Equal(t, test.expected, updated.activeRelation.item)
			assert.True(t, updated.activeRelation.set)
			assert.True(t, updated.data.loading)
		})
	}
}

func TestEnterActivatesHighlightedRelation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}, {Name: "Artist"}}
	model.navigator.selectIndex(1, model.layout.navigatorListRows)

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	require.NotNil(t, command)
	assert.Equal(t, navigatorItem{name: "Artist", section: navigatorTables}, updated.activeRelation.item)
	assert.True(t, updated.activeRelation.set)
	assert.Equal(t, uint64(1), updated.activeRelation.request)
	assert.True(t, updated.data.loading)
	assert.Empty(t, updated.data.page.Rows)
}

func TestDoubleClickActivatesHighlightedRelation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}
	click := tea.MouseClickMsg{X: 1, Y: model.layout.navigatorListY, Button: tea.MouseLeft}

	updated, command := updateModel(t, model, click)

	assert.Nil(t, command)
	assert.False(t, updated.activeRelation.set)

	updated, command = updateModel(t, updated, click)

	require.NotNil(t, command)
	assert.Equal(t, navigatorItem{name: "Album", section: navigatorTables}, updated.activeRelation.item)
	assert.True(t, updated.activeRelation.set)
	assert.True(t, updated.data.loading)
}

func TestDoubleClickInQueryPanelDoesNotActivateNavigatorRelation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.panel = panelQuery
	model.navigator.tables = []db.Table{{Name: "Album"}}
	click := tea.MouseClickMsg{X: 1, Y: model.layout.navigatorListY, Button: tea.MouseLeft}

	updated, command := updateModel(t, model, click)
	assert.Nil(t, command)

	updated, command = updateModel(t, updated, click)

	assert.Nil(t, command)
	assert.False(t, updated.activeRelation.set)
	assert.False(t, updated.data.loading)
}

func TestActiveRelationPersistsWhileNavigating(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}, {Name: "Artist"}}
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Album", section: navigatorTables},
		request: 3,
		set:     true,
	}
	model.data = dataModel{page: db.RowPage{Columns: []string{"id"}, Rows: [][]any{{1}}}}

	updated, command := updateModel(t, model, keyPress(tea.KeyDown, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, "Artist", updated.navigator.selectedName())
	assert.Equal(t, navigatorItem{name: "Album", section: navigatorTables}, updated.activeRelation.item)
	assert.Equal(t, [][]any{{1}}, updated.data.page.Rows)
	assert.Equal(t, "Album", updated.dataStatus().tableName)
}

func TestEditRowUsesActiveRelationWhileAnotherTableIsHighlighted(t *testing.T) {
	database := &fakeDatabase{name: "chinook"}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = database
	model.focus = focusData
	model.navigator.tables = []db.Table{{Name: "Album"}, {Name: "Artist"}}
	model.navigator.selectIndex(1, model.layout.navigatorListRows)
	model.activeRelation = activeRelation{item: navigatorItem{name: "Album", section: navigatorTables}, set: true}
	model.data = dataModel{page: db.RowPage{Rows: [][]any{{1}}}}

	_, command := updateModel(t, model, keyPress('e', "e", 0))

	require.NotNil(t, command)
	message := command()
	assert.Equal(t, db.Table{Name: "Album"}, database.listColumnsTable)
	assert.Equal(t, editRowColumnsLoadedMsg{
		table:   db.Table{Name: "Album"},
		row:     []any{1},
		session: model.session,
	}, message)
}

func TestEnterOnActiveRelationDoesNothing(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Album", section: navigatorTables},
		request: 4,
		set:     true,
	}
	model.data = dataModel{page: db.RowPage{Rows: [][]any{{1}}}, offset: 100}

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, uint64(4), updated.activeRelation.request)
	assert.Equal(t, 100, updated.data.offset)
	assert.Equal(t, [][]any{{1}}, updated.data.page.Rows)
}

func TestEnterInDataPanelDoesNotActivateRelation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.focus = focusData
	model.navigator.tables = []db.Table{{Name: "Album"}, {Name: "Artist"}}
	model.navigator.selectIndex(1, model.layout.navigatorListRows)
	model.activeRelation = activeRelation{item: navigatorItem{name: "Album", section: navigatorTables}, set: true}

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, navigatorItem{name: "Album", section: navigatorTables}, updated.activeRelation.item)
}

func TestEnterInQueryPanelDoesNotActivateNavigatorRelation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.panel = panelQuery
	model.focus = focusNavigator
	model.navigator.tables = []db.Table{{Name: "Album"}}
	model.query.editor.SetValue("SELECT 1")
	_ = model.query.focusEditor()

	updated, _ := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	assert.False(t, updated.activeRelation.set)
	assert.False(t, updated.data.loading)
	assert.Equal(t, "SELECT 1\n", updated.query.editor.Value())
}

func TestSearchEnterActivatesHighlightedRelation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}, {Name: "Artist"}}
	model.navigator.filter.SetValue("artist")
	model.navigator.searching = true

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	require.NotNil(t, command)
	assert.False(t, updated.navigator.searching)
	assert.Equal(t, navigatorItem{name: "Artist", section: navigatorTables}, updated.activeRelation.item)
}

func TestRowResultRequiresCurrentRequest(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Album", section: navigatorTables},
		request: 3,
		set:     true,
	}
	model.data = dataModel{loading: true}

	stale, command := updateModel(t, model, rowsLoadedMsg{
		relation: navigatorItem{name: "Album", section: navigatorTables},
		request:  2,
		session:  8,
		page:     db.RowPage{Rows: [][]any{{"stale"}}},
	})

	assert.Nil(t, command)
	assert.True(t, stale.data.loading)
	assert.Empty(t, stale.data.page.Rows)

	updated, command := updateModel(t, stale, rowsLoadedMsg{
		relation: navigatorItem{name: "Album", section: navigatorTables},
		request:  3,
		session:  8,
		page:     db.RowPage{Columns: []string{"value"}, Rows: [][]any{{"current"}}},
	})

	assert.Nil(t, command)
	assert.False(t, updated.data.loading)
	assert.Equal(t, [][]any{{"current"}}, updated.data.page.Rows)
}

func TestHighlightMovementDoesNotInvalidateActiveRowRequest(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Album"}, {Name: "Artist"}}
	model.activeRelation = activeRelation{
		item:    navigatorItem{name: "Album", section: navigatorTables},
		request: 2,
		set:     true,
	}
	model.data = dataModel{loading: true}

	moved, command := updateModel(t, model, keyPress(tea.KeyDown, "", 0))

	assert.Nil(t, command)
	assert.Equal(t, "Artist", moved.navigator.selectedName())

	updated, command := updateModel(t, moved, rowsLoadedMsg{
		relation: navigatorItem{name: "Album", section: navigatorTables},
		request:  2,
		session:  8,
		page:     db.RowPage{Columns: []string{"value"}, Rows: [][]any{{"current"}}},
	})

	assert.Nil(t, command)
	assert.False(t, updated.data.loading)
	assert.Equal(t, [][]any{{"current"}}, updated.data.page.Rows)
}

func TestOnlyNewestActivationResultIsApplied(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.session = 8
	model.navigator.tables = []db.Table{{Name: "Album"}, {Name: "Artist"}}

	model, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)
	model, command = updateModel(t, model, keyPress(tea.KeyDown, "", 0))
	assert.Nil(t, command)
	model, command = updateModel(t, model, keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)
	model, command = updateModel(t, model, keyPress(tea.KeyUp, "", 0))
	assert.Nil(t, command)
	model, command = updateModel(t, model, keyPress(tea.KeyEnter, "", 0))
	require.NotNil(t, command)
	assert.Equal(t, uint64(3), model.activeRelation.request)

	stale, command := updateModel(t, model, rowsLoadedMsg{
		relation: navigatorItem{name: "Album", section: navigatorTables},
		request:  1,
		session:  8,
		page:     db.RowPage{Columns: []string{"value"}, Rows: [][]any{{"stale album"}}},
	})
	assert.Nil(t, command)
	assert.True(t, stale.data.loading)

	stale, command = updateModel(t, stale, rowsLoadedMsg{
		relation: navigatorItem{name: "Artist", section: navigatorTables},
		request:  2,
		session:  8,
		page:     db.RowPage{Columns: []string{"value"}, Rows: [][]any{{"stale artist"}}},
	})
	assert.Nil(t, command)
	assert.True(t, stale.data.loading)

	updated, command := updateModel(t, stale, rowsLoadedMsg{
		relation: navigatorItem{name: "Album", section: navigatorTables},
		request:  3,
		session:  8,
		page:     db.RowPage{Columns: []string{"value"}, Rows: [][]any{{"current album"}}},
	})
	assert.Nil(t, command)
	assert.False(t, updated.data.loading)
	assert.Equal(t, [][]any{{"current album"}}, updated.data.page.Rows)
}

func TestDataPanelPromptsForExplicitActivation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}

	view := model.baseView()

	assert.Contains(t, view.Content, "No relation active.")
	assert.Contains(t, view.Content, "Press Enter to load highlighted relation: Album")
	assert.Contains(t, model.footerText(), "Enter load rows")
}
