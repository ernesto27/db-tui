package app

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestNavigatorSelectIndexClampsAndKeepsSelectionVisible(t *testing.T) {
	navigator := newNavigatorModel()
	for index := range 20 {
		navigator.tables = append(navigator.tables, db.Table{
			Name: fmt.Sprintf("table_%02d", index),
		})
	}

	assert.True(t, navigator.selectIndex(12, 5))
	assert.Equal(t, 12, navigator.cursor().selected)
	assert.Equal(t, 8, navigator.cursor().offset)

	assert.True(t, navigator.selectIndex(-10, 5))
	assert.Equal(t, 0, navigator.cursor().selected)
	assert.Equal(t, 0, navigator.cursor().offset)

	assert.True(t, navigator.selectIndex(100, 5))
	assert.Equal(t, 19, navigator.cursor().selected)
	assert.Equal(t, 15, navigator.cursor().offset)

	assert.False(t, navigator.selectIndex(19, 5))
}

func TestNavigatorSelectIndexResetsEmptyList(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.cursors[navigatorTables] = navigatorCursor{selected: 4, offset: 3}

	changed := navigator.selectIndex(10, 5)

	assert.False(t, changed)
	assert.Zero(t, navigator.cursor().selected)
	assert.Zero(t, navigator.cursor().offset)
}

func TestNavigatorVisibleTablesFiltersCaseInsensitiveSubstring(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.tables = []db.Table{
		{Name: "Album"},
		{Name: "Artist"},
		{Name: "Track"},
	}

	assert.Equal(t, navigator.tables, navigator.visibleTables())

	navigator.filter.SetValue(" AR ")
	assert.Equal(t, []db.Table{
		{Name: "Artist"},
	}, navigator.visibleTables())

	navigator.filter.SetValue("missing")
	assert.Empty(t, navigator.visibleTables())
}

func TestNavigatorObjectSectionsRequireEngineCapabilities(t *testing.T) {
	navigator := newNavigatorModel()
	layout := newAppLayout(100, 24)

	assert.False(t, navigator.selectSection(navigatorMaterializedViews, layout.navigatorListRows))
	assert.False(t, navigator.selectSection(navigatorFunctions, layout.navigatorListRows))

	navigator.setMaterializedViewsAvailable(true)
	assert.True(t, navigator.selectSection(navigatorMaterializedViews, layout.navigatorListRows))
	assert.Equal(t, navigatorMaterializedViews, navigator.section)
	assert.Contains(t, navigator.view(navigatorStatus{databaseConneced: true}, layout, true), "Materialized views")

	navigator.setMaterializedViewsAvailable(false)
	assert.Equal(t, navigatorViews, navigator.section)

	navigator.setFunctionsAvailable(true)
	assert.True(t, navigator.selectSection(navigatorFunctions, layout.navigatorListRows))
	assert.Equal(t, navigatorFunctions, navigator.section)
}

func TestNavigatorNormalizeSelectionPreservesVisibleTable(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.tables = []db.Table{
		{Name: "Album"},
		{Name: "Artist"},
		{Name: "Track"},
	}
	navigator.cursors[navigatorTables].selected = 1
	selected, ok := navigator.selectedItem()
	require.True(t, ok)

	navigator.filter.SetValue("art")
	changed := navigator.normalizeSelectionWithPrevious(selected, true, 5)

	assert.False(t, changed)
	assert.Zero(t, navigator.cursor().selected)
	table, ok := navigator.selectedTable()
	require.True(t, ok)
	assert.Equal(t, "Artist", table.Name)
}

func TestNavigatorNormalizeSelectionSelectsFirstWhenCurrentDisappears(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.tables = []db.Table{
		{Name: "Album"},
		{Name: "Artist"},
		{Name: "Track"},
	}
	navigator.cursors[navigatorTables].selected = 1
	selected, ok := navigator.selectedItem()
	require.True(t, ok)

	navigator.filter.SetValue("track")
	changed := navigator.normalizeSelectionWithPrevious(selected, true, 5)

	assert.True(t, changed)
	assert.Zero(t, navigator.cursor().selected)
	table, ok := navigator.selectedTable()
	require.True(t, ok)
	assert.Equal(t, "Track", table.Name)
}

