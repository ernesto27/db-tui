package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestObjectsModalListsSupportedObjectTypes(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.setMaterializedViewsAvailable(true)
	navigator.setFunctionsAvailable(true)

	modal := newObjectsModal(navigator)

	assert.Equal(t, []navigatorSection{
		navigatorTables,
		navigatorViews,
		navigatorMaterializedViews,
		navigatorFunctions,
	}, modal.sections)
	assert.Contains(t, modal.view(80), "Functions")
}

func TestObjectsModalPreservesCurrentSectionAndClampsMovement(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.setFunctionsAvailable(true)
	navigator.section = navigatorFunctions
	modal := newObjectsModal(navigator)

	assert.Equal(t, 2, modal.selected)
	modal.move(10)
	assert.Equal(t, navigatorFunctions, modal.selectedSection())
	modal.move(-10)
	assert.Equal(t, navigatorTables, modal.selectedSection())
}

func TestDatabaseExplorerModalListsAvailableSchemaObjectPairs(t *testing.T) {
	modal := newDatabaseExplorerModal([]db.SchemaObjectGroup{
		{Schema: "analytics", Type: db.SchemaObjectViews},
		{Schema: "public", Type: db.SchemaObjectTables},
	})

	view := modal.view(newAppLayout(80, 24))

	assert.Contains(t, view, "analytics — Views")
	assert.Contains(t, view, "public — Tables")
	assert.NotContains(t, view, "Functions")
}

func TestDatabaseExplorerModalSelectsOneSchemaObjectPair(t *testing.T) {
	groups := []db.SchemaObjectGroup{
		{Schema: "analytics", Type: db.SchemaObjectViews},
		{Schema: "public", Type: db.SchemaObjectTables},
	}
	modal := newDatabaseExplorerModal(groups)

	modal.move(1, newAppLayout(80, 24))

	assert.Equal(t, groups[1], modal.selectedGroup())
}

func TestDatabaseExplorerModalScrollsAndTruncatesLabels(t *testing.T) {
	layout := newAppLayout(64, 16)
	groups := make([]db.SchemaObjectGroup, 12)
	for index := range groups {
		groups[index] = db.SchemaObjectGroup{Schema: "schema-" + string(rune('a'+index)), Type: db.SchemaObjectTables}
	}
	modal := newDatabaseExplorerModal(groups)

	modal.move(len(groups)-1, layout)
	view := modal.view(layout)

	assert.Equal(t, len(groups)-1, modal.selected)
	assert.Greater(t, modal.offset, 0)
	assert.Contains(t, view, "schema-l")
	assert.NotContains(t, view, "schema-a")

	longLabelModal := newDatabaseExplorerModal([]db.SchemaObjectGroup{{
		Schema: strings.Repeat("long-schema-name-", 8),
		Type:   db.SchemaObjectMaterializedViews,
	}})
	assert.Contains(t, longLabelModal.view(layout), "…")
}

func TestSchemaObjectGroupsLoadedUpdatesCurrentSessionOnly(t *testing.T) {
	groups := []db.SchemaObjectGroup{{Schema: "reporting", Type: db.SchemaObjectViews}}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 7
	model.schemaObjectGroupsLoading = true

	updated, command := updateModel(t, model, schemaObjectGroupsLoadedMsg{groups: groups, session: 7})

	assert.Nil(t, command)
	assert.False(t, updated.schemaObjectGroupsLoading)
	assert.Equal(t, groups, updated.schemaObjectGroups)

	stale, command := updateModel(t, updated, schemaObjectGroupsLoadedMsg{session: 6})
	assert.Nil(t, command)
	assert.Equal(t, groups, stale.schemaObjectGroups)
}

func TestObjectsShortcutOpensDatabaseExplorerModalForPostgreSQL(t *testing.T) {
	groups := []db.SchemaObjectGroup{{Schema: "reporting", Type: db.SchemaObjectViews}}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook", engine: db.EnginePostgreSQL}
	model.schemaObjectGroups = groups

	updated, command := updateModel(t, model, keyPress('o', "", tea.ModCtrl))

	require.NotNil(t, updated.databaseExplorerModal)
	assert.Nil(t, updated.objectsModal)
	assert.Equal(t, groups, updated.databaseExplorerModal.groups)
	assert.Nil(t, command)
}

