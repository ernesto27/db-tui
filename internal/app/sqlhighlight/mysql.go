package sqlhighlight

import "unicode"

// MySQL finds the supported MySQL keyword spans in SQL source text.
type MySQL struct{}

var _ Highlighter = MySQL{}

var mySQLKeywords = map[string]struct{}{
	"ACCESSIBLE": {}, "ADD": {}, "ALL": {}, "ALTER": {}, "ANALYZE": {}, "AND": {},
	"AS": {}, "ASC": {}, "ASENSITIVE": {}, "BEFORE": {}, "BETWEEN": {}, "BIGINT": {},
	"BINARY": {}, "BLOB": {}, "BOTH": {}, "BY": {}, "CALL": {}, "CASCADE": {},
	"CASE": {}, "CHANGE": {}, "CHAR": {}, "CHARACTER": {}, "CHECK": {}, "COLLATE": {},
	"COLUMN": {}, "CONDITION": {}, "CONSTRAINT": {}, "CONTINUE": {}, "CONVERT": {}, "CREATE": {},
	"CROSS": {}, "CUBE": {}, "CUME_DIST": {}, "CURRENT_DATE": {}, "CURRENT_TIME": {}, "CURRENT_TIMESTAMP": {},
	"CURRENT_USER": {}, "CURSOR": {}, "DATABASE": {}, "DATABASES": {}, "DAY_HOUR": {}, "DAY_MICROSECOND": {},
	"DAY_MINUTE": {}, "DAY_SECOND": {}, "DEC": {}, "DECIMAL": {}, "DECLARE": {}, "DEFAULT": {},
	"DELAYED": {}, "DELETE": {}, "DENSE_RANK": {}, "DESC": {}, "DESCRIBE": {}, "DETERMINISTIC": {},
	"DISTINCT": {}, "DISTINCTROW": {}, "DIV": {}, "DOUBLE": {}, "DROP": {}, "DUAL": {},
	"EACH": {}, "ELSE": {}, "ELSEIF": {}, "EMPTY": {}, "ENCLOSED": {}, "ESCAPED": {},
	"EXCEPT": {}, "EXISTS": {}, "EXIT": {}, "EXPLAIN": {}, "FALSE": {}, "FETCH": {},
	"FIRST_VALUE": {}, "FLOAT": {}, "FLOAT4": {}, "FLOAT8": {}, "FOR": {}, "FORCE": {},
	"FOREIGN": {}, "FROM": {}, "FULLTEXT": {}, "FUNCTION": {}, "GENERATED": {}, "GET": {},
	"GRANT": {}, "GROUP": {}, "GROUPING": {}, "GROUPS": {}, "HAVING": {}, "HIGH_PRIORITY": {},
	"HOUR_MICROSECOND": {}, "HOUR_MINUTE": {}, "HOUR_SECOND": {}, "IF": {}, "IGNORE": {}, "IN": {},
	"INDEX": {}, "INFILE": {}, "INNER": {}, "INOUT": {}, "INSENSITIVE": {}, "INSERT": {},
	"INT": {}, "INT1": {}, "INT2": {}, "INT3": {}, "INT4": {}, "INT8": {},
	"INTEGER": {}, "INTERSECT": {}, "INTERVAL": {}, "INTO": {}, "IO_AFTER_GTIDS": {}, "IO_BEFORE_GTIDS": {},
	"IS": {}, "ITERATE": {}, "JOIN": {}, "JSON_TABLE": {}, "KEY": {}, "KEYS": {},
	"KILL": {}, "LAG": {}, "LAST_VALUE": {}, "LATERAL": {}, "LEAD": {}, "LEADING": {},
	"LEAVE": {}, "LEFT": {}, "LIKE": {}, "LIMIT": {}, "LINEAR": {}, "LINES": {},
	"LOAD": {}, "LOCALTIME": {}, "LOCALTIMESTAMP": {}, "LOCK": {}, "LONG": {}, "LONGBLOB": {},
	"LONGTEXT": {}, "LOOP": {}, "LOW_PRIORITY": {}, "MATCH": {}, "MAXVALUE": {}, "MEDIUMBLOB": {},
	"MEDIUMINT": {}, "MEDIUMTEXT": {}, "MIDDLEINT": {}, "MINUTE_MICROSECOND": {}, "MINUTE_SECOND": {}, "MOD": {},
	"MODIFIES": {}, "NATURAL": {}, "NOT": {}, "NO_WRITE_TO_BINLOG": {}, "NTH_VALUE": {}, "NTILE": {},
	"NULL": {}, "NUMERIC": {}, "OF": {}, "ON": {}, "OPTIMIZE": {}, "OPTIMIZER_COSTS": {},
	"OPTION": {}, "OPTIONALLY": {}, "OR": {}, "ORDER": {}, "OUT": {}, "OUTER": {},
	"OUTFILE": {}, "OVER": {}, "PARTITION": {}, "PERCENT_RANK": {}, "PRECISION": {}, "PRIMARY": {},
	"PROCEDURE": {}, "PURGE": {}, "QUALIFY": {}, "RANGE": {}, "RANK": {}, "READ": {},
	"READS": {}, "READ_WRITE": {}, "REAL": {}, "RECURSIVE": {}, "REFERENCES": {}, "REGEXP": {},
	"RELEASE": {}, "RENAME": {}, "REPEAT": {}, "REPLACE": {}, "REQUIRE": {}, "RESIGNAL": {},
	"RESTRICT": {}, "RETURN": {}, "REVOKE": {}, "RIGHT": {}, "RLIKE": {}, "ROW": {},
	"ROWS": {}, "ROW_NUMBER": {}, "SCHEMA": {}, "SCHEMAS": {}, "SECOND_MICROSECOND": {}, "SELECT": {},
	"SENSITIVE": {}, "SEPARATOR": {}, "SET": {}, "SHOW": {}, "SIGNAL": {}, "SMALLINT": {},
	"SPATIAL": {}, "SPECIFIC": {}, "SQL": {}, "SQLEXCEPTION": {}, "SQLSTATE": {}, "SQLWARNING": {},
	"SQL_BIG_RESULT": {}, "SQL_CALC_FOUND_ROWS": {}, "SQL_SMALL_RESULT": {}, "SSL": {}, "STARTING": {}, "STORED": {},
	"STRAIGHT_JOIN": {}, "SYSTEM": {}, "TABLE": {}, "TABLESAMPLE": {}, "TERMINATED": {}, "THEN": {},
	"TINYBLOB": {}, "TINYINT": {}, "TINYTEXT": {}, "TO": {}, "TRAILING": {}, "TRIGGER": {},
	"TRUE": {}, "UNDO": {}, "UNION": {}, "UNIQUE": {}, "UNLOCK": {}, "UNSIGNED": {},
	"UPDATE": {}, "USAGE": {}, "USE": {}, "USING": {}, "UTC_DATE": {}, "UTC_TIME": {},
	"UTC_TIMESTAMP": {}, "VALUES": {}, "VARBINARY": {}, "VARCHAR": {}, "VARCHARACTER": {}, "VARYING": {},
	"VIRTUAL": {}, "WHEN": {}, "WHERE": {}, "WHILE": {}, "WINDOW": {}, "WITH": {},
	"WRITE": {}, "XOR": {}, "YEAR_MONTH": {}, "ZEROFILL": {}, "_FILENAME": {},
}

// KeywordSpans implements Highlighter.
func (MySQL) KeywordSpans(value string) []Span {
	return keywordSpans(value, mySQLKeywords, skipMySQLRegion, isMySQLKeywordCandidate)
}

func isMySQLKeywordCandidate(runes []rune, start, _ int) bool {
	for index := start - 1; index >= 0; index-- {
		if unicode.IsSpace(runes[index]) {
			continue
		}
		return runes[index] != '.'
	}
	return true
}

func skipMySQLRegion(runes []rune, index int) (int, bool) {
	switch {
	case runes[index] == '\'':
		return skipQuoted(runes, index, '\'', true), true
	case runes[index] == '"' || runes[index] == '`':
		return skipQuoted(runes, index, runes[index], true), true
	case runes[index] == '#':
		return skipLineComment(runes, index), true
	case isMySQLDashCommentStart(runes, index):
		return skipLineComment(runes, index), true
	case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
		return skipBlockComment(runes, index), true
	}
	return 0, false
}

func isMySQLDashCommentStart(runes []rune, index int) bool {
	return index+2 < len(runes) && runes[index] == '-' && runes[index+1] == '-' &&
		(unicode.IsSpace(runes[index+2]) || unicode.IsControl(runes[index+2]))
}
