package sqlhighlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLiteImplementsHighlighter(t *testing.T) {
	var highlighter Highlighter = SQLite{}

	assert.Equal(t, []Span{{Start: 0, End: 6}}, highlighter.KeywordSpans("SELECT"))
}

func TestSQLiteKeywordSpans(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "recognizes SQLite keywords case insensitively",
			input: "select materialized FROM albums WHERE id = 1",
			want:  []string{"select", "materialized", "FROM", "WHERE"},
		},
		{
			name:  "does not match keyword lookalikes",
			input: "selection my_select SELECT",
			want:  []string{"SELECT"},
		},
		{
			name:  "skips strings quoted identifiers and comments",
			input: "SELECT 'FROM' AS \"WHERE\" FROM `JOIN` [ORDER] -- SELECT\nWHERE id = 1 /* FROM */",
			want:  []string{"SELECT", "AS", "FROM", "WHERE"},
		},
		{
			name:  "skips doubled quote escapes",
			input: "SELECT 'it''s FROM' \"WHERE\"\"name\" FROM albums",
			want:  []string{"SELECT", "FROM"},
		},
		{
			name:  "treats every double dash as a line comment",
			input: "SELECT--FROM\nWHERE id = 1",
			want:  []string{"SELECT", "WHERE"},
		},
		{
			name:  "ends a SQLite block comment at its first closing delimiter",
			input: "/* outer /* SELECT */ FROM albums",
			want:  []string{"FROM"},
		},
		{
			name:  "recognizes the SQLite keyword vocabulary",
			input: "ABORT ACTION ADD AFTER ALL ALTER ALWAYS ANALYZE AND AS ASC ATTACH AUTOINCREMENT BEFORE BEGIN BETWEEN BY CASCADE CASE CAST CHECK COLLATE COLUMN COMMIT CONFLICT CONSTRAINT CREATE CROSS CURRENT CURRENT_DATE CURRENT_TIME CURRENT_TIMESTAMP DATABASE DEFAULT DEFERRABLE DEFERRED DELETE DESC DETACH DISTINCT DO DROP EACH ELSE END ESCAPE EXCEPT EXCLUDE EXCLUSIVE EXISTS EXPLAIN FAIL FILTER FIRST FOLLOWING FOR FOREIGN FROM FULL GENERATED GLOB GROUP GROUPS HAVING IF IGNORE IMMEDIATE IN INDEX INDEXED INITIALLY INNER INSERT INSTEAD INTERSECT INTO IS ISNULL JOIN KEY LAST LEFT LIKE LIMIT MATCH MATERIALIZED NATURAL NO NOT NOTHING NOTNULL NULL NULLS OF OFFSET ON OR ORDER OTHERS OUTER OVER PARTITION PLAN PRAGMA PRECEDING PRIMARY QUERY RAISE RANGE RECURSIVE REFERENCES REGEXP REINDEX RELEASE RENAME REPLACE RESTRICT RETURNING RIGHT ROLLBACK ROW ROWS SAVEPOINT SELECT SET TABLE TEMP TEMPORARY THEN TIES TO TRANSACTION TRIGGER UNBOUNDED UNION UNIQUE UPDATE USING VACUUM VALUES VIEW VIRTUAL WHEN WHERE WINDOW WITH WITHOUT",
			want:  []string{"ABORT", "ACTION", "ADD", "AFTER", "ALL", "ALTER", "ALWAYS", "ANALYZE", "AND", "AS", "ASC", "ATTACH", "AUTOINCREMENT", "BEFORE", "BEGIN", "BETWEEN", "BY", "CASCADE", "CASE", "CAST", "CHECK", "COLLATE", "COLUMN", "COMMIT", "CONFLICT", "CONSTRAINT", "CREATE", "CROSS", "CURRENT", "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP", "DATABASE", "DEFAULT", "DEFERRABLE", "DEFERRED", "DELETE", "DESC", "DETACH", "DISTINCT", "DO", "DROP", "EACH", "ELSE", "END", "ESCAPE", "EXCEPT", "EXCLUDE", "EXCLUSIVE", "EXISTS", "EXPLAIN", "FAIL", "FILTER", "FIRST", "FOLLOWING", "FOR", "FOREIGN", "FROM", "FULL", "GENERATED", "GLOB", "GROUP", "GROUPS", "HAVING", "IF", "IGNORE", "IMMEDIATE", "IN", "INDEX", "INDEXED", "INITIALLY", "INNER", "INSERT", "INSTEAD", "INTERSECT", "INTO", "IS", "ISNULL", "JOIN", "KEY", "LAST", "LEFT", "LIKE", "LIMIT", "MATCH", "MATERIALIZED", "NATURAL", "NO", "NOT", "NOTHING", "NOTNULL", "NULL", "NULLS", "OF", "OFFSET", "ON", "OR", "ORDER", "OTHERS", "OUTER", "OVER", "PARTITION", "PLAN", "PRAGMA", "PRECEDING", "PRIMARY", "QUERY", "RAISE", "RANGE", "RECURSIVE", "REFERENCES", "REGEXP", "REINDEX", "RELEASE", "RENAME", "REPLACE", "RESTRICT", "RETURNING", "RIGHT", "ROLLBACK", "ROW", "ROWS", "SAVEPOINT", "SELECT", "SET", "TABLE", "TEMP", "TEMPORARY", "THEN", "TIES", "TO", "TRANSACTION", "TRIGGER", "UNBOUNDED", "UNION", "UNIQUE", "UPDATE", "USING", "VACUUM", "VALUES", "VIEW", "VIRTUAL", "WHEN", "WHERE", "WINDOW", "WITH", "WITHOUT"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runes := []rune(test.input)
			spans := SQLite{}.KeywordSpans(test.input)
			got := make([]string, len(spans))
			for index, span := range spans {
				got[index] = string(runes[span.Start:span.End])
			}

			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkipBracketQuoted(t *testing.T) {
	assert.Equal(t, len([]rune("[SELECT]")), skipBracketQuoted([]rune("[SELECT] FROM"), 0))
	assert.Equal(t, len([]rune("[SELECT")), skipBracketQuoted([]rune("[SELECT"), 0))
}
