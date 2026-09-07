package sqlhighlight

import (
	"strings"
	"unicode"
)

type lexicalSkipper func([]rune, int) (int, bool)
type keywordCandidate func([]rune, int, int) bool

func keywordSpans(value string, keywords map[string]struct{}, skip lexicalSkipper, candidate keywordCandidate) []Span {
	runes := []rune(value)
	var spans []Span

	for index := 0; index < len(runes); {
		if skip != nil {
			if next, ok := skip(runes, index); ok {
				index = next
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
		if _, ok := keywords[strings.ToUpper(string(runes[start:index]))]; ok &&
			(candidate == nil || candidate(runes, start, index)) {
			spans = append(spans, Span{Start: start, End: index})
		}
	}

	return spans
}

func isIdentifierContinuationRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func skipQuoted(runes []rune, index int, quote rune, backslashEscapes bool) int {
	index++
	for index < len(runes) {
		if backslashEscapes && runes[index] == '\\' && index+1 < len(runes) {
			index += 2
			continue
		}
		if runes[index] != quote {
			index++
			continue
		}
		if index+1 < len(runes) && runes[index+1] == quote {
			index += 2
			continue
		}
		return index + 1
	}
	return len(runes)
}

func skipLineComment(runes []rune, index int) int {
	for index < len(runes) && runes[index] != '\n' {
		index++
	}
	return index
}

func skipBlockComment(runes []rune, index int) int {
	index += 2
	for index+1 < len(runes) {
		if runes[index] == '*' && runes[index+1] == '/' {
			return index + 2
		}
		index++
	}
	return len(runes)
}

func skipNestedBlockComment(runes []rune, index int) int {
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
