package sqlhighlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeywordSpans(t *testing.T) {
	keywords := map[string]struct{}{"SELECT": {}, "FROM": {}}
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "matches complete keywords case insensitively",
			input: "select FROM selection my_from",
			want:  []string{"select", "FROM"},
		},
		{
			name:  "uses the lexical skipper",
			input: "SELECT 'FROM' FROM",
			want:  []string{"SELECT", "FROM"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spans := keywordSpans(test.input, keywords, skipQuotedRegion, nil)

			assert.Equal(t, test.want, spanText(test.input, spans))
		})
	}
}

func TestKeywordSpansUsesCandidateFilter(t *testing.T) {
	keywords := map[string]struct{}{"SELECT": {}, "INTERVAL": {}}
	input := "SELECT orders.INTERVAL"

	spans := keywordSpans(input, keywords, nil, func(runes []rune, start, _ int) bool {
		return start == 0 || runes[start-1] != '.'
	})

	assert.Equal(t, []string{"SELECT"}, spanText(input, spans))
}

func TestIsIdentifierContinuationRune(t *testing.T) {
	tests := []struct {
		name string
		rune rune
		want bool
	}{
		{name: "underscore", rune: '_', want: true}, {name: "letter", rune: 'a', want: true},
		{name: "digit", rune: '7', want: true}, {name: "unicode letter", rune: 'é', want: true},
		{name: "dollar sign", rune: '$', want: true}, {name: "hyphen", rune: '-', want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { assert.Equal(t, test.want, isIdentifierContinuationRune(test.rune)) })
	}
}

func TestSkipQuoted(t *testing.T) {
	tests := []struct {
		name, input string
		want        int
	}{
		{name: "closes at single quote", input: "'value' trailing", want: len([]rune("'value'"))},
		{name: "skips doubled quote escape", input: "'it''s' trailing", want: len([]rune("'it''s'"))},
		{name: "consumes unterminated string", input: "'value", want: len([]rune("'value"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, skipQuoted([]rune(test.input), 0, '\'', false))
		})
	}
}

func TestSkipQuotedWithBackslashEscapes(t *testing.T) {
	input := ` 'it\'s FROM' trailing`
	assert.Equal(t, len([]rune(` 'it\'s FROM'`)), skipQuoted([]rune(input), 1, '\'', true))
}

func TestSkipQuotedDoubleQuotes(t *testing.T) {
	tests := []struct {
		name, input string
		want        int
	}{
		{name: "closes at double quote", input: "\"column\" trailing", want: len([]rune("\"column\""))},
		{name: "skips doubled quote escape", input: "\"a\"\"b\" trailing", want: len([]rune("\"a\"\"b\""))},
		{name: "consumes unterminated identifier", input: "\"column", want: len([]rune("\"column"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, skipQuoted([]rune(test.input), 0, '"', false))
		})
	}
}

func TestSkipLineComment(t *testing.T) {
	assert.Equal(t, len([]rune("-- SELECT")), skipLineComment([]rune("-- SELECT\nFROM"), 0))
	assert.Equal(t, len([]rune("-- SELECT")), skipLineComment([]rune("-- SELECT"), 0))
}

func TestSkipBlockComment(t *testing.T) {
	assert.Equal(t, len([]rune("/* SELECT */")), skipBlockComment([]rune("/* SELECT */ FROM"), 0))
	assert.Equal(t, len([]rune("/* SELECT")), skipBlockComment([]rune("/* SELECT"), 0))
}

func TestSkipNestedBlockComment(t *testing.T) {
	assert.Equal(t, len([]rune("/* SELECT */")), skipNestedBlockComment([]rune("/* SELECT */ FROM"), 0))
	assert.Equal(t, len([]rune("/* outer /* inner */ end */")), skipNestedBlockComment([]rune("/* outer /* inner */ end */ SELECT"), 0))
	assert.Equal(t, len([]rune("/* SELECT")), skipNestedBlockComment([]rune("/* SELECT"), 0))
}

func skipQuotedRegion(runes []rune, index int) (int, bool) {
	if runes[index] != '\'' {
		return 0, false
	}
	return skipQuoted(runes, index, '\'', false), true
}

func spanText(value string, spans []Span) []string {
	runes := []rune(value)
	text := make([]string, len(spans))
	for index, span := range spans {
		text[index] = string(runes[span.Start:span.End])
	}
	return text
}
