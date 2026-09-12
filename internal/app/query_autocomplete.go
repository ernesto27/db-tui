package app

import (
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ernestoponce27/db-tui/internal/app/sqlhighlight"
	"github.com/ernestoponce27/db-tui/internal/db"
)

const tableCompletionMaxMatches = 5

type tableCompletionModel struct {
	matches  []db.Table
	selected int
	start    int
	end      int
	visible  bool
}

func (m *tableCompletionModel) dismiss() {
	*m = tableCompletionModel{}
}

func (m *queryModel) refreshTableCompletion(tables []db.Table) {
	start, end, prefix, ok := tableCompletionPrefix(m.editor)
	if !ok {
		m.completion.dismiss()
		return
	}

	matches := matchingTables(tables, prefix)
	if len(matches) == 0 {
		m.completion.dismiss()
		return
	}

	m.completion = tableCompletionModel{matches: matches, start: start, end: end, visible: true}
}

func (m *queryModel) handleTableCompletionKey(msg tea.KeyPressMsg) bool {
	if !m.completion.visible {
		return false
	}

	switch msg.String() {
	case "esc":
		m.completion.dismiss()
	case "up":
		m.completion.selected = max(0, m.completion.selected-1)
	case "down":
		m.completion.selected = min(len(m.completion.matches)-1, m.completion.selected+1)
	case "tab", "enter":
		m.acceptTableCompletion()
	default:
		return false
	}
	return true
}

func (m *queryModel) acceptTableCompletion() {
	completion := m.completion
	if !completion.visible || completion.selected < 0 || completion.selected >= len(completion.matches) {
		return
	}

	value := []rune(m.editor.Value())
	replacement := []rune(quotePostgreSQLIdentifier(completion.matches[completion.selected].Name))
	updated := make([]rune, 0, len(value)-completion.end+completion.start+len(replacement))
	updated = append(updated, value[:completion.start]...)
	updated = append(updated, replacement...)
	updated = append(updated, value[completion.end:]...)

	m.editor.SetValue(string(updated))
	m.restoreEditorCursorOffset(completion.start + len(replacement))
	m.completion.dismiss()
}

func (m *queryModel) restoreEditorCursorOffset(offset int) {
	lines := strings.Split(m.editor.Value(), "\n")
	line, column := 0, max(0, offset)
	for line < len(lines)-1 && column > len([]rune(lines[line])) {
		column -= len([]rune(lines[line])) + 1
		line++
	}

	m.editor.CursorStart()
	for m.editor.Line() > 0 {
		m.editor.CursorUp()
	}
	for m.editor.Line() < line {
		previous := m.editor.Line()
		m.editor.CursorDown()
		if m.editor.Line() == previous {
			break
		}
	}
	m.editor.SetCursorColumn(max(0, column))
}

func tableCompletionPrefix(editor textarea.Model) (start, end int, prefix string, ok bool) {
	runes := []rune(editor.Value())
	end = editorCursorOffset(editor)
	if end <= 0 || end > len(runes) {
		return 0, 0, "", false
	}
	cursor := end

	start = end
	for start > 0 && isTableCompletionIdentifierRune(runes[start-1]) {
		start--
	}
	if start == cursor {
		return 0, 0, "", false
	}
	for end < len(runes) && isTableCompletionIdentifierRune(runes[end]) {
		end++
	}
	if !tableCompletionCodePosition(runes, start) || !tableCompletionKeywordBefore(runes, start) {
		return 0, 0, "", false
	}
	return start, end, string(runes[start:cursor]), true
}

func editorCursorOffset(editor textarea.Model) int {
	lines := strings.Split(editor.Value(), "\n")
	line := min(max(editor.Line(), 0), len(lines)-1)
	offset := 0
	for _, value := range lines[:line] {
		offset += len([]rune(value)) + 1
	}
	return offset + editor.Column()
}

func tableCompletionKeywordBefore(runes []rune, start int) bool {
	spans := (sqlhighlight.PostgreSQL{}).KeywordSpans(string(runes[:start]))
	if len(spans) == 0 {
		return false
	}

	last := spans[len(spans)-1]
	if strings.TrimSpace(string(runes[last.End:start])) != "" {
		return false
	}
	switch strings.ToUpper(string(runes[last.Start:last.End])) {
	case "FROM", "JOIN", "UPDATE", "INTO", "TRUNCATE":
		return true
	default:
		return false
	}
}

func tableCompletionCodePosition(runes []rune, position int) bool {
	for index := 0; index < position; {
		switch {
		case runes[index] == '\'':
			next := quotedCompletionEnd(runes, index, '\'')
			if next > position {
				return false
			}
			index = next
		case runes[index] == '"':
			next := quotedCompletionEnd(runes, index, '"')
			if next > position {
				return false
			}
			index = next
		case index+1 < position && runes[index] == '-' && runes[index+1] == '-':
			for index < position && runes[index] != '\n' {
				index++
			}
			if index == position {
				return false
			}
		case index+1 < position && runes[index] == '/' && runes[index+1] == '*':
			index += 2
			for index+1 < position && !(runes[index] == '*' && runes[index+1] == '/') {
				index++
			}
			if index+1 >= position {
				return false
			}
			index += 2
		default:
			index++
		}
	}
	return true
}

func quotedCompletionEnd(runes []rune, index int, quote rune) int {
	index++
	for index < len(runes) {
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

func isTableCompletionIdentifierRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func matchingTables(tables []db.Table, prefix string) []db.Table {
	matches := make([]db.Table, 0, min(len(tables), tableCompletionMaxMatches))
	prefix = strings.ToLower(prefix)
	for _, table := range tables {
		if !strings.HasPrefix(strings.ToLower(table.Name), prefix) {
			continue
		}
		matches = append(matches, table)
		if len(matches) == tableCompletionMaxMatches {
			break
		}
	}
	return matches
}

func quotePostgreSQLIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func (m queryModel) completionOverlay(editorView string) string {
	if !m.completion.visible {
		return editorView
	}

	lines := strings.Split(editorView, "\n")
	row, column, ok := m.completionPosition()
	if !ok || len(lines) < 2 {
		return editorView
	}

	above, below := row, len(lines)-row-1
	count := min(len(m.completion.matches), max(above, below))
	if count == 0 {
		return editorView
	}
	start := row + 1
	if above > below {
		start = row - count
	}
	first := min(max(m.completion.selected-count/2, 0), len(m.completion.matches)-count)
	width := max(1, m.editor.Width()-column)
	for index := 0; index < count; index++ {
		marker := "  "
		style := lipgloss.NewStyle().Width(width).Background(colorInputBackground)
		match := first + index
		if match == m.completion.selected {
			marker = "> "
			style = style.Foreground(colorSelectionForeground).Background(colorSelectionBackground)
		}
		lines[start+index] = ansi.Cut(lines[start+index], 0, column) +
			style.Render(truncateLabel(marker+sanitizeText(m.completion.matches[match].Name), max(1, width-len(marker))))
	}
	return strings.Join(lines, "\n")
}

func (m queryModel) completionPosition() (int, int, bool) {
	for row, cells := range m.visibleEditorRows() {
		column := 0
		for _, cell := range cells {
			if cell.source == m.completion.start {
				return row, column, true
			}
			column += cell.width
		}
	}
	return 0, 0, false
}
