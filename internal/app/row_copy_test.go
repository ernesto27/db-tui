package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/redis"
)

func TestRowClipboardTextPreservesFullValues(t *testing.T) {
	longValue := strings.Repeat("value", 100)
	row := []any{42, nil, []byte("bytes"), "a\tb\n\"quoted\"", "", longValue}
	assert.Equal(t, "42\tNULL\tbytes\ta\tb\n\"quoted\"\t\t"+longValue, rowClipboardText(row))
}

func rowCopyTestModel() Model {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "copy-test", engine: db.EnginePostgreSQL}
	model.focus = focusData
	model.activeRelation = activeRelation{set: true, item: navigatorItem{section: navigatorTables, name: "people"}}
	model.data.page = db.RowPage{Columns: []string{"id", "name"}, Rows: [][]any{{1, "first"}, {2, "second"}}}
	model.data.selected = 1
	return model
}

func TestCopyShortcutUsesSelectedRowInSQLAndRedis(t *testing.T) {
	for _, surface := range []string{"table", "query", "redis"} {
		t.Run(surface, func(t *testing.T) {
			model := rowCopyTestModel()
			want := []any{2, "second"}
			switch surface {
			case "query":
				model.panel = panelQuery
				model.query.finishExecute(db.QueryResult{Columns: model.data.page.Columns, Rows: model.data.page.Rows}, 0, nil)
				model.query.selectedRow = 1
			case "redis":
				model.database = nil
				model.redis.client = &redis.Client{}
				model.activeRelation = activeRelation{}
				want = []any{"key:2", strings.Repeat("value", 50), "string", "1m0s"}
				model.data.page = db.RowPage{Columns: []string{"key", "value", "type", "ttl"}, Rows: [][]any{{"key:1", "first", "string", "—"}, want}}
			}
			model.data.columnOffset = 1
			footer := model.footerText()
			updated, command := updateModel(t, model, keyPress('c', "c", 0))
			require.NotNil(t, command)
			row, ok := updated.selectedRowForCopy()
			require.True(t, ok)
			assert.Equal(t, want, row)
			assert.Contains(t, ansi.Strip(updated.View().Content), rowCopyNoticeText)
			assert.Equal(t, footer, updated.footerText())
		})
	}
}

func TestCopyShortcutRemainsTextInputInEditors(t *testing.T) {
	for _, surface := range []string{"SQL", "Redis"} {
		t.Run(surface, func(t *testing.T) {
			model := rowCopyTestModel()
			model.panel = panelQuery
			if surface == "Redis" {
				model.database = nil
				model.redis.client = &redis.Client{}
			}
			_ = model.query.focusEditor()
			updated, _ := updateModel(t, model, keyPress('c', "c", 0))
			assert.Equal(t, "c", updated.query.editor.Value())
			assert.False(t, updated.rowCopy.visible)
		})
	}
}

func TestCopyToastExpiryPreservesNewerCopies(t *testing.T) {
	model := rowCopyTestModel()
	model, _ = updateModel(t, model, keyPress('c', "c", 0))
	first := rowCopyNoticeExpiredMsg{session: model.session, request: model.rowCopy.request}
	model, _ = updateModel(t, model, keyPress('c', "c", 0))
	second := rowCopyNoticeExpiredMsg{session: model.session, request: model.rowCopy.request}
	model, _ = updateModel(t, model, first)
	assert.True(t, model.rowCopy.visible)
	model, _ = updateModel(t, model, second)
	assert.False(t, model.rowCopy.visible)
	assert.NotContains(t, ansi.Strip(model.View().Content), rowCopyNoticeText)
}
