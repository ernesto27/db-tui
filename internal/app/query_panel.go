package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	queryPanelHorizontalPadding = 1
	querySectionHeightDivisor   = 2
	querySectionMinimumHeight   = 3
	querySectionReservedRows    = 2
	queryExecutingText          = "Query executing: "
	queryCancelControlText      = "[Cancel]"
)

type queryModel struct {
	editor             textarea.Model
	selection          sqlSelection
	result             db.QueryResult
	loading            bool
	err                error
	request            uint64
	lastExecutedSQL    string
	loadedScriptName   string
	saveWarning        string
	viewport           int
	resultsFocused     bool
	executionDuration  time.Duration
	executionStartedAt time.Time
	cancel             context.CancelFunc
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
	m.cancelExecution()
	request := m.request
	*m = newQueryModel(layout)
	m.request = request
}

func (m *queryModel) resize(layout appLayout) {
	styleQueryEditor(&m.editor)
	m.editor.SetWidth(queryContentWidth(layout))
	m.editor.SetHeight(queryEditorViewportHeight(layout))
}

func (m *queryModel) beginExecute(sql string) uint64 {
	m.loading = true
	m.err = nil
	m.result = db.QueryResult{}
	m.viewport = 0
	m.resultsFocused = false
	m.lastExecutedSQL = sql
	m.request++
	m.executionDuration = 0
	m.executionStartedAt = time.Now()
	m.saveWarning = ""
	return m.request
}

func (m *queryModel) finishExecute(result db.QueryResult, duration time.Duration, err error) {
	m.cancelExecution()
	m.loading = false
	m.result = result
	m.err = err
	m.viewport = 0
	m.executionDuration = duration
	m.resultsFocused = len(result.Rows) > 0
	if m.resultsFocused {
		m.editor.Blur()
	}
}

func (m *queryModel) cancelExecution() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
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

func (m queryModel) executionTimeText() string {
	return "Execution time: " + m.executionDuration.String()
}

func (m queryModel) cancelControlContains(x, y int, layout appLayout) bool {
	if !m.loading {
		return false
	}

	controlX := layout.data.x + 2 + len(queryExecutingText) + len(m.executionDuration.String()) + 2
	controlY := layout.data.y + querySectionHeight(layout) + 1
	return x >= controlX && x < controlX+len(queryCancelControlText) && y == controlY
}

func (m queryModel) view(layout appLayout, focused, connected bool) string {
	headingText := "RAW QUERY"
	if m.resultsFocused {
		headingText += "  •  results focused"
	}
	heading := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render(headingText)
	result := m.resultView(layout, connected)
	sections := []string{heading, m.editorView(layout), ""}
	if m.saveWarning != "" {
		sections = append(sections, lipgloss.NewStyle().Foreground(colorError).Render("⚠ SQL script was not saved: "+sanitizeText(m.saveWarning)), "")
	}
	sections = append(sections, result)
	content := strings.Join(sections, "\n")
	return queryPanelStyle(layout.data.width, layout.data.height, focused).Render(content)
}

func (m queryModel) resultView(layout appLayout, connected bool) string {
	switch {
	case !connected:
		return "A database connection is required to run SQL."
	case m.loading:
		cancelControl := lipgloss.NewStyle().Foreground(colorTextInactive).Render(queryCancelControlText)
		return queryExecutingText + m.executionDuration.String() + "  " + cancelControl
	case m.err != nil:
		return "Query failed  •  " + m.executionTimeText() +
			":\n" + sanitizeText(m.err.Error())
	case len(m.result.Columns) == 0 && m.result.CommandTag != "":
		return "Command completed: " + sanitizeText(m.result.CommandTag) +
			"  •  " + m.executionTimeText()
	case len(m.result.Columns) == 0:
		return "Write SQL above, then press Ctrl+P to execute it."
	case len(m.result.Rows) == 0:
		return "Query returned no rows.  •  " +
			sanitizeText(m.result.CommandTag) +
			"  •  " + m.executionTimeText()
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
	title := fmt.Sprintf(
		"Results  •  rows %d–%d/%d  •  %s  •  %s",
		m.viewport+1,
		lastRow,
		len(m.result.Rows),
		sanitizeText(m.result.CommandTag),
		m.executionTimeText(),
	)
	title = truncateLabel(title, contentWidth)
	return strings.Join([]string{
		lipgloss.NewStyle().Foreground(colorTextMuted).Render(title),
		grid.String(),
	}, "\n")
}

func queryPanelStyle(width, height int, focused bool) lipgloss.Style {
	borderColor := colorBorderInactive
	if focused {
		borderColor = colorBorderActive
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
	focusedText := colorText
	blurredText := colorTextInactive
	styles.Focused.Text = styles.Focused.Text.Foreground(focusedText)
	styles.Blurred.Text = styles.Blurred.Text.Foreground(blurredText)
	styles.Focused.CursorLine = lipgloss.NewStyle().Foreground(focusedText)
	styles.Blurred.CursorLine = lipgloss.NewStyle().Foreground(blurredText)
	styles.Cursor.Color = focusedText
	editor.SetStyles(styles)
}
