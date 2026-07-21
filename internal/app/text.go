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
	if value, ok := value.([]byte); ok {
		return sanitizeText(string(value))
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

func truncateLabel(label string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(sanitizeText(label), width, "…")
}
