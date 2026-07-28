package app

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

func formatCell(value any) string {
	if value == nil {
		return "NULL"
	}
	if _, ok := value.([]byte); ok {
		return "BINARY DATA"
	}
	return sanitizeText(fmt.Sprint(value))
}

func sanitizeText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '�'
		}
		return r
	}, text)
}

func sanitizeMultilineText(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return '�'
	}, text)
}

func truncateLabel(label string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(sanitizeText(label), width, "…")
}
