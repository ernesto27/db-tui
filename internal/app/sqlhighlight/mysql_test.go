package sqlhighlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLImplementsHighlighter(t *testing.T) {
	var highlighter Highlighter = MySQL{}

	assert.Equal(t, []Span{{Start: 0, End: 6}}, highlighter.KeywordSpans("SELECT"))
}

func TestMySQLKeywordSpans(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "recognizes MySQL reserved keywords case insensitively",
			input: "select interval, Qualify FROM orders WHERE id = 1",
			want:  []string{"select", "interval", "Qualify", "FROM", "WHERE"},
		},
		{
			name:  "skips strings backtick identifiers and comments",
			input: "SELECT 'FROM' AS `WHERE` FROM orders # SELECT\nWHERE id = 1",
			want:  []string{"SELECT", "AS", "FROM", "WHERE"},
		},
		{
			name:  "recognizes reserved words across the MySQL 8.4 vocabulary",
			input: "ACCESSIBLE ASENSITIVE DAY_MICROSECOND JSON_TABLE QUALIFY ROW_NUMBER TABLESAMPLE _FILENAME",
			want:  []string{"ACCESSIBLE", "ASENSITIVE", "DAY_MICROSECOND", "JSON_TABLE", "QUALIFY", "ROW_NUMBER", "TABLESAMPLE", "_FILENAME"},
		},
		{
			name:  "skips MySQL string and identifier escape forms",
			input: "SELECT 'it\\'s FROM' \"WHERE\" `JOIN``name` FROM orders",
			want:  []string{"SELECT", "FROM"},
		},
		{
			name:  "skips hash and whitespace-prefixed dash comments",
			input: "SELECT # FROM\nWHERE id = 1; SELECT -- FROM\nWHERE id = 2",
			want:  []string{"SELECT", "WHERE", "SELECT", "WHERE"},
		},
		{
			name:  "does not treat dashes without whitespace as a comment",
			input: "SELECT--FROM",
			want:  []string{"SELECT", "FROM"},
		},
		{
			name:  "does not highlight reserved identifiers after compact or spaced qualifiers",
			input: "SELECT orders.interval, orders . interval FROM orders",
			want:  []string{"SELECT", "FROM"},
		},
		{
			name:  "ends a MySQL block comment at its first closing delimiter",
			input: "/* outer /* SELECT */ FROM orders",
			want:  []string{"FROM"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runes := []rune(test.input)
			spans := MySQL{}.KeywordSpans(test.input)
			got := make([]string, len(spans))
			for index, span := range spans {
				got[index] = string(runes[span.Start:span.End])
			}

			assert.Equal(t, test.want, got)
		})
	}
}
