package app

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"

	"github.com/ernestoponce27/db-tui/internal/app/textselection"
)

type sqlSelection struct {
	drag   textselection.Selection
	anchor int
	head   int
	active bool
}

type sqlEditorCell struct {
	source int
	rune   rune
	width  int
}

func (s *sqlSelection) start(point textselection.Point, region textselection.Region, index int) bool {
	if !s.drag.Start(point, region) {
		return false
	}
	s.anchor = index
	s.head = index
	s.active = false
	return true
}

func (s *sqlSelection) update(point textselection.Point, index int) bool {
	if !s.drag.Update(point) {
		return false
	}
	s.head = index
	s.active = s.head != s.anchor
	return true
}

func (s *sqlSelection) release(point textselection.Point, index int) bool {
	if !s.drag.Release(point) {
		return false
	}
	s.head = index
	s.active = s.head != s.anchor
	return true
}

func (s *sqlSelection) clear() {
	s.drag.Clear()
	s.anchor = 0
	s.head = 0
	s.active = false
}

func (s sqlSelection) dragging() bool {
	return s.drag.Dragging()
}

func (s sqlSelection) selectedSQL(value string) (string, bool) {
	if !s.active {
		return "", false
	}
	runes := []rune(value)
	start, end := s.anchor, s.head
	if start > end {
		start, end = end, start
	}
	if start < 0 || end >= len(runes) {
		return "", false
	}
	return string(runes[start : end+1]), true
}

func (m queryModel) executableSQL() string {
	if selected, ok := m.selection.selectedSQL(m.editor.Value()); ok {
		return selected
	}
	return m.editor.Value()
}

func (m *queryModel) beginSelection(x, y int, layout appLayout) bool {
	point, index, ok := m.selectionPointAndIndexAt(x, y, layout, false)
	if !ok {
		return false
	}
	return m.selection.start(point, textselection.Region{
		Left: 0, Right: m.editor.Width() - 1,
		Top: 0, Bottom: m.editor.Height() - 1,
	}, index)
}

func (m *queryModel) extendSelection(x, y int, layout appLayout) bool {
	point, index, ok := m.selectionPointAndIndexAt(x, y, layout, true)
	if !ok {
		return false
	}
	return m.selection.update(point, index)
}

func (m *queryModel) finishSelection(x, y int, layout appLayout) bool {
	point, index, ok := m.selectionPointAndIndexAt(x, y, layout, true)
	if !ok {
		m.selection.clear()
		return false
	}
	return m.selection.release(point, index) && m.selection.active
}

func (m queryModel) editorView(layout appLayout) string {
	rendered := m.editor.View()
	if !m.selection.active {
		return rendered
	}

	selectedStart, selectedEnd := m.selection.anchor, m.selection.head
	if selectedStart > selectedEnd {
		selectedStart, selectedEnd = selectedEnd, selectedStart
	}
	rows := m.visibleEditorRows()
	lines := strings.Split(rendered, "\n")
	for row, cells := range rows {
		if row >= len(lines) {
			break
		}
		left, right, ok := selectedCellRange(cells, selectedStart, selectedEnd)
		if !ok {
			continue
		}
		lines[row] = ansi.Cut(lines[row], 0, left) +
			textSelectionStyle.Render(ansi.Strip(ansi.Cut(lines[row], left, right))) +
			ansi.Cut(lines[row], right, ansi.StringWidth(lines[row]))
	}
	return strings.Join(lines, "\n")
}

func (m queryModel) selectionPointAndIndexAt(x, y int, layout appLayout, clamp bool) (textselection.Point, int, bool) {
	left, top := queryEditorOrigin(layout)
	width, height := m.editor.Width(), m.editor.Height()
	if width < 1 || height < 1 || x < left || x >= left+width || y < top || y >= top+height {
		if !clamp {
			return textselection.Point{}, 0, false
		}
		x = min(max(x, left), left+width-1)
		y = min(max(y, top), top+height-1)
	}
	point := textselection.Point{X: x - left, Y: y - top}

	rows := m.visibleEditorRows()
	row := point.Y
	if row < len(rows) {
		if index, ok := sqlEditorCellAt(rows[row], point.X); ok {
			return point, index, true
		}
	}
	if !clamp {
		return textselection.Point{}, 0, false
	}
	index, ok := nearestEditorCell(rows, row, point.X)
	return point, index, ok
}

