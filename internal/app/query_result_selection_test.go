package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestQueryResultSelectionNavigation(t *testing.T) {
	model := rowCopyTestModel()
	model.panel = panelQuery
	rows := make([][]any, 40)
	for i := range rows {
		rows[i] = []any{i}
	}
	model.query.finishExecute(db.QueryResult{Columns: []string{"id"}, Rows: rows}, 0, nil)
	assert.Zero(t, model.query.selectedRow)

	model, _ = updateModel(t, model, keyPress(tea.KeyDown, "", 0))
	assert.Equal(t, 1, model.query.selectedRow)
	pageSize := model.query.resultPageSize(model.layout)
	model, _ = updateModel(t, model, keyPress(tea.KeyPgDown, "", 0))
	assert.Equal(t, 1+pageSize, model.query.selectedRow)

	model.query.moveResultSelection(100, model.layout)
	assert.Equal(t, 39, model.query.selectedRow)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 64, Height: 16})
	assert.GreaterOrEqual(t, model.query.selectedRow, model.query.viewport)
	assert.Less(t, model.query.selectedRow, model.query.visibleResultEnd(model.layout))

	model.query.moveResultSelection(-100, model.layout)
	assert.Zero(t, model.query.selectedRow)
	model.query.beginExecute("SELECT id")
	assert.Equal(t, -1, model.query.selectedRow)
}
