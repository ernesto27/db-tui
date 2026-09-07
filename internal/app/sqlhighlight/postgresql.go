// Package sqlhighlight finds SQL token spans for query-editor rendering.
package sqlhighlight

import "unicode"

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
	return keywordSpans(value, postgreSQLKeywords, skipPostgreSQLRegion, nil)
}

func skipPostgreSQLRegion(runes []rune, index int) (int, bool) {
	switch {
	case isEscapeStringStart(runes, index):
		return skipQuoted(runes, index+1, '\'', true), true
	case runes[index] == '\'':
		return skipQuoted(runes, index, '\'', false), true
	case runes[index] == '"':
		return skipQuoted(runes, index, '"', false), true
	case index+1 < len(runes) && runes[index] == '-' && runes[index+1] == '-':
		return skipLineComment(runes, index), true
	case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
		return skipNestedBlockComment(runes, index), true
	case runes[index] == '$':
		if delimiter, ok := dollarQuoteDelimiterAt(runes, index); ok {
			return skipDollarQuoted(runes, delimiter, index+len(delimiter)), true
		}
	}
	return 0, false
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
