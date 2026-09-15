package app

import (
	"errors"
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
	}, true)

	view := modal.view(newAppLayout(80, 24))

	assert.Contains(t, view, "analytics — Views")
	assert.Contains(t, view, "public — Tables")
	assert.Contains(t, view, extensionPanelTitle)
	assert.NotContains(t, view, "Functions")
}

func TestDatabaseExplorerModalSelectsOneSchemaObjectPair(t *testing.T) {
	groups := []db.SchemaObjectGroup{
		{Schema: "analytics", Type: db.SchemaObjectViews},
		{Schema: "public", Type: db.SchemaObjectTables},
	}
	modal := newDatabaseExplorerModal(groups, false)

	modal.move(1, newAppLayout(80, 24))

	assert.Equal(t, groups[1], modal.selectedGroup())
}

func TestDatabaseExplorerModalScrollsAndTruncatesLabels(t *testing.T) {
	layout := newAppLayout(64, 16)
	groups := make([]db.SchemaObjectGroup, 12)
	for index := range groups {
		groups[index] = db.SchemaObjectGroup{Schema: "schema-" + string(rune('a'+index)), Type: db.SchemaObjectTables}
	}
	modal := newDatabaseExplorerModal(groups, false)

	modal.move(len(groups)-1, layout)
	view := modal.view(layout)

	assert.Equal(t, len(groups)-1, modal.selected)
	assert.Greater(t, modal.offset, 0)
	assert.Contains(t, view, "schema-l")
	assert.NotContains(t, view, "schema-a")

	longLabelModal := newDatabaseExplorerModal([]db.SchemaObjectGroup{{
		Schema: strings.Repeat("long-schema-name-", 8),
		Type:   db.SchemaObjectMaterializedViews,
	}}, false)
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
	assert.True(t, updated.databaseExplorerModal.extensionsAvailable)
	assert.Nil(t, command)
}

func TestDatabaseExplorerModalMovesAndCloses(t *testing.T) {
	groups := []db.SchemaObjectGroup{
		{Schema: "analytics", Type: db.SchemaObjectViews},
		{Schema: "public", Type: db.SchemaObjectTables},
	}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newDatabaseExplorerModal(groups, false)
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
	modal := newDatabaseExplorerModal([]db.SchemaObjectGroup{{Schema: "analytics", Type: db.SchemaObjectTables}}, false)
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

func TestDatabaseExplorerModalLoadsExtensionsAsReadOnlyData(t *testing.T) {
	database := &fakeDatabase{
		engine:     db.EnginePostgreSQL,
		extensions: []db.ExtensionData{{Name: "pg_trgm", Schema: "public", Version: "1.6"}},
	}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = database
	modal := newDatabaseExplorerModal(nil, true)
	model.databaseExplorerModal = &modal

	updated, command := updateModel(t, model, keyPress(tea.KeyEnter, "", 0))

	require.NotNil(t, command)
	assert.Nil(t, updated.databaseExplorerModal)
	assert.True(t, updated.activeExtensions.set)
	assert.False(t, updated.activeRelation.set)
	assert.True(t, updated.data.loading)

	updated, command = updateModel(t, updated, extensionsLoadedMsg{
		extensions: database.extensions,
		session:    updated.session,
		request:    updated.activeExtensions.request,
	})
	assert.Nil(t, command)
	assert.False(t, updated.data.loading)
	assert.Equal(t, db.RowPage{Columns: []string{extensionNameColumn, extensionSchemaColumn, extensionVersionColumn}, Rows: [][]any{{"pg_trgm", "public", "1.6"}}}, updated.data.page)
	assert.Contains(t, updated.View().Content, extensionPanelTitle)
}

func TestExtensionsRefreshAndIgnoreStaleResults(t *testing.T) {
	database := &fakeDatabase{engine: db.EnginePostgreSQL}
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = database
	model.panel = panelData
	model.focus = focusData
	model.activeExtensions = activeExtensions{request: 2, set: true}
	model.data = dataModel{page: extensionRowPage([]db.ExtensionData{{Name: "hstore", Schema: "public", Version: "1.8"}})}

	stale, command := updateModel(t, model, extensionsLoadedMsg{
		extensions: []db.ExtensionData{{Name: "pg_trgm", Schema: "public", Version: "1.6"}},
		session:    model.session,
		request:    1,
	})
	assert.Nil(t, command)
	assert.Equal(t, "hstore", stale.data.page.Rows[0][0])

	updated, command := updateModel(t, stale, keyPress('r', "r", 0))
	require.NotNil(t, command)
	assert.Equal(t, uint64(3), updated.activeExtensions.request)
	assert.True(t, updated.data.loading)
}

func TestExtensionsDisableRelationActions(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{engine: db.EnginePostgreSQL}
	model.panel = panelData
	model.focus = focusData
	model.activeExtensions = activeExtensions{set: true}
	model.navigator.tables = []db.Table{{Name: "Album"}}

	updated, command := updateModel(t, model, keyPress('e', "e", 0))
	assert.Nil(t, command)
	assert.Nil(t, updated.exportModal)

	updated, command = updateModel(t, updated, keyPress('g', "g", tea.ModCtrl))
	assert.Nil(t, command)
	assert.Nil(t, updated.actionsModal)
}

func TestExtensionsShowEmptyAndErrorStates(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{engine: db.EnginePostgreSQL}
	model.activeExtensions = activeExtensions{request: 1, set: true}
	model.data.finishLoad(extensionRowPage(nil), 0, nil, model.layout)

	assert.Contains(t, model.View().Content, noExtensionsText)

	loadErr := errors.New("catalog access denied")
	updated, command := updateModel(t, model, extensionsLoadedMsg{
		session: model.session,
		request: model.activeExtensions.request,
		err:     loadErr,
	})
	assert.Nil(t, command)
	assert.Contains(t, updated.View().Content, extensionLoadErrorText)
	assert.Contains(t, updated.View().Content, loadErr.Error())
}

func TestDatabaseExplorerOmitsExtensionsOutsidePostgreSQL(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{engine: db.EngineSQLServer}

	updated, command := updateModel(t, model, keyPress('o', "", tea.ModCtrl))

	assert.Nil(t, command)
	require.NotNil(t, updated.databaseExplorerModal)
	assert.False(t, updated.databaseExplorerModal.extensionsAvailable)
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
