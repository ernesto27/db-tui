package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestTableCompletionPrefix(t *testing.T) {
	tests := []struct {
		name  string
		sql   string
		want  string
		valid bool
	}{
		{name: "from", sql: "SELECT * FROM al", want: "al", valid: true},
		{name: "join", sql: "SELECT * FROM Artist JOIN al", want: "al", valid: true},
		{name: "delete from", sql: "DELETE FROM al", want: "al", valid: true},
		{name: "quoted identifier", sql: "SELECT * FROM \"al", valid: false},
		{name: "string", sql: "SELECT 'FROM al", valid: false},
		{name: "line comment", sql: "SELECT * -- FROM al", valid: false},
		{name: "block comment", sql: "SELECT /* FROM al", valid: false},
		{name: "non relation", sql: "SELECT al", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := newQueryModel(newAppLayout(100, 24))
			query.editor.SetValue(test.sql)
			_, _, got, ok := tableCompletionPrefix(query.editor)
			assert.Equal(t, test.valid, ok)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestMatchingTables(t *testing.T) {
	tests := []struct {
		name   string
		tables []db.Table
		prefix string
		want   []db.Table
	}{
		{
			name:   "matches current schema tables",
			tables: []db.Table{{Schema: "public", Name: "Album"}, {Schema: "public", Name: "AlbumTrack"}, {Schema: "reporting", Name: "AnnualReport"}},
			prefix: "al",
			want:   []db.Table{{Schema: "public", Name: "Album"}, {Schema: "public", Name: "AlbumTrack"}},
		},
		{
			name:   "matches case insensitively",
			tables: []db.Table{{Name: "Album"}, {Name: "Artist"}},
			prefix: "ALB",
			want:   []db.Table{{Name: "Album"}},
		},
		{
			name:   "returns no matches",
			tables: []db.Table{{Name: "Album"}},
			prefix: "track",
			want:   []db.Table{},
		},
		{
			name:   "limits results",
			tables: []db.Table{{Name: "AlbumA"}, {Name: "AlbumB"}, {Name: "AlbumC"}, {Name: "AlbumD"}, {Name: "AlbumE"}, {Name: "AlbumF"}},
			prefix: "al",
			want:   []db.Table{{Name: "AlbumA"}, {Name: "AlbumB"}, {Name: "AlbumC"}, {Name: "AlbumD"}, {Name: "AlbumE"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, matchingTables(test.tables, test.prefix))
		})
	}
}

func TestTableCompletionAcceptsSelection(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		cursorAt string
		key      rune
		want     string
	}{
		{
			name:     "enter replaces active token",
			sql:      "SELECT * FROM al WHERE id = 1",
			cursorAt: "SELECT * FROM al",
			key:      tea.KeyEnter,
			want:     "SELECT * FROM \"Album\" WHERE id = 1",
		},
		{
			name:     "tab replaces active token",
			sql:      "SELECT * FROM al",
			cursorAt: "SELECT * FROM al",
			key:      tea.KeyTab,
			want:     "SELECT * FROM \"Album\"",
		},
		{
			name:     "replaces whole token when cursor is in the middle",
			sql:      "SELECT * FROM alxyz",
			cursorAt: "SELECT * FROM al",
			key:      tea.KeyEnter,
			want:     "SELECT * FROM \"Album\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := newQueryModel(newAppLayout(100, 24))
			query.editor.SetValue(test.sql)
			query.restoreEditorCursorOffset(len([]rune(test.cursorAt)))
			query.refreshTableCompletion([]db.Table{{Name: "Album"}, {Name: "Artist"}})
			require.True(t, query.completion.visible)

			assert.True(t, query.handleTableCompletionKey(keyPress(test.key, "", 0)))
			assert.Equal(t, test.want, query.editor.Value())
			assert.False(t, query.completion.visible)
		})
	}
}

func TestTableCompletionNavigatesAndDismisses(t *testing.T) {
	query := newQueryModel(newAppLayout(100, 24))
	query.editor.SetValue("SELECT * FROM al")
	query.refreshTableCompletion([]db.Table{{Name: "Album"}, {Name: "AlbumTrack"}})
	require.True(t, query.completion.visible)

	assert.True(t, query.handleTableCompletionKey(keyPress(tea.KeyDown, "", 0)))
	assert.Equal(t, 1, query.completion.selected)
	assert.True(t, query.handleTableCompletionKey(keyPress(tea.KeyUp, "", 0)))
	assert.Zero(t, query.completion.selected)
	assert.True(t, query.handleTableCompletionKey(keyPress(tea.KeyEscape, "", 0)))
	assert.False(t, query.completion.visible)
}

func TestTableCompletionOverlayPosition(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		height    int
		wantRow   int
		lineCount int
	}{
		{
			name:      "appears below the active token",
			sql:       "SELECT * FROM al",
			height:    4,
			wantRow:   1,
			lineCount: 4,
		},
		{
			name:      "appears above the active token at editor bottom",
			sql:       "SELECT * FROM Album\nSELECT * FROM al",
			height:    2,
			wantRow:   0,
			lineCount: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := newQueryModel(newAppLayout(100, 24))
			query.editor.SetWidth(40)
			query.editor.SetHeight(test.height)
			query.editor.SetValue(test.sql)
			query.refreshTableCompletion([]db.Table{{Name: "Album"}})

			lines := strings.Split(query.completionOverlay(query.editor.View()), "\n")
			require.Len(t, lines, test.lineCount)
			assert.Contains(t, ansi.Strip(lines[test.wantRow]), "> Album")
		})
	}
}

func TestQuotePostgreSQLIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{name: "ordinary name", identifier: "Album", want: `"Album"`},
		{name: "embedded quote", identifier: `has"quote`, want: `"has""quote"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, quotePostgreSQLIdentifier(test.identifier))
		})
	}
}
