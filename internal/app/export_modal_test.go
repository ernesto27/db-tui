package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestCSVExportFlow(t *testing.T) {
	for _, engine := range []string{db.EnginePostgreSQL, db.EngineMySQL} {
		t.Run(engine, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = &fakeDatabase{name: "chinook"}
			model.databaseEngine = engine
			model.navigator.tables = []db.Table{{Name: "Album"}}

			updated, command := updateModel(t, model, keyPress('e', "", tea.ModCtrl))
			assert.Nil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportConfirming, updated.exportModal.state)
			assert.Equal(t, "Album", updated.exportModal.tableName)
			assert.Contains(t, updated.exportModal.view(80, "⠋"), "Export Album to CSV?")

			updated, command = updateModel(t, updated, keyPress(tea.KeyEnter, "", 0))
			require.NotNil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportRunning, updated.exportModal.state)

			updated, command = updateModel(t, updated, exportFinishedMsg{session: updated.session})
			assert.Nil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportSucceeded, updated.exportModal.state)
			successView := updated.exportModal.view(80, "⠋")
			assert.Contains(t, successView, "CSV exported successfully")
			assert.NotContains(t, successView, ".csv")

			updated, command = updateModel(t, updated, keyPress(tea.KeyEscape, "", 0))
			assert.Nil(t, command)
			assert.Nil(t, updated.exportModal)
		})
	}
}

func TestRawQueryExportUsesLastExecutedSQL(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.panel = panelQuery
	model.query.lastExecutedSQL = "SELECT * FROM Album"
	model.query.result = db.QueryResult{Columns: []string{"AlbumId"}, Rows: [][]any{{1}}}
	model.query.editor.SetValue("SELECT * FROM Artist")

	updated, command := updateModel(t, model, keyPress('e', "", tea.ModCtrl))

	assert.Nil(t, command)
	require.NotNil(t, updated.exportModal)
	assert.Equal(t, exportQuerySource, updated.exportModal.source)
	assert.Equal(t, "SELECT * FROM Album", updated.exportModal.query)
	assert.Contains(t, updated.exportModal.view(80, "⠋"), "Export query results to CSV?")
}