func (m queryModel) visibleEditorRows() [][]sqlEditorCell {
	rows := sqlEditorRows([]rune(m.editor.Value()), m.editor.Width())
	offset := min(max(m.editor.ScrollYOffset(), 0), len(rows))
	end := min(offset+m.editor.Height(), len(rows))
	return rows[offset:end]
}

func queryEditorOrigin(layout appLayout) (int, int) {
	return layout.data.x + 2, layout.data.y + 2 // border, padding, and heading
}

func sqlEditorRows(runes []rune, width int) [][]sqlEditorCell {
	if width < 1 {
		return nil
	}
	rows := make([][]sqlEditorCell, 0, len(runes)+1)
	lineStart := 0
	for index := 0; index <= len(runes); index++ {
		if index != len(runes) && runes[index] != '\n' {
			continue
		}
		line := make([]sqlEditorCell, index-lineStart)
		for offset, r := range runes[lineStart:index] {
			line[offset] = sqlEditorCell{source: lineStart + offset, rune: r, width: ansi.StringWidth(string(r))}
		}
		rows = append(rows, wrapSQLEditorLine(line, width)...)
		lineStart = index + 1
	}
	return rows
}

// wrapSQLEditorLine mirrors bubbles/textarea's wrapping so terminal positions
// resolve back to the editor's logical rune indexes.
func wrapSQLEditorLine(line []sqlEditorCell, width int) [][]sqlEditorCell {
	rows := [][]sqlEditorCell{{}}
	word := make([]sqlEditorCell, 0)
	spaces := make([]sqlEditorCell, 0)

	for _, cell := range line {
		if unicode.IsSpace(cell.rune) {
			spaces = append(spaces, cell)
		} else {
			word = append(word, cell)
		}

		if len(spaces) > 0 {
			if editorCellsWidth(rows[len(rows)-1])+editorCellsWidth(word)+len(spaces) > width {
				rows = append(rows, append(append([]sqlEditorCell{}, word...), spaces...))
			} else {
				rows[len(rows)-1] = append(rows[len(rows)-1], word...)
				rows[len(rows)-1] = append(rows[len(rows)-1], spaces...)
			}
			word = nil
			spaces = nil
			continue
		}

		lastWidth := word[len(word)-1].width
		if editorCellsWidth(word)+lastWidth > width {
			if len(rows[len(rows)-1]) > 0 {
				rows = append(rows, []sqlEditorCell{})
			}
			rows[len(rows)-1] = append(rows[len(rows)-1], word...)
			word = nil
		}
	}

	if editorCellsWidth(rows[len(rows)-1])+editorCellsWidth(word)+len(spaces) >= width {
		rows = append(rows, append(append([]sqlEditorCell{}, word...), spaces...))
		rows[len(rows)-1] = append(rows[len(rows)-1], sqlEditorCell{source: -1, rune: ' ', width: 1})
	} else {
		rows[len(rows)-1] = append(rows[len(rows)-1], word...)
		rows[len(rows)-1] = append(rows[len(rows)-1], spaces...)
		rows[len(rows)-1] = append(rows[len(rows)-1], sqlEditorCell{source: -1, rune: ' ', width: 1})
	}
	return rows
}

func editorCellsWidth(cells []sqlEditorCell) int {
	width := 0
	for _, cell := range cells {
		width += cell.width
	}
	return width
}

func sqlEditorCellAt(cells []sqlEditorCell, column int) (int, bool) {
	position := 0
	for _, cell := range cells {
		if cell.source >= 0 && column >= position && column < position+max(1, cell.width) {
			return cell.source, true
		}
		position += cell.width
	}
	for index := len(cells) - 1; index >= 0; index-- {
		if cells[index].source >= 0 {
			return cells[index].source, true
		}
	}
	return 0, false
}

func nearestEditorCell(rows [][]sqlEditorCell, row, column int) (int, bool) {
	maxDistance := max(row, len(rows)-1-row)
	for distance := 0; distance <= maxDistance; distance++ {
		for _, candidate := range []int{row - distance, row + distance} {
			if candidate < 0 || candidate >= len(rows) {
				continue
			}
			if index, ok := sqlEditorCellAt(rows[candidate], column); ok {
				return index, true
			}
		}
	}
	return 0, false
}

func selectedCellRange(cells []sqlEditorCell, start, end int) (int, int, bool) {
	column := 0
	left, right := 0, 0
	found := false
	for _, cell := range cells {
		if cell.source >= start && cell.source <= end {
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

func (m *queryModel) clearSelectionBeforeEditorUpdate() {
	m.selection.clear()
}
