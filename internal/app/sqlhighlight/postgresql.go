// Package sqlhighlight finds SQL token spans for query-editor rendering.
package sqlhighlight

import (
	"strings"
	"unicode"
)

// Span identifies a half-open rune range in SQL source text.
type Span struct {
	Start int
	End   int
}

// Highlighter finds source spans that a caller may render with highlighting.
type Highlighter interface {
	KeywordSpans(value string) []Span
}

// PostgreSQL finds the supported PostgreSQL keyword spans in SQL source text.
type PostgreSQL struct{}

var _ Highlighter = PostgreSQL{}

var postgreSQLKeywords = map[string]struct{}{
	"SELECT": {}, "FROM": {}, "WHERE": {}, "JOIN": {},
	"INNER": {}, "LEFT": {}, "RIGHT": {}, "FULL": {}, "ON": {}, "AS": {},
	"INSERT": {}, "INTO": {}, "VALUES": {}, "UPDATE": {}, "SET": {}, "DELETE": {},
	"CREATE": {}, "ALTER": {}, "DROP": {}, "TABLE": {},
	"ORDER": {}, "BY": {}, "GROUP": {}, "HAVING": {}, "LIMIT": {}, "OFFSET": {},
	"UNION": {}, "ALL": {}, "DISTINCT": {}, "AND": {}, "OR": {}, "NOT": {},
	"NULL": {}, "CASE": {}, "WHEN": {}, "THEN": {}, "ELSE": {}, "END": {},
	"ANALYSE": {}, "ANALYZE": {}, "ANY": {}, "ARRAY": {}, "ASC": {}, "ASYMMETRIC": {},
	"AUTHORIZATION": {}, "BINARY": {}, "BOTH": {}, "CAST": {}, "CHECK": {}, "COLLATE": {},
	"COLLATION": {}, "COLUMN": {}, "CONCURRENTLY": {}, "CONSTRAINT": {}, "CROSS": {},
	"CURRENT_CATALOG": {}, "CURRENT_DATE": {}, "CURRENT_ROLE": {}, "CURRENT_TIME": {},
	"CURRENT_TIMESTAMP": {}, "CURRENT_USER": {}, "DEFAULT": {}, "DEFERRABLE": {}, "DESC": {},
	"DO": {}, "EXCEPT": {}, "FALSE": {}, "FETCH": {}, "FOR": {}, "FOREIGN": {}, "FREEZE": {},
	"GRANT": {}, "IN": {}, "INITIALLY": {}, "INTERSECT": {}, "LATERAL": {}, "LEADING": {},
	"ILIKE": {}, "IS": {}, "ISNULL": {}, "LIKE": {}, "LOCALTIME": {}, "LOCALTIMESTAMP": {},
	"NATURAL": {}, "NOTNULL": {}, "ONLY": {}, "OUTER": {}, "OVERLAPS": {}, "PLACING": {}, "PRIMARY": {},
	"REFERENCES": {}, "RETURNING": {}, "SESSION_USER": {}, "SOME": {}, "SYMMETRIC": {},
	"SIMILAR": {}, "SYSTEM_USER": {}, "TABLESAMPLE": {}, "TO": {}, "TRAILING": {}, "TRUE": {},
	"UNIQUE": {}, "USER": {}, "USING": {}, "VARIADIC": {}, "VERBOSE": {}, "WINDOW": {}, "WITH": {},
}

// KeywordSpans implements Highlighter.
func (PostgreSQL) KeywordSpans(value string) []Span {
	runes := []rune(value)
	var spans []Span

	for index := 0; index < len(runes); {
		switch {
		case isEscapeStringStart(runes, index):
			index = skipSingleQuoted(runes, index+1, true)
			continue
		case runes[index] == '\'':
			index = skipSingleQuoted(runes, index, false)
			continue
		case runes[index] == '"':
			index = skipDoubleQuoted(runes, index)
			continue
		case index+1 < len(runes) && runes[index] == '-' && runes[index+1] == '-':
			index = skipLineComment(runes, index)
			continue
		case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
			index = skipBlockComment(runes, index)
			continue
		case runes[index] == '$':
			if delimiter, ok := dollarQuoteDelimiterAt(runes, index); ok {
				index = skipDollarQuoted(runes, delimiter, index+len(delimiter))
				continue
			}
		}

		if !isIdentifierContinuationRune(runes[index]) {
			index++
			continue
		}

		start := index
		for index < len(runes) && isIdentifierContinuationRune(runes[index]) {
			index++
		}

		if _, ok := postgreSQLKeywords[strings.ToUpper(string(runes[start:index]))]; ok {
			spans = append(spans, Span{Start: start, End: index})
		}
	}

	return spans
}

func isIdentifierContinuationRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isDollarQuoteTagRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isEscapeStringStart(runes []rune, index int) bool {
	if index+1 >= len(runes) || (runes[index] != 'E' && runes[index] != 'e') || runes[index+1] != '\'' {
		return false
	}
	return index == 0 || !isIdentifierContinuationRune(runes[index-1])
}

func skipSingleQuoted(runes []rune, index int, escapeString bool) int {
	index++
	for index < len(runes) {
		if escapeString && runes[index] == '\\' && index+1 < len(runes) {
			index += 2
			continue
		}
		if runes[index] != '\'' {
			index++
			continue
		}
		if index+1 < len(runes) && runes[index+1] == '\'' {
			index += 2
			continue
		}
		return index + 1
	}
	return len(runes)
}

func skipDoubleQuoted(runes []rune, index int) int {
	index++
	for index < len(runes) {
		if runes[index] != '"' {
			index++
			continue
		}
		if index+1 < len(runes) && runes[index+1] == '"' {
			index += 2
			continue
		}
		return index + 1
	}
	return len(runes)
}

func skipLineComment(runes []rune, index int) int {
	index += 2
	for index < len(runes) && runes[index] != '\n' {
		index++
	}
	return index
}

func skipBlockComment(runes []rune, index int) int {
	depth := 1
	index += 2
	for index < len(runes) {
		switch {
		case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
			depth++
			index += 2
		case index+1 < len(runes) && runes[index] == '*' && runes[index+1] == '/':
			depth--
			index += 2
			if depth == 0 {
				return index
			}
		default:
			index++
		}
	}
	return len(runes)
}

func isIdentifierStartRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func dollarQuoteDelimiterAt(runes []rune, index int) ([]rune, bool) {
	if runes[index] != '$' {
		return nil, false
	}

	end := index + 1
	if end < len(runes) && runes[end] == '$' {
		return runes[index : end+1], true
	}
	if end >= len(runes) || !isIdentifierStartRune(runes[end]) {
		return nil, false
	}

	end++
	for end < len(runes) && isDollarQuoteTagRune(runes[end]) {
		end++
	}
	if end >= len(runes) || runes[end] != '$' {
		return nil, false
	}
	return runes[index : end+1], true
}

func skipDollarQuoted(runes, delimiter []rune, index int) int {
	for index < len(runes) {
		if runesEqualAt(runes, delimiter, index) {
			return index + len(delimiter)
		}
		index++
	}
	return len(runes)
}

func runesEqualAt(runes, want []rune, index int) bool {
	if index+len(want) > len(runes) {
		return false
	}
	for offset, r := range want {
		if runes[index+offset] != r {
			return false
		}
	}
	return true
}
