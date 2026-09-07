package sqlhighlight

import "unicode"

// SQLServer finds the supported SQL Server keyword spans in SQL source text.
type SQLServer struct{}

var _ Highlighter = SQLServer{}

var sqlServerKeywords = map[string]struct{}{
	"ADD": {}, "ALL": {}, "ALTER": {}, "AND": {}, "ANY": {}, "AS": {}, "ASC": {},
	"AUTHORIZATION": {}, "BACKUP": {}, "BEGIN": {}, "BETWEEN": {}, "BREAK": {}, "BROWSE": {},
	"BULK": {}, "BY": {}, "CASCADE": {}, "CASE": {}, "CHECK": {}, "CHECKPOINT": {},
	"CLOSE": {}, "CLUSTERED": {}, "COALESCE": {}, "COLLATE": {}, "COLUMN": {}, "COMMIT": {},
	"COMPUTE": {}, "CONSTRAINT": {}, "CONTAINS": {}, "CONTAINSTABLE": {}, "CONTINUE": {},
	"CONVERT": {}, "CREATE": {}, "CROSS": {}, "CURRENT": {}, "CURRENT_DATE": {},
	"CURRENT_TIME": {}, "CURRENT_TIMESTAMP": {}, "CURRENT_USER": {}, "CURSOR": {}, "DATABASE": {},
	"DBCC": {}, "DEALLOCATE": {}, "DECLARE": {}, "DEFAULT": {}, "DELETE": {}, "DENY": {},
	"DESC": {}, "DISK": {}, "DISTINCT": {}, "DISTRIBUTED": {}, "DOUBLE": {}, "DROP": {},
	"DUMP": {}, "ELSE": {}, "END": {}, "ERRLVL": {}, "ESCAPE": {}, "EXCEPT": {},
	"EXEC": {}, "EXECUTE": {}, "EXISTS": {}, "EXIT": {}, "EXTERNAL": {}, "FETCH": {},
	"FILE": {}, "FILLFACTOR": {}, "FOR": {}, "FOREIGN": {}, "FREETEXT": {}, "FREETEXTTABLE": {},
	"FROM": {}, "FULL": {}, "FUNCTION": {}, "GOTO": {}, "GRANT": {}, "GROUP": {}, "HAVING": {},
	"HOLDLOCK": {}, "IDENTITY": {}, "IDENTITY_INSERT": {}, "IDENTITYCOL": {}, "IF": {},
	"IN": {}, "INDEX": {}, "INNER": {}, "INSERT": {}, "INTERSECT": {}, "INTO": {}, "IS": {},
	"JOIN": {}, "KEY": {}, "KILL": {}, "LEFT": {}, "LIKE": {}, "LINENO": {}, "LOAD": {},
	"MERGE": {}, "NATIONAL": {}, "NOCHECK": {}, "NONCLUSTERED": {}, "NOT": {}, "NULL": {},
	"NULLIF": {}, "OF": {}, "OFF": {}, "OFFSETS": {}, "ON": {}, "OPEN": {},
	"OPENDATASOURCE": {}, "OPENQUERY": {}, "OPENROWSET": {}, "OPENXML": {}, "OPTION": {},
	"OR": {}, "ORDER": {}, "OUTER": {}, "OVER": {}, "PERCENT": {}, "PIVOT": {}, "PLAN": {},
	"PRECISION": {}, "PRIMARY": {}, "PRINT": {}, "PROC": {}, "PROCEDURE": {}, "PUBLIC": {},
	"RAISERROR": {}, "READ": {}, "READTEXT": {}, "RECONFIGURE": {}, "REFERENCES": {},
	"REPLICATION": {}, "RESTORE": {}, "RESTRICT": {}, "RETURN": {}, "REVERT": {}, "REVOKE": {},
	"RIGHT": {}, "ROLLBACK": {}, "ROWCOUNT": {}, "ROWGUIDCOL": {}, "RULE": {}, "SAVE": {},
	"SCHEMA": {}, "SECURITYAUDIT": {}, "SELECT": {}, "SEMANTICKEYPHRASETABLE": {},
	"SEMANTICSIMILARITYDETAILSTABLE": {}, "SEMANTICSIMILARITYTABLE": {}, "SESSION_USER": {},
	"SET": {}, "SETUSER": {}, "SHUTDOWN": {}, "SOME": {}, "STATISTICS": {}, "SYSTEM_USER": {},
	"TABLE": {}, "TABLESAMPLE": {}, "TEXTSIZE": {}, "THEN": {}, "TO": {}, "TOP": {},
	"TRAN": {}, "TRANSACTION": {}, "TRIGGER": {}, "TRUNCATE": {}, "TRY_CONVERT": {},
	"TSEQUAL": {}, "UNION": {}, "UNIQUE": {}, "UNPIVOT": {}, "UPDATE": {}, "UPDATETEXT": {},
	"USE": {}, "USER": {}, "VALUES": {}, "VARYING": {}, "VIEW": {}, "WAITFOR": {}, "WHEN": {},
	"WHERE": {}, "WHILE": {}, "WITH": {}, "WITHIN": {}, "WRITETEXT": {},
}

// KeywordSpans implements Highlighter.
func (SQLServer) KeywordSpans(value string) []Span {
	return keywordSpans(value, sqlServerKeywords, skipSQLServerRegion, isSQLServerKeywordCandidate)
}

func isSQLServerKeywordCandidate(runes []rune, start, _ int) bool {
	for index := start - 1; index >= 0; index-- {
		if unicode.IsSpace(runes[index]) {
			continue
		}
		return runes[index] != '.' && runes[index] != '@'
	}
	return true
}

func skipSQLServerRegion(runes []rune, index int) (int, bool) {
	switch {
	case runes[index] == '\'':
		return skipQuoted(runes, index, '\'', false), true
	case runes[index] == '"':
		return skipQuoted(runes, index, '"', false), true
	case runes[index] == '[':
		return skipSQLServerBracketQuoted(runes, index), true
	case index+1 < len(runes) && runes[index] == '-' && runes[index+1] == '-':
		return skipLineComment(runes, index), true
	case index+1 < len(runes) && runes[index] == '/' && runes[index+1] == '*':
		return skipNestedBlockComment(runes, index), true
	}
	return 0, false
}

func skipSQLServerBracketQuoted(runes []rune, index int) int {
	index++
	for index < len(runes) {
		if runes[index] != ']' {
			index++
			continue
		}
		if index+1 < len(runes) && runes[index+1] == ']' {
			index += 2
			continue
		}
		return index + 1
	}
	return len(runes)
}
