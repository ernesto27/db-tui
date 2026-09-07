package app

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ernestoponce27/db-tui/internal/app/sqlhighlight"
)

type sqlKeywordCellRange struct {
	left  int
	right int
}

func keywordCellRanges(cells []sqlEditorCell, spans []sqlhighlight.Span) []sqlKeywordCellRange {
	ranges := make([]sqlKeywordCellRange, 0, len(spans))
	for _, span := range spans {
		left, right, ok := keywordCellRange(cells, span)
		if ok {
			ranges = append(ranges, sqlKeywordCellRange{left: left, right: right})
		}
	}
	return ranges
}

func keywordCellRange(cells []sqlEditorCell, span sqlhighlight.Span) (int, int, bool) {
	column := 0
	left, right := 0, 0
	found := false
	for _, cell := range cells {
		if cell.source >= span.Start && cell.source < span.End {
			if !found {
				left = column
				found = true
			}
			right = column + cell.width
		}
		column += cell.width
	}
	return left, right, found
}

func highlightSQLKeywords(rendered string, rows [][]sqlEditorCell, value string, highlighter sqlhighlight.Highlighter) string {
	lines := strings.Split(rendered, "\n")
	spans := highlighter.KeywordSpans(value)
	keywordStyle := lipgloss.NewStyle().Foreground(colorSQLKeyword)

	for row, cells := range rows {
		if row >= len(lines) {
			break
		}
		ranges := keywordCellRanges(cells, spans)
		for index := len(ranges) - 1; index >= 0; index-- {
			keywordRange := ranges[index]
			keyword := ansi.Cut(lines[row], keywordRange.left, keywordRange.right)
			lines[row] = ansi.Cut(lines[row], 0, keywordRange.left) +
				keywordStyle.Render(ansi.Strip(keyword)) +
				ansi.Cut(lines[row], keywordRange.right, ansi.StringWidth(lines[row]))
		}
	}

	return strings.Join(lines, "\n")
}
