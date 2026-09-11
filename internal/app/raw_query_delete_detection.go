package app

import "strings"

type rawQueryTokenKind uint8

const (
	rawQueryTokenWord rawQueryTokenKind = iota
	rawQueryTokenIdentifier
	rawQueryTokenOpenParen
	rawQueryTokenCloseParen
	rawQueryTokenComma
	rawQueryTokenSemicolon
)

type rawQueryToken struct {
	kind rawQueryTokenKind
	text string
}

func rawQueryContainsDelete(sql string) bool {
	tokens := rawQueryTokens(sql)
	statementStart := 0
	depth := 0

	for index, token := range tokens {
		switch token.kind {
		case rawQueryTokenOpenParen:
			depth++
		case rawQueryTokenCloseParen:
			if depth > 0 {
				depth--
			}
		case rawQueryTokenSemicolon:
			if depth == 0 {
				if rawQueryStatementContainsDelete(tokens[statementStart:index]) {
					return true
				}
				statementStart = index + 1
			}
		}
	}

	return rawQueryStatementContainsDelete(tokens[statementStart:])
}

func rawQueryStatementContainsDelete(tokens []rawQueryToken) bool {
	if len(tokens) == 0 || tokens[0].kind != rawQueryTokenWord {
		return false
	}

	switch tokens[0].text {
	case "delete":
		return true
	case "with":
		return rawQueryWithContainsDelete(tokens)
	default:
		return false
	}
}

func rawQueryWithContainsDelete(tokens []rawQueryToken) bool {
	index := 1
	if index < len(tokens) && tokens[index].kind == rawQueryTokenWord && tokens[index].text == "recursive" {
		index++
	}

	for {
		if index >= len(tokens) || !rawQueryIsIdentifier(tokens[index]) {
			return false
		}
		index++

		for index < len(tokens) && !(tokens[index].kind == rawQueryTokenWord && tokens[index].text == "as") {
			if tokens[index].kind == rawQueryTokenSemicolon {
				return false
			}
			index++
		}
		if index >= len(tokens) {
			return false
		}
		index++

		for index < len(tokens) && tokens[index].kind != rawQueryTokenOpenParen {
			if tokens[index].kind == rawQueryTokenSemicolon {
				return false
			}
			index++
		}
		if index >= len(tokens) {
			return false
		}

		bodyStart := index + 1
		bodyEnd, ok := rawQueryClosingParen(tokens, index)
		if !ok {
			return false
		}
		if rawQueryStatementContainsDelete(tokens[bodyStart:bodyEnd]) {
			return true
		}
		index = bodyEnd + 1

		if index < len(tokens) && tokens[index].kind == rawQueryTokenComma {
			index++
			continue
		}
		return rawQueryStatementContainsDelete(tokens[index:])
	}
}

func rawQueryClosingParen(tokens []rawQueryToken, start int) (int, bool) {
	depth := 0
	for index := start; index < len(tokens); index++ {
		switch tokens[index].kind {
		case rawQueryTokenOpenParen:
			depth++
		case rawQueryTokenCloseParen:
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}
	return 0, false
}

func rawQueryIsIdentifier(token rawQueryToken) bool {
	return token.kind == rawQueryTokenWord || token.kind == rawQueryTokenIdentifier
}

func rawQueryTokens(sql string) []rawQueryToken {
	var tokens []rawQueryToken
	for index := 0; index < len(sql); {
		switch {
		case rawQueryWhitespace(sql[index]):
			index++
		case sql[index] == '-' && index+1 < len(sql) && sql[index+1] == '-':
			index = rawQueryLineCommentEnd(sql, index+2)
		case sql[index] == '#':
			index = rawQueryLineCommentEnd(sql, index+1)
		case sql[index] == '/' && index+1 < len(sql) && sql[index+1] == '*':
			index = rawQueryBlockCommentEnd(sql, index+2)
		case sql[index] == '\'':
			index = rawQueryQuotedEnd(sql, index, '\'')
		case sql[index] == '"':
			tokens = append(tokens, rawQueryToken{kind: rawQueryTokenIdentifier})
			index = rawQueryQuotedEnd(sql, index, '"')
		case sql[index] == '`':
			tokens = append(tokens, rawQueryToken{kind: rawQueryTokenIdentifier})
			index = rawQueryQuotedEnd(sql, index, '`')
		case sql[index] == '[':
			tokens = append(tokens, rawQueryToken{kind: rawQueryTokenIdentifier})
			index = rawQueryBracketIdentifierEnd(sql, index)
		case sql[index] == '$':
			if end, ok := rawQueryDollarQuoteEnd(sql, index); ok {
				index = end
			} else {
				start := index
				index++
				for index < len(sql) && rawQueryWordByte(sql[index]) {
					index++
				}
				tokens = append(tokens, rawQueryToken{kind: rawQueryTokenWord, text: strings.ToLower(sql[start:index])})
			}
		case rawQueryWordByte(sql[index]):
			start := index
			for index < len(sql) && rawQueryWordByte(sql[index]) {
				index++
			}
			tokens = append(tokens, rawQueryToken{kind: rawQueryTokenWord, text: strings.ToLower(sql[start:index])})
		default:
			switch sql[index] {
			case '(':
				tokens = append(tokens, rawQueryToken{kind: rawQueryTokenOpenParen})
			case ')':
				tokens = append(tokens, rawQueryToken{kind: rawQueryTokenCloseParen})
			case ',':
				tokens = append(tokens, rawQueryToken{kind: rawQueryTokenComma})
			case ';':
				tokens = append(tokens, rawQueryToken{kind: rawQueryTokenSemicolon})
			}
			index++
		}
	}
	return tokens
}

func rawQueryWhitespace(value byte) bool {
	return value == ' ' || value == '\n' || value == '\r' || value == '\t' || value == '\f'
}

func rawQueryWordByte(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' ||
		value == '_' || value == '$'
}

func rawQueryLineCommentEnd(sql string, index int) int {
	for index < len(sql) && sql[index] != '\n' {
		index++
	}
	return index
}

func rawQueryBlockCommentEnd(sql string, index int) int {
	for index+1 < len(sql) {
		if sql[index] == '*' && sql[index+1] == '/' {
			return index + 2
		}
		index++
	}
	return len(sql)
}

func rawQueryQuotedEnd(sql string, start int, quote byte) int {
	for index := start + 1; index < len(sql); index++ {
		if sql[index] == '\\' && index+1 < len(sql) {
			index++
			continue
		}
		if sql[index] != quote {
			continue
		}
		if index+1 < len(sql) && sql[index+1] == quote {
			index++
			continue
		}
		return index + 1
	}
	return len(sql)
}

func rawQueryBracketIdentifierEnd(sql string, start int) int {
	for index := start + 1; index < len(sql); index++ {
		if sql[index] != ']' {
			continue
		}
		if index+1 < len(sql) && sql[index+1] == ']' {
			index++
			continue
		}
		return index + 1
	}
	return len(sql)
}

func rawQueryDollarQuoteEnd(sql string, start int) (int, bool) {
	end := start + 1
	for end < len(sql) && (sql[end] == '_' || sql[end] >= 'a' && sql[end] <= 'z' || sql[end] >= 'A' && sql[end] <= 'Z' || sql[end] >= '0' && sql[end] <= '9') {
		end++
	}
	if end >= len(sql) || sql[end] != '$' {
		return 0, false
	}

	delimiter := sql[start : end+1]
	closing := strings.Index(sql[end+1:], delimiter)
	if closing < 0 {
		return len(sql), true
	}
	return end + 1 + closing + len(delimiter), true
}
