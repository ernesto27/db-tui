package sqlhighlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostgreSQLImplementsHighlighter(t *testing.T) {
	var highlighter Highlighter = PostgreSQL{}

	assert.Equal(t, []Span{{Start: 0, End: 6}}, highlighter.KeywordSpans("SELECT"))
}

func TestPostgreSQLKeywordSpans(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "recognizes complete keywords case insensitively", input: "select * FrOm albums WHERE id = 1", want: []string{"select", "FrOm", "WHERE"}},
		{name: "does not match keyword lookalikes", input: "selection my_select SELECT", want: []string{"SELECT"}},
		{name: "skips quoted strings and quoted identifiers", input: `SELECT 'FROM' AS "WHERE" FROM albums`, want: []string{"SELECT", "AS", "FROM"}},
		{name: "skips escaped quotes inside strings", input: `SELECT 'it''s FROM' FROM albums`, want: []string{"SELECT", "FROM"}},
		{name: "skips line comments and resumes after newline", input: "-- SELECT FROM\nSELECT", want: []string{"SELECT"}},
		{name: "skips nested block comments", input: "/* outer /* SELECT */ FROM */ SELECT", want: []string{"SELECT"}},
		{name: "skips tagged dollar quoted text", input: "$body$ SELECT FROM $body$ SELECT", want: []string{"SELECT"}},
		{name: "skips untagged dollar quoted text", input: "$$ SELECT FROM $$ SELECT", want: []string{"SELECT"}},
		{name: "continues identifiers through dollar signs", input: "SELECT foo$tag$ FROM albums", want: []string{"SELECT", "FROM"}},
		{name: "skips backslash escaped quotes in escape strings", input: `SELECT E'it\'s FROM here' AS value`, want: []string{"SELECT", "AS"}},
		{
			name:  "recognizes the missing PostgreSQL reserved keywords",
			input: "ANALYSE ANALYZE ANY ARRAY ASC ASYMMETRIC AUTHORIZATION BINARY BOTH CAST CHECK COLLATE COLLATION COLUMN CONCURRENTLY CONSTRAINT CROSS CURRENT_CATALOG CURRENT_DATE CURRENT_ROLE CURRENT_TIME CURRENT_TIMESTAMP CURRENT_USER DEFAULT DEFERRABLE DESC DO EXCEPT FALSE FETCH FOR FOREIGN FREEZE GRANT ILIKE IN INITIALLY INTERSECT IS ISNULL LATERAL LEADING LIKE LOCALTIME LOCALTIMESTAMP NATURAL NOTNULL ONLY OUTER OVERLAPS PLACING PRIMARY REFERENCES RETURNING SESSION_USER SIMILAR SOME SYMMETRIC SYSTEM_USER TABLESAMPLE TO TRAILING TRUE UNIQUE USER USING VARIADIC VERBOSE WINDOW WITH",
			want:  []string{"ANALYSE", "ANALYZE", "ANY", "ARRAY", "ASC", "ASYMMETRIC", "AUTHORIZATION", "BINARY", "BOTH", "CAST", "CHECK", "COLLATE", "COLLATION", "COLUMN", "CONCURRENTLY", "CONSTRAINT", "CROSS", "CURRENT_CATALOG", "CURRENT_DATE", "CURRENT_ROLE", "CURRENT_TIME", "CURRENT_TIMESTAMP", "CURRENT_USER", "DEFAULT", "DEFERRABLE", "DESC", "DO", "EXCEPT", "FALSE", "FETCH", "FOR", "FOREIGN", "FREEZE", "GRANT", "ILIKE", "IN", "INITIALLY", "INTERSECT", "IS", "ISNULL", "LATERAL", "LEADING", "LIKE", "LOCALTIME", "LOCALTIMESTAMP", "NATURAL", "NOTNULL", "ONLY", "OUTER", "OVERLAPS", "PLACING", "PRIMARY", "REFERENCES", "RETURNING", "SESSION_USER", "SIMILAR", "SOME", "SYMMETRIC", "SYSTEM_USER", "TABLESAMPLE", "TO", "TRAILING", "TRUE", "UNIQUE", "USER", "USING", "VARIADIC", "VERBOSE", "WINDOW", "WITH"},
		},
		{name: "does not color PostgreSQL words that may be identifiers", input: "SELECT between, exists FROM albums", want: []string{"SELECT", "FROM"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runes := []rune(test.input)
			spans := PostgreSQL{}.KeywordSpans(test.input)
			got := make([]string, len(spans))
			for index, span := range spans {
				got[index] = string(runes[span.Start:span.End])
			}
			assert.Equal(t, test.want, got)
		})
	}
}

func TestIsDollarQuoteTagRune(t *testing.T) {
	assert.True(t, isDollarQuoteTagRune('_'))
	assert.True(t, isDollarQuoteTagRune('a'))
	assert.True(t, isDollarQuoteTagRune('7'))
	assert.False(t, isDollarQuoteTagRune('$'))
	assert.False(t, isDollarQuoteTagRune('-'))
}

func TestIsEscapeStringStart(t *testing.T) {
	tests := []struct {
		name  string
		input string
		index int
		want  bool
	}{
		{name: "uppercase prefix", input: "E'value'", want: true},
		{name: "lowercase prefix", input: "e'value'", want: true},
		{name: "identifier continuation", input: "nameE'value'", index: 4, want: false},
		{name: "missing quote", input: "Evalue", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isEscapeStringStart([]rune(test.input), test.index))
		})
	}
}

func TestIsIdentifierStartRune(t *testing.T) {
	assert.True(t, isIdentifierStartRune('_'))
	assert.True(t, isIdentifierStartRune('a'))
	assert.True(t, isIdentifierStartRune('é'))
	assert.False(t, isIdentifierStartRune('7'))
	assert.False(t, isIdentifierStartRune('$'))
}

func TestDollarQuoteDelimiterAt(t *testing.T) {
	tests := []struct {
		name, input, want string
		index             int
		found             bool
	}{
		{name: "untagged delimiter", input: "$$ body $$", want: "$$", found: true},
		{name: "tagged delimiter", input: "$body$ SELECT", want: "$body$", found: true},
		{name: "delimiter after prefix", input: "x$fn_2$", index: 1, want: "$fn_2$", found: true},
		{name: "digit cannot start tag", input: "$5$"}, {name: "missing closing dollar", input: "$body"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			delimiter, found := dollarQuoteDelimiterAt([]rune(test.input), test.index)
			assert.Equal(t, test.found, found)
			if found {
				assert.Equal(t, test.want, string(delimiter))
			}
		})
	}
}

func TestSkipDollarQuoted(t *testing.T) {
	assert.Equal(t, len([]rune("$body$ SELECT $body$")), skipDollarQuoted([]rune("$body$ SELECT $body$ trailing"), []rune("$body$"), len([]rune("$body$"))))
	assert.Equal(t, len([]rune("SELECT $other$ more $body$")), skipDollarQuoted([]rune("SELECT $other$ more $body$ trailing"), []rune("$body$"), 0))
	assert.Equal(t, len([]rune("SELECT FROM")), skipDollarQuoted([]rune("SELECT FROM"), []rune("$$"), 0))
}

func TestRunesEqualAt(t *testing.T) {
	assert.True(t, runesEqualAt([]rune("$body$"), []rune("$body$"), 0))
	assert.True(t, runesEqualAt([]rune("x$body$"), []rune("$body$"), 1))
	assert.False(t, runesEqualAt([]rune("$body$"), []rune("$other$"), 0))
	assert.False(t, runesEqualAt([]rune("$body"), []rune("$body$"), 0))
}
