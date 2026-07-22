package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	queryPanelHorizontalPadding = 1
	querySectionHeightDivisor   = 4
	querySectionMinimumHeight   = 3
	querySectionReservedRows    = 2
)

type queryModel struct {
	editor         textarea.Model
	result         db.QueryResult
	loading        bool
	err            error
	request        uint64
	viewport       int
	resultsFocused bool
}

func newQueryModel(layout appLayout) queryModel {
	editor := textarea.New()
	editor.Placeholder = "Write SQL…"
	editor.Prompt = ""
	editor.ShowLineNumbers = false
	styleQueryEditor(&editor)
	editor.MaxHeight = 0
	editor.MaxContentHeight = 0
	editor.SetHeight(queryEditorViewportHeight(layout))
	editor.SetWidth(queryContentWidth(layout))
	return queryModel{editor: editor}
}

func (m *queryModel) reset(layout appLayout) {
	*m = newQueryModel(layout)
}

func (m *queryModel) resize(layout appLayout) {
	styleQueryEditor(&m.editor)
	m.editor.SetWidth(queryContentWidth(layout))
	m.editor.SetHeight(queryEditorViewportHeight(layout))
}

func (m *queryModel) beginExecute() uint64 {
	m.loading = true
	m.err = nil
	m.result = db.QueryResult{}
	m.viewport = 0
	m.resultsFocused = false
	m.request++
	return m.request
}

func (m *queryModel) finishExecute(result db.QueryResult, err error) {
	m.loading = false
	m.result = result
	m.err = err
	m.viewport = 0
	m.resultsFocused = len(result.Rows) > 0
	if m.resultsFocused {
		m.editor.Blur()
	}
}

func (m *queryModel) focusEditor() tea.Cmd {
	m.resultsFocused = false
	return m.editor.Focus()
}

func (m *queryModel) toggleFocus() tea.Cmd {
	m.resultsFocused = !m.resultsFocused
	if m.resultsFocused {
		m.editor.Blur()
		return nil
	}
	return m.editor.Focus()
}

func (m *queryModel) scrollResults(delta int) {
	if len(m.result.Rows) == 0 {
		m.viewport = 0
		return
	}
	m.viewport = min(max(m.viewport+delta, 0), len(m.result.Rows)-1)
}

func (m queryModel) view(layout appLayout, focused, connected bool, spinner string) string {
	headingText := "RAW QUERY"
	if m.resultsFocused {
		headingText += "  •  results focused"
	}
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(headingText)
	result := m.resultView(layout, connected, spinner)
	content := strings.Join([]string{heading, m.editor.View(), "", result}, "\n")
	return queryPanelStyle(layout.data.width, layout.data.height, focused).Render(content)
}

func (m queryModel) resultView(layout appLayout, connected bool, spinner string) string {
	switch {
	case !connected:
		return "A database connection is required to run SQL."
	case m.loading:
		return spinner + " Query executing…"
	case m.err != nil:
		return "Query failed:\n" + sanitizeText(m.err.Error())
	case len(m.result.Columns) == 0 && m.result.CommandTag != "":
		return "Command completed: " + sanitizeText(m.result.CommandTag)
	case len(m.result.Columns) == 0:
		return "Write SQL above, then press Ctrl+P to execute it."
	case len(m.result.Rows) == 0:
		return "Query returned no rows.  •  " + sanitizeText(m.result.CommandTag)
	}

	contentWidth := queryContentWidth(layout)
	gridModel := dataModel{
		page:     db.RowPage{Columns: m.result.Columns, Rows: m.result.Rows},
		selected: -1,
		viewport: m.viewport,
	}
	firstColumn, lastColumn := gridModel.visibleColumnRange(contentWidth)
	resultHeight := m.resultHeight(layout)
	lastRow := gridModel.visibleDataEnd(contentWidth, resultHeight, firstColumn, lastColumn, m.viewport)
	grid := gridModel.dataGrid(contentWidth, firstColumn, lastColumn, m.viewport, lastRow)
	title := fmt.Sprintf("Results  •  rows %d–%d/%d  •  %s", m.viewport+1, lastRow, len(m.result.Rows), sanitizeText(m.result.CommandTag))
	return strings.Join([]string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(title),
		grid.String(),
	}, "\n")
}

func queryPanelStyle(width, height int, focused bool) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if focused {
		borderColor = lipgloss.Color("62")
	}
	return lipgloss.NewStyle().Width(width).Height(height).
		Padding(0, queryPanelHorizontalPadding).
		Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
}

func queryContentWidth(layout appLayout) int {
	return max(1, layout.data.width-2-queryPanelHorizontalPadding*2)
}

func (m queryModel) resultHeight(layout appLayout) int {
	return max(1, layout.data.height-querySectionHeight(layout))
}

func querySectionHeight(layout appLayout) int {
	return max(querySectionMinimumHeight, layout.data.height/querySectionHeightDivisor)
}

func queryEditorViewportHeight(layout appLayout) int {
	return max(1, querySectionHeight(layout)-querySectionReservedRows)
}

func styleQueryEditor(editor *textarea.Model) {
	styles := editor.Styles()
	focusedText := lipgloss.Color("252")
	blurredText := lipgloss.Color("250")
	styles.Focused.Text = styles.Focused.Text.Foreground(focusedText)
	styles.Blurred.Text = styles.Blurred.Text.Foreground(blurredText)
	styles.Focused.CursorLine = lipgloss.NewStyle().Foreground(focusedText)
	styles.Blurred.CursorLine = lipgloss.NewStyle().Foreground(blurredText)
	styles.Cursor.Color = focusedText
	editor.SetStyles(styles)
}
