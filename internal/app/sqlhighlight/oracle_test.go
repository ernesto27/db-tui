package sqlhighlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOracleImplementsHighlighter(t *testing.T) {
	var highlighter Highlighter = Oracle{}

	assert.Equal(t, []Span{{Start: 0, End: 6}}, highlighter.KeywordSpans("SELECT"))
}

func TestOracleKeywordSpans(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "recognizes Oracle keywords case insensitively", input: "select rownum FROM dual WHERE id = 1", want: []string{"select", "rownum", "FROM", "WHERE"}},
		{name: "does not match keyword lookalikes", input: "selection my_select SELECT", want: []string{"SELECT"}},
		{name: "skips strings quoted identifiers and comments", input: "SELECT 'FROM' AS \"WHERE\" FROM dual -- SELECT\nWHERE id = 1 /* FROM */", want: []string{"SELECT", "AS", "FROM", "WHERE"}},
		{name: "skips doubled quote escapes", input: "SELECT 'it''s FROM' FROM dual", want: []string{"SELECT", "FROM"}},
		{name: "skips paired alternative quoted strings", input: "SELECT q'{FROM WHERE}' AS value FROM dual", want: []string{"SELECT", "AS", "FROM"}},
		{name: "skips national alternative quoted strings", input: "SELECT nq'!FROM WHERE!' AS value FROM dual", want: []string{"SELECT", "AS", "FROM"}},
		{name: "recognizes Oracle reserved and structural keywords", input: "ACCESS ADD ALTER AUDIT CONNECT MERGE MINUS ROWNUM SYSDATE VARCHAR2 BEGIN DECLARE EXCEPTION JOIN OFFSET RETURNING", want: []string{"ACCESS", "ADD", "ALTER", "AUDIT", "CONNECT", "MERGE", "MINUS", "ROWNUM", "SYSDATE", "VARCHAR2", "BEGIN", "DECLARE", "EXCEPTION", "JOIN", "OFFSET", "RETURNING"}},
		{name: "ends an Oracle block comment at its first closing delimiter", input: "/* outer /* SELECT */ FROM dual", want: []string{"FROM"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runes := []rune(test.input)
			spans := Oracle{}.KeywordSpans(test.input)
			got := make([]string, len(spans))
			for index, span := range spans {
				got[index] = string(runes[span.Start:span.End])
			}

			assert.Equal(t, test.want, got)
		})
	}
}

func TestOracleClosingQuoteDelimiter(t *testing.T) {
	assert.Equal(t, ']', oracleClosingQuoteDelimiter('['))
	assert.Equal(t, '}', oracleClosingQuoteDelimiter('{'))
	assert.Equal(t, ')', oracleClosingQuoteDelimiter('('))
	assert.Equal(t, '>', oracleClosingQuoteDelimiter('<'))
	assert.Equal(t, '!', oracleClosingQuoteDelimiter('!'))
}
