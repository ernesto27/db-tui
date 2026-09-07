package sqlhighlight

// SQLite finds the supported SQLite keyword spans in SQL source text.
type SQLite struct{}

var _ Highlighter = SQLite{}

var sqliteKeywords = map[string]struct{}{
	"ABORT": {}, "ACTION": {}, "ADD": {}, "AFTER": {}, "ALL": {}, "ALTER": {},
	"ALWAYS": {}, "ANALYZE": {}, "AND": {}, "AS": {}, "ASC": {}, "ATTACH": {},
	"AUTOINCREMENT": {}, "BEFORE": {}, "BEGIN": {}, "BETWEEN": {}, "BY": {}, "CASCADE": {},
	"CASE": {}, "CAST": {}, "CHECK": {}, "COLLATE": {}, "COLUMN": {}, "COMMIT": {},
	"CONFLICT": {}, "CONSTRAINT": {}, "CREATE": {}, "CROSS": {}, "CURRENT": {}, "CURRENT_DATE": {},
	"CURRENT_TIME": {}, "CURRENT_TIMESTAMP": {}, "DATABASE": {}, "DEFAULT": {}, "DEFERRABLE": {}, "DEFERRED": {},
	"DELETE": {}, "DESC": {}, "DETACH": {}, "DISTINCT": {}, "DO": {}, "DROP": {},
	"EACH": {}, "ELSE": {}, "END": {}, "ESCAPE": {}, "EXCEPT": {}, "EXCLUDE": {},
	"EXCLUSIVE": {}, "EXISTS": {}, "EXPLAIN": {}, "FAIL": {}, "FILTER": {}, "FIRST": {},
	"FOLLOWING": {}, "FOR": {}, "FOREIGN": {}, "FROM": {}, "FULL": {}, "GENERATED": {},
	"GLOB": {}, "GROUP": {}, "GROUPS": {}, "HAVING": {}, "IF": {}, "IGNORE": {},
	"IMMEDIATE": {}, "IN": {}, "INDEX": {}, "INDEXED": {}, "INITIALLY": {}, "INNER": {},
	"INSERT": {}, "INSTEAD": {}, "INTERSECT": {}, "INTO": {}, "IS": {}, "ISNULL": {},
	"JOIN": {}, "KEY": {}, "LAST": {}, "LEFT": {}, "LIKE": {}, "LIMIT": {},
	"MATCH": {}, "MATERIALIZED": {}, "NATURAL": {}, "NO": {}, "NOT": {}, "NOTHING": {},
	"NOTNULL": {}, "NULL": {}, "NULLS": {}, "OF": {}, "OFFSET": {}, "ON": {},
	"OR": {}, "ORDER": {}, "OTHERS": {}, "OUTER": {}, "OVER": {}, "PARTITION": {},
	"PLAN": {}, "PRAGMA": {}, "PRECEDING": {}, "PRIMARY": {}, "QUERY": {}, "RAISE": {},
	"RANGE": {}, "RECURSIVE": {}, "REFERENCES": {}, "REGEXP": {}, "REINDEX": {}, "RELEASE": {},
	"RENAME": {}, "REPLACE": {}, "RESTRICT": {}, "RETURNING": {}, "RIGHT": {}, "ROLLBACK": {},
	"ROW": {}, "ROWS": {}, "SAVEPOINT": {}, "SELECT": {}, "SET": {}, "TABLE": {},
	"TEMP": {}, "TEMPORARY": {}, "THEN": {}, "TIES": {}, "TO": {}, "TRANSACTION": {},
	"TRIGGER": {}, "UNBOUNDED": {}, "UNION": {}, "UNIQUE": {}, "UPDATE": {}, "USING": {},
	"VACUUM": {}, "VALUES": {}, "VIEW": {}, "VIRTUAL": {}, "WHEN": {}, "WHERE": {},
	"WINDOW": {}, "WITH": {}, "WITHOUT": {},
}

// KeywordSpans implements Highlighter.
func (SQLite) KeywordSpans(value string) []Span {
	return keywordSpans(value, sqliteKeywords, skipSQLiteRegion, nil)
}

func skipSQLiteRegion(runes []rune, index int) (int, bool) {
	switch {
	case runes[index] == '\'':
		return skipQuoted(runes, index, '\'', false), true
	case runes[index] == '"' || runes[index] == '`':
		return skipQuoted(runes, index, runes[index], false), true
	case runes[index] == '[':
		return skipBracketQuoted(runes, index), true
	case index+1 < len(runes) && runes[index] == '-' && runes[index+1] == '-':
		return skipLineComment(runes, index), true
	case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
		return skipBlockComment(runes, index), true
	}
	return 0, false
}

func skipBracketQuoted(runes []rune, index int) int {
	index++
	for index < len(runes) {
		if runes[index] == ']' {
			return index + 1
		}
		index++
	}
	return len(runes)
}
