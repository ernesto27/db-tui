package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestTableExportFormatPickerFlow(t *testing.T) {
	for _, engine := range []string{db.EnginePostgreSQL, db.EngineMySQL} {
		t.Run(engine, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			database := &fakeDatabase{name: "chinook", engine: engine}
			model.database = database
			model.navigator.tables = []db.Table{{Schema: "reporting", Name: "Album"}}

			updated, command := updateModel(t, model, keyPress('e', "", tea.ModCtrl))
			assert.Nil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportSelecting, updated.exportModal.state)
			assert.Equal(t, db.Table{Schema: "reporting", Name: "Album"}, updated.exportModal.table)
			assert.Equal(t, "Album", updated.exportModal.tableName)
			assert.Equal(t, db.ExportTypeCSV, updated.exportModal.format)
			pickerView := updated.exportModal.view(80, "⠋")
			assert.Contains(t, pickerView, "Export Album as:")
			assert.Contains(t, pickerView, "› CSV")
			assert.Contains(t, pickerView, "  JSON")

			updated, command = updateModel(t, updated, keyPress(tea.KeyDown, "", 0))
			assert.Nil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, db.ExportTypeJSON, updated.exportModal.format)

			updated, command = updateModel(t, updated, keyPress(tea.KeyEnter, "", 0))
			assert.Nil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportConfirming, updated.exportModal.state)
			assert.Contains(t, updated.exportModal.view(80, "⠋"), "Export Album to JSON?")

			updated, command = updateModel(t, updated, keyPress(tea.KeyEnter, "", 0))
			require.NotNil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportRunning, updated.exportModal.state)
			assert.Contains(t, updated.exportModal.view(80, "⠋"), "Exporting Album to JSON")
			batch, ok := command().(tea.BatchMsg)
			require.True(t, ok)
			require.NotEmpty(t, batch)
			_, ok = batch[0]().(exportFinishedMsg)
			require.True(t, ok)
			assert.Equal(t, db.ExportTypeJSON, database.exportType)
			assert.Equal(t, db.Table{Schema: "reporting", Name: "Album"}, database.exportTable)
			assert.True(t, database.exportDeadline)

			updated, command = updateModel(t, updated, exportFinishedMsg{session: updated.session})
			assert.Nil(t, command)
			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportSucceeded, updated.exportModal.state)
			successView := updated.exportModal.view(80, "⠋")
			assert.Contains(t, successView, "JSON exported successfully")

			updated, command = updateModel(t, updated, keyPress(tea.KeyEscape, "", 0))
			assert.Nil(t, command)
			assert.Nil(t, updated.exportModal)
		})
	}
}

func TestTableExportFormatPickerNavigation(t *testing.T) {
	tests := []struct {
		name       string
		keys       []tea.KeyPressMsg
		wantFormat string
	}{
		{
			name:       "up stays on CSV",
			keys:       []tea.KeyPressMsg{keyPress(tea.KeyUp, "", 0)},
			wantFormat: db.ExportTypeCSV,
		},
		{
			name:       "down selects JSON and clamps",
			keys:       []tea.KeyPressMsg{keyPress(tea.KeyDown, "", 0), keyPress(tea.KeyDown, "", 0)},
			wantFormat: db.ExportTypeJSON,
		},
		{
			name:       "j selects JSON",
			keys:       []tea.KeyPressMsg{keyPress('j', "j", 0)},
			wantFormat: db.ExportTypeJSON,
		},
		{
			name:       "k selects CSV",
			keys:       []tea.KeyPressMsg{keyPress(tea.KeyDown, "", 0), keyPress('k', "k", 0)},
			wantFormat: db.ExportTypeCSV,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = &fakeDatabase{name: "chinook"}
			model.navigator.tables = []db.Table{{Name: "Album"}}

			updated, _ := updateModel(t, model, keyPress('e', "", tea.ModCtrl))
			for _, key := range test.keys {
				updated, _ = updateModel(t, updated, key)
			}

			require.NotNil(t, updated.exportModal)
			assert.Equal(t, exportSelecting, updated.exportModal.state)
			assert.Equal(t, test.wantFormat, updated.exportModal.format)
		})
	}
}

func TestTableExportFormatPickerEscapeCancels(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}

	updated, _ := updateModel(t, model, keyPress('e', "", tea.ModCtrl))
	updated, command := updateModel(t, updated, keyPress(tea.KeyEscape, "", 0))

	assert.Nil(t, command)
	assert.Nil(t, updated.exportModal)
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
	assert.Equal(t, exportConfirming, updated.exportModal.state)
	assert.Equal(t, db.ExportTypeCSV, updated.exportModal.format)
	assert.Equal(t, "SELECT * FROM Album", updated.exportModal.query)
	assert.Contains(t, updated.exportModal.view(80, "⠋"), "Export query results to CSV?")
}
