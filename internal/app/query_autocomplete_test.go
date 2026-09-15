package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/app/sqlhighlight"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestTableCompletionPrefix(t *testing.T) {
	tests := []struct {
		name        string
		engine      string
		sql         string
		want        string
		valid       bool
		highlighter sqlhighlight.Highlighter
	}{
		{name: "from", sql: "SELECT * FROM al", want: "al", valid: true},
		{name: "join", sql: "SELECT * FROM Artist JOIN al", want: "al", valid: true},
		{name: "delete from", sql: "DELETE FROM al", want: "al", valid: true},
		{name: "quoted identifier", sql: "SELECT * FROM \"al", valid: false},
		{name: "string", sql: "SELECT 'FROM al", valid: false},
		{name: "line comment", sql: "SELECT * -- FROM al", valid: false},
		{name: "block comment", sql: "SELECT /* FROM al", valid: false},
		{name: "non relation", sql: "SELECT al", valid: false},
		{name: "MySQL from", sql: "SELECT * FROM al", want: "al", valid: true, highlighter: sqlhighlight.MySQL{}},
		{name: "MySQL dash subtraction", sql: "SELECT 1--1 FROM al", want: "al", valid: true, highlighter: sqlhighlight.MySQL{}},
		{name: "MySQL escaped string", sql: "SELECT 'it\\'s' FROM al", want: "al", valid: true, highlighter: sqlhighlight.MySQL{}},
		{name: "MySQL hash comment", sql: "SELECT # FROM al", valid: false, highlighter: sqlhighlight.MySQL{}},
		{name: "MySQL backtick identifier", sql: "SELECT * FROM `al", valid: false, highlighter: sqlhighlight.MySQL{}},
		{name: "PostgreSQL truncate", sql: "TRUNCATE al", want: "al", valid: true},
		{name: "PostgreSQL truncate table", sql: "TRUNCATE TABLE al", want: "al", valid: true},
		{name: "MySQL truncate", engine: db.EngineMySQL, sql: "TRUNCATE al", want: "al", valid: true, highlighter: sqlhighlight.MySQL{}},
		{name: "MySQL truncate table", engine: db.EngineMySQL, sql: "TRUNCATE TABLE al", want: "al", valid: true, highlighter: sqlhighlight.MySQL{}},
		{name: "Oracle bare truncate", engine: db.EngineOracle, sql: "TRUNCATE al", valid: false, highlighter: sqlhighlight.Oracle{}},
		{name: "Oracle truncate table", engine: db.EngineOracle, sql: "TRUNCATE TABLE al", want: "al", valid: true, highlighter: sqlhighlight.Oracle{}},
		{name: "SQL Server bare truncate", engine: db.EngineSQLServer, sql: "TRUNCATE al", valid: false, highlighter: sqlhighlight.SQLServer{}},
		{name: "SQL Server truncate table", engine: db.EngineSQLServer, sql: "TRUNCATE TABLE al", want: "al", valid: true, highlighter: sqlhighlight.SQLServer{}},
		{name: "SQLite truncate", engine: db.EngineSQLite, sql: "TRUNCATE al", valid: false, highlighter: sqlhighlight.SQLite{}},
		{name: "Oracle from", sql: "SELECT * FROM al", want: "al", valid: true, highlighter: sqlhighlight.Oracle{}},
		{name: "Oracle alternative quote", sql: "SELECT q'[FROM al", valid: false, highlighter: sqlhighlight.Oracle{}},
		{name: "SQLite from", sql: "SELECT * FROM al", want: "al", valid: true, highlighter: sqlhighlight.SQLite{}},
		{name: "SQL Server from", sql: "SELECT * FROM al", want: "al", valid: true, highlighter: sqlhighlight.SQLServer{}},
		{name: "SQL Server bracket identifier", sql: "SELECT * FROM [al", valid: false, highlighter: sqlhighlight.SQLServer{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := newQueryModel(newAppLayout(100, 24))
			query.editor.SetValue(test.sql)
			highlighter := test.highlighter
			if highlighter == nil {
				highlighter = sqlhighlight.PostgreSQL{}
			}
			engine := test.engine
			if engine == "" {
				engine = db.EnginePostgreSQL
			}
			_, _, got, ok := tableCompletionPrefix(query.editor, engine, highlighter)
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
		engine   string
		sql      string
		cursorAt string
		key      rune
		want     string
	}{
		{
			name:     "enter replaces active token",
			engine:   db.EnginePostgreSQL,
			sql:      "SELECT * FROM al WHERE id = 1",
			cursorAt: "SELECT * FROM al",
			key:      tea.KeyEnter,
			want:     "SELECT * FROM \"Album\" WHERE id = 1",
		},
		{
			name:     "tab replaces active token",
			engine:   db.EnginePostgreSQL,
			sql:      "SELECT * FROM al",
			cursorAt: "SELECT * FROM al",
			key:      tea.KeyTab,
			want:     "SELECT * FROM \"Album\"",
		},
		{
			name:     "replaces whole token when cursor is in the middle",
			engine:   db.EnginePostgreSQL,
			sql:      "SELECT * FROM alxyz",
			cursorAt: "SELECT * FROM al",
			key:      tea.KeyEnter,
			want:     "SELECT * FROM \"Album\"",
		},
		{name: "MySQL", engine: db.EngineMySQL, sql: "SELECT * FROM al", cursorAt: "SELECT * FROM al", key: tea.KeyEnter, want: "SELECT * FROM `Album`"},
		{name: "Oracle", engine: db.EngineOracle, sql: "SELECT * FROM al", cursorAt: "SELECT * FROM al", key: tea.KeyEnter, want: "SELECT * FROM \"Album\""},
		{name: "SQLite", engine: db.EngineSQLite, sql: "SELECT * FROM al", cursorAt: "SELECT * FROM al", key: tea.KeyEnter, want: "SELECT * FROM \"Album\""},
		{name: "SQL Server", engine: db.EngineSQLServer, sql: "SELECT * FROM al", cursorAt: "SELECT * FROM al", key: tea.KeyEnter, want: "SELECT * FROM [Album]"},
		{name: "PostgreSQL truncate", engine: db.EnginePostgreSQL, sql: "TRUNCATE al", cursorAt: "TRUNCATE al", key: tea.KeyEnter, want: "TRUNCATE \"Album\""},
		{name: "PostgreSQL truncate table", engine: db.EnginePostgreSQL, sql: "TRUNCATE TABLE al", cursorAt: "TRUNCATE TABLE al", key: tea.KeyEnter, want: "TRUNCATE TABLE \"Album\""},
		{name: "MySQL truncate", engine: db.EngineMySQL, sql: "TRUNCATE al", cursorAt: "TRUNCATE al", key: tea.KeyEnter, want: "TRUNCATE `Album`"},
		{name: "MySQL truncate table", engine: db.EngineMySQL, sql: "TRUNCATE TABLE al", cursorAt: "TRUNCATE TABLE al", key: tea.KeyEnter, want: "TRUNCATE TABLE `Album`"},
		{name: "Oracle truncate table", engine: db.EngineOracle, sql: "TRUNCATE TABLE al", cursorAt: "TRUNCATE TABLE al", key: tea.KeyEnter, want: "TRUNCATE TABLE \"Album\""},
		{name: "SQL Server truncate table", engine: db.EngineSQLServer, sql: "TRUNCATE TABLE al", cursorAt: "TRUNCATE TABLE al", key: tea.KeyEnter, want: "TRUNCATE TABLE [Album]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := newQueryModel(newAppLayout(100, 24))
			query.editor.SetValue(test.sql)
			query.restoreEditorCursorOffset(len([]rune(test.cursorAt)))
			query.refreshTableCompletion([]db.Table{{Name: "Album"}, {Name: "Artist"}}, test.engine, rawQueryHighlighter(&fakeDatabase{engine: test.engine}))
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
	query.refreshTableCompletion([]db.Table{{Name: "Album"}, {Name: "AlbumTrack"}}, db.EnginePostgreSQL, sqlhighlight.PostgreSQL{})
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
			query.refreshTableCompletion([]db.Table{{Name: "Album"}}, db.EnginePostgreSQL, sqlhighlight.PostgreSQL{})

			lines := strings.Split(query.completionOverlay(query.editor.View()), "\n")
			require.Len(t, lines, test.lineCount)
			assert.Contains(t, ansi.Strip(lines[test.wantRow]), "> Album")
		})
	}
}

func TestQuoteTableCompletionIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		engine     string
		identifier string
		want       string
	}{
		{name: "PostgreSQL", engine: db.EnginePostgreSQL, identifier: `has"quote`, want: `"has""quote"`},
		{name: "MySQL", engine: db.EngineMySQL, identifier: "has`quote", want: "`has``quote`"},
		{name: "Oracle", engine: db.EngineOracle, identifier: `has"quote`, want: `"has""quote"`},
		{name: "SQLite", engine: db.EngineSQLite, identifier: `has"quote`, want: `"has""quote"`},
		{name: "SQL Server", engine: db.EngineSQLServer, identifier: "has]quote", want: "[has]]quote]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, quoteTableCompletionIdentifier(test.engine, test.identifier))
		})
	}
}
