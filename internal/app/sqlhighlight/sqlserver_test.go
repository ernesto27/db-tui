package sqlhighlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLServerImplementsHighlighter(t *testing.T) {
	var highlighter Highlighter = SQLServer{}

	assert.Equal(t, []Span{{Start: 0, End: 6}}, highlighter.KeywordSpans("SELECT"))
}

func TestSQLServerKeywordSpans(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "recognizes SQL Server keywords case insensitively", input: "select TOP 10 FROM orders WHERE id = 1", want: []string{"select", "TOP", "FROM", "WHERE"}},
		{name: "does not match keyword lookalikes", input: "selection my_select SELECT", want: []string{"SELECT"}},
		{name: "skips strings quoted identifiers and comments", input: "SELECT 'FROM' AS \"WHERE\" FROM [JOIN] -- SELECT\nWHERE id = 1 /* FROM */", want: []string{"SELECT", "AS", "FROM", "WHERE"}},
		{name: "skips doubled quote escapes", input: "SELECT 'it''s FROM' FROM orders", want: []string{"SELECT", "FROM"}},
		{name: "skips bracket identifiers with escaped closing brackets", input: "SELECT [FROM]]WHERE] FROM orders", want: []string{"SELECT", "FROM"}},
		{name: "skips nested block comments", input: "/* outer /* SELECT */ FROM */ SELECT", want: []string{"SELECT"}},
		{name: "does not highlight qualified identifiers or variables", input: "SELECT dbo . table, @select, @@rowcount FROM orders", want: []string{"SELECT", "FROM"}},
		{name: "recognizes SQL Server reserved keywords", input: "BACKUP BEGIN BULK DBCC EXEC MERGE PIVOT RAISERROR TABLESAMPLE TRY_CONVERT UNPIVOT WITHIN WRITETEXT", want: []string{"BACKUP", "BEGIN", "BULK", "DBCC", "EXEC", "MERGE", "PIVOT", "RAISERROR", "TABLESAMPLE", "TRY_CONVERT", "UNPIVOT", "WITHIN", "WRITETEXT"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runes := []rune(test.input)
			spans := SQLServer{}.KeywordSpans(test.input)
			got := make([]string, len(spans))
			for index, span := range spans {
				got[index] = string(runes[span.Start:span.End])
			}

			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkipSQLServerBracketQuoted(t *testing.T) {
	assert.Equal(t, len([]rune("[SELECT]]FROM]")), skipSQLServerBracketQuoted([]rune("[SELECT]]FROM] WHERE"), 0))
	assert.Equal(t, len([]rune("[SELECT")), skipSQLServerBracketQuoted([]rune("[SELECT"), 0))
}
