package sqlhighlight

import "unicode"

// Oracle finds the supported Oracle keyword spans in SQL source text.
type Oracle struct{}

var _ Highlighter = Oracle{}

var oracleKeywords = map[string]struct{}{
	"ACCESS": {}, "ADD": {}, "ALL": {}, "ALTER": {}, "AND": {}, "ANY": {},
	"AS": {}, "ASC": {}, "AUDIT": {}, "BETWEEN": {}, "BY": {}, "CHAR": {},
	"CHECK": {}, "CLUSTER": {}, "COLUMN": {}, "COLUMN_VALUE": {}, "COMMENT": {},
	"COMPRESS": {}, "CONNECT": {}, "CREATE": {}, "CURRENT": {}, "DATE": {},
	"DECIMAL": {}, "DEFAULT": {}, "DELETE": {}, "DESC": {}, "DISTINCT": {},
	"DROP": {}, "ELSE": {}, "EXCLUSIVE": {}, "EXISTS": {}, "FILE": {}, "FLOAT": {},
	"FOR": {}, "FROM": {}, "GRANT": {}, "GROUP": {}, "HAVING": {}, "IDENTIFIED": {},
	"IMMEDIATE": {}, "IN": {}, "INCREMENT": {}, "INDEX": {}, "INITIAL": {},
	"INSERT": {}, "INTEGER": {}, "INTERSECT": {}, "INTO": {}, "IS": {}, "LEVEL": {},
	"LIKE": {}, "LOCK": {}, "LONG": {}, "MAXEXTENTS": {}, "MINUS": {}, "MLSLABEL": {},
	"MODE": {}, "MODIFY": {}, "NESTED_TABLE_ID": {}, "NOAUDIT": {}, "NOCOMPRESS": {},
	"NOT": {}, "NOWAIT": {}, "NULL": {}, "NUMBER": {}, "OF": {}, "OFFLINE": {},
	"ON": {}, "ONLINE": {}, "OPTION": {}, "OR": {}, "ORDER": {}, "PCTFREE": {},
	"PRIOR": {}, "PUBLIC": {}, "RAW": {}, "RENAME": {}, "RESOURCE": {}, "REVOKE": {},
	"ROW": {}, "ROWID": {}, "ROWNUM": {}, "ROWS": {}, "SELECT": {}, "SESSION": {},
	"SET": {}, "SHARE": {}, "SIZE": {}, "SMALLINT": {}, "SUCCESSFUL": {},
	"SYNONYM": {}, "SYSDATE": {}, "TABLE": {}, "THEN": {}, "TO": {}, "TRIGGER": {},
	"UID": {}, "UNION": {}, "UNIQUE": {}, "UPDATE": {}, "USER": {}, "VALIDATE": {},
	"VALUES": {}, "VARCHAR": {}, "VARCHAR2": {}, "VIEW": {}, "WHENEVER": {}, "WHERE": {},
	"WITH": {},

	"BEGIN": {}, "BINARY_DOUBLE": {}, "BINARY_FLOAT": {}, "BLOB": {}, "BODY": {},
	"BULK": {}, "CASE": {}, "CAST": {}, "CLOB": {}, "COLLATE": {}, "COMMIT": {},
	"CONSTRAINT": {}, "CONTINUE": {}, "CROSS": {}, "CURSOR": {}, "DECLARE": {},
	"DETERMINISTIC": {}, "DOUBLE": {}, "EACH": {}, "END": {}, "ESCAPE": {},
	"EXCEPTION": {}, "EXECUTE": {}, "FETCH": {}, "FIRST": {}, "FOREIGN": {}, "FULL": {},
	"FUNCTION": {}, "GOTO": {}, "IF": {}, "INNER": {}, "INTERVAL": {}, "JOIN": {},
	"JSON": {}, "LEFT": {}, "LOOP": {}, "MATERIALIZED": {}, "MERGE": {}, "NATURAL": {},
	"NEXT": {}, "NOCYCLE": {}, "OFFSET": {}, "ONLY": {}, "OPEN": {}, "OUTER": {},
	"PACKAGE": {}, "PARTITION": {}, "PIPELINED": {}, "PRECISION": {}, "PROCEDURE": {},
	"RANGE": {}, "RECURSIVE": {}, "REFERENCES": {}, "RETURN": {}, "RETURNING": {},
	"RIGHT": {}, "ROLLBACK": {}, "SAVEPOINT": {}, "SECOND": {}, "SIBLINGS": {},
	"START": {}, "SUBPARTITION": {}, "TIMESTAMP": {}, "TRUE": {}, "TYPE": {},
	"UNBOUNDED": {}, "USING": {}, "VARRAY": {}, "WHEN": {}, "WHILE": {}, "WINDOW": {},
}

// KeywordSpans implements Highlighter.
func (Oracle) KeywordSpans(value string) []Span {
	return keywordSpans(value, oracleKeywords, skipOracleRegion, nil)
}

func skipOracleRegion(runes []rune, index int) (int, bool) {
	switch {
	case isOracleAlternativeQuoteStart(runes, index):
		return skipOracleAlternativeQuoted(runes, index), true
	case runes[index] == '\'':
		return skipQuoted(runes, index, '\'', false), true
	case runes[index] == '"':
		return skipQuoted(runes, index, '"', false), true
	case index+1 < len(runes) && runes[index] == '-' && runes[index+1] == '-':
		return skipLineComment(runes, index), true
	case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
		return skipBlockComment(runes, index), true
	}
	return 0, false
}

func isOracleAlternativeQuoteStart(runes []rune, index int) bool {
	if index > 0 && isIdentifierContinuationRune(runes[index-1]) {
		return false
	}
	if index+2 < len(runes) && (runes[index] == 'q' || runes[index] == 'Q') {
		return runes[index+1] == '\'' && isOracleQuoteDelimiter(runes[index+2])
	}
	return index+3 < len(runes) && (runes[index] == 'n' || runes[index] == 'N') &&
		(runes[index+1] == 'q' || runes[index+1] == 'Q') && runes[index+2] == '\'' &&
		isOracleQuoteDelimiter(runes[index+3])
}

func isOracleQuoteDelimiter(r rune) bool {
	return !unicode.IsSpace(r) && !unicode.IsControl(r)
}

func skipOracleAlternativeQuoted(runes []rune, index int) int {
	if runes[index] == 'n' || runes[index] == 'N' {
		index++
	}
	opening := runes[index+2]
	closing := oracleClosingQuoteDelimiter(opening)
	for index += 3; index+1 < len(runes); index++ {
		if runes[index] == closing && runes[index+1] == '\'' {
			return index + 2
		}
	}
	return len(runes)
}

func oracleClosingQuoteDelimiter(opening rune) rune {
	switch opening {
	case '[':
		return ']'
	case '{':
		return '}'
	case '(':
		return ')'
	case '<':
		return '>'
	default:
		return opening
	}
}
