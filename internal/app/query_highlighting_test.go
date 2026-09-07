package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/ernestoponce27/db-tui/internal/app/sqlhighlight"
	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestKeywordCellRange(t *testing.T) {
	cells := []sqlEditorCell{
		{source: 0, rune: 'S', width: 1},
		{source: 1, rune: 'E', width: 1},
		{source: 2, rune: '界', width: 2},
		{source: 3, rune: 'T', width: 1},
	}

	left, right, found := keywordCellRange(cells, sqlhighlight.Span{Start: 1, End: 4})

	assert.True(t, found)
	assert.Equal(t, 1, left)
	assert.Equal(t, 5, right)

	_, _, found = keywordCellRange(cells, sqlhighlight.Span{Start: 4, End: 5})
	assert.False(t, found)
}

func TestKeywordCellRanges(t *testing.T) {
	cells := sqlEditorRows([]rune("SELECT"), 80)[0]
	spans := []sqlhighlight.Span{{Start: 0, End: 6}, {Start: 9, End: 13}}

	ranges := keywordCellRanges(cells, spans)

	assert.Equal(t, []sqlKeywordCellRange{{left: 0, right: 6}}, ranges)
}

func TestHighlightSQLKeywords(t *testing.T) {
	const input = "SELECT FROM"
	rows := sqlEditorRows([]rune(input), 80)

	highlighted := highlightSQLKeywords(input, rows, input, sqlhighlight.PostgreSQL{})
	keywordStyle := lipgloss.NewStyle().Foreground(colorSQLKeyword)

	assert.Equal(t, input, ansi.Strip(highlighted))
	assert.Contains(t, highlighted, keywordStyle.Render("SELECT"))
	assert.Contains(t, highlighted, keywordStyle.Render("FROM"))
}

func TestBaseViewHighlightsKeywordsForSupportedEngines(t *testing.T) {
	tests := []struct {
		name            string
		database        db.Database
		wantHighlighted bool
	}{
		{
			name:            "PostgreSQL",
			database:        &fakeDatabase{engine: db.EnginePostgreSQL},
			wantHighlighted: true,
		},
		{
			name:            "MySQL",
			database:        &fakeDatabase{engine: db.EngineMySQL},
			wantHighlighted: true,
		},
		{name: "Oracle", database: &fakeDatabase{engine: db.EngineOracle}},
		{name: "disconnected"},
	}

	keywordStyle := lipgloss.NewStyle().Foreground(colorSQLKeyword)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := New(config.Config{}, ConnectionSettings{}, nil)
			model.database = test.database
			model.panel = panelQuery
			model.query.editor.SetValue("SELECT 1")

			rendered := model.baseView().Content

			if test.wantHighlighted {
				assert.Contains(t, rendered, keywordStyle.Render("SELECT"))
				return
			}
			assert.NotContains(t, rendered, keywordStyle.Render("SELECT"))
		})
	}
}

func TestQueryEditorViewSelectionReplacesKeywordStyle(t *testing.T) {
	layout := newAppLayout(100, 24)
	query := newQueryModel(layout)
	query.editor.SetValue("SELECT")
	query.selection = sqlSelection{anchor: 0, head: 5, active: true}

	rendered := query.editorView(sqlhighlight.PostgreSQL{})
	selected, active := query.selection.selectedSQL(query.editor.Value())
	keywordStyle := lipgloss.NewStyle().Foreground(colorSQLKeyword)

	assert.True(t, active)
	assert.Equal(t, "SELECT", selected)
	assert.Equal(t, "SELECT", strings.TrimSpace(ansi.Strip(rendered)))
	assert.NotContains(t, rendered, keywordStyle.Render("SELECT"))
	assert.Contains(t, rendered, textSelectionStyle.Render("SELECT"))
}