func TestDatabaseExplorerModalMovesAndCloses(t *testing.T) {
	groups := []db.SchemaObjectGroup{
		{Schema: "analytics", Type: db.SchemaObjectViews},
		{Schema: "public", Type: db.SchemaObjectTables},
	}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newDatabaseExplorerModal(groups)
	model.databaseExplorerModal = &modal

	moved, command := updateModel(t, model, keyPress(tea.KeyDown, "", 0))

	assert.Nil(t, command)
	require.NotNil(t, moved.databaseExplorerModal)
	assert.Equal(t, groups[1], moved.databaseExplorerModal.selectedGroup())

	closed, command := updateModel(t, moved, keyPress(tea.KeyEsc, "", 0))

	assert.Nil(t, command)
	assert.Nil(t, closed.databaseExplorerModal)
}

func TestDatabaseExplorerModalLoadsSelectedSchemaTables(t *testing.T) {
	database := &fakeDatabase{engine: db.EnginePostgreSQL}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = database
	modal := newDatabaseExplorerModal([]db.SchemaObjectGroup{{Schema: "analytics", Type: db.SchemaObjectTables}})
	model.databaseExplorerModal = &modal

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	require.NotNil(t, command)
	assert.Nil(t, updated.databaseExplorerModal)
	assert.Equal(t, "analytics", updated.navigator.schema)
	assert.Equal(t, navigatorTables, updated.navigator.section)

	message, ok := command().(tablesLoadedMsg)
	require.True(t, ok)
	assert.Equal(t, "analytics", database.listTablesSchema)
	assert.Equal(t, updated.session, message.session)
}

func TestSchemaObjectTableLoadIgnoresStaleSchema(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.session = 7
	model.loading = true
	model.navigator.schema = "reporting"
	model.navigator.tables = []db.Table{{Schema: "reporting", Name: "monthly_sales"}}

	updated, command := updateModel(t, model, tablesLoadedMsg{
		schema:  "analytics",
		tables:  []db.Table{{Schema: "analytics", Name: "events"}},
		session: 7,
	})

	assert.Nil(t, command)
	assert.True(t, updated.loading)
	assert.Equal(t, []db.Table{{Schema: "reporting", Name: "monthly_sales"}}, updated.navigator.tables)
}

func TestSchemaObjectLoadsIgnoreStaleSchema(t *testing.T) {
	t.Run("views", func(t *testing.T) {
		model := New(config.Config{}, ConnectionSettings{}, nil)
		model.session = 7
		model.viewsLoading = true
		model.navigator.schema = "reporting"
		model.navigator.views = []db.View{{Name: "monthly_sales"}}

		updated, _ := updateModel(t, model, viewsLoadedMsg{
			schema: "analytics", views: []db.View{{Name: "events"}}, session: 7,
		})

		assert.True(t, updated.viewsLoading)
		assert.Equal(t, []db.View{{Name: "monthly_sales"}}, updated.navigator.views)
	})

	t.Run("materialized views", func(t *testing.T) {
		model := New(config.Config{}, ConnectionSettings{}, nil)
		model.session = 7
		model.materializedViewsLoading = true
		model.navigator.schema = "reporting"
		model.navigator.materializedViews = []db.MaterializedView{{Name: "monthly_sales"}}

		updated, _ := updateModel(t, model, materializedViewsLoadedMsg{
			schema: "analytics", materializedViews: []db.MaterializedView{{Name: "events"}}, session: 7,
		})

		assert.True(t, updated.materializedViewsLoading)
		assert.Equal(t, []db.MaterializedView{{Name: "monthly_sales"}}, updated.navigator.materializedViews)
	})

	t.Run("functions", func(t *testing.T) {
		model := New(config.Config{}, ConnectionSettings{}, nil)
		model.session = 7
		model.functionsLoading = true
		model.navigator.schema = "reporting"
		model.navigator.functions = []db.FunctionColumns{{Name: "monthly_sales"}}

		updated, _ := updateModel(t, model, functionsLoadedMsg{
			schema: "analytics", functions: []db.FunctionColumns{{Name: "events"}}, session: 7,
		})

		assert.True(t, updated.functionsLoading)
		assert.Equal(t, []db.FunctionColumns{{Name: "monthly_sales"}}, updated.navigator.functions)
	})
}