func TestNavigatorNormalizeSelectionResetsWhenNothingMatches(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.tables = []db.Table{{Name: "Album"}}
	navigator.cursors[navigatorTables] = navigatorCursor{offset: 1}
	selected, ok := navigator.selectedItem()
	require.True(t, ok)

	navigator.filter.SetValue("missing")
	changed := navigator.normalizeSelectionWithPrevious(selected, true, 5)

	assert.True(t, changed)
	assert.Zero(t, navigator.cursor().selected)
	assert.Zero(t, navigator.cursor().offset)
}

func TestNavigatorCancelSearchClearsFilterAndPreservesTable(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.tables = []db.Table{
		{Name: "Album"},
		{Name: "Artist"},
		{Name: "Track"},
	}
	navigator.filter.SetValue("art")
	navigator.searching = true
	navigator.cursors[navigatorTables].selected = 0

	changed := navigator.cancelSearch(5)

	assert.False(t, changed)
	assert.Empty(t, navigator.filter.Value())
	assert.False(t, navigator.searching)
	assert.Equal(t, 1, navigator.cursor().selected)
	table, ok := navigator.selectedTable()
	require.True(t, ok)
	assert.Equal(t, "Artist", table.Name)
}

func TestNavigatorTableAtMouse(t *testing.T) {
	layout := newAppLayout(100, 24)
	navigator := newNavigatorModel()
	for index := range 20 {
		navigator.tables = append(navigator.tables, db.Table{
			Name: fmt.Sprintf("table_%02d", index),
		})
	}
	navigator.cursors[navigatorTables].offset = 3

	tests := []struct {
		name      string
		message   tea.MouseClickMsg
		wantIndex int
		wantOK    bool
	}{
		{
			name: "first visible row",
			message: tea.MouseClickMsg{
				X: 1, Y: layout.navigatorListY, Button: tea.MouseLeft,
			},
			wantIndex: 3,
			wantOK:    true,
		},
		{
			name: "last visible row",
			message: tea.MouseClickMsg{
				X:      1,
				Y:      layout.navigatorListY + layout.navigatorListRows - 1,
				Button: tea.MouseLeft,
			},
			wantIndex: 18,
			wantOK:    true,
		},
		{
			name: "above list",
			message: tea.MouseClickMsg{
				X: 1, Y: layout.navigatorListY - 1, Button: tea.MouseLeft,
			},
		},
		{
			name: "below list",
			message: tea.MouseClickMsg{
				X:      1,
				Y:      layout.navigatorListY + layout.navigatorListRows,
				Button: tea.MouseLeft,
			},
		},
		{
			name: "left border",
			message: tea.MouseClickMsg{
				X: 0, Y: layout.navigatorListY, Button: tea.MouseLeft,
			},
		},
		{
			name: "right border",
			message: tea.MouseClickMsg{
				X:      layout.navigator.width - 1,
				Y:      layout.navigatorListY,
				Button: tea.MouseLeft,
			},
		},
		{
			name: "non-left button",
			message: tea.MouseClickMsg{
				X: 1, Y: layout.navigatorListY, Button: tea.MouseRight,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index, ok := navigator.itemAtMouse(test.message, layout)
			assert.Equal(t, test.wantOK, ok)
			if test.wantOK {
				assert.Equal(t, test.wantIndex, index)
			}
		})
	}
}

func TestNavigatorTableAtMouseRejectsIndexPastFilteredList(t *testing.T) {
	layout := newAppLayout(100, 24)
	navigator := newNavigatorModel()
	navigator.tables = []db.Table{
		{Name: "Album"},
		{Name: "Artist"},
	}
	navigator.filter.SetValue("album")

	_, ok := navigator.itemAtMouse(tea.MouseClickMsg{
		X:      1,
		Y:      layout.navigatorListY + 1,
		Button: tea.MouseLeft,
	}, layout)

	assert.False(t, ok)
}
