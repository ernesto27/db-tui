package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	rowCopyActionText     = "Copy selected row"
	rowCopyHelpText       = "c copy row"
	rowCopyNoticeText     = "Copied to clipboard"
	rowCopyNoticeDuration = 2 * time.Second
)

type rowCopyNoticeExpiredMsg struct {
	session uint64
	request uint64
}

type rowCopyState struct {
	visible bool
	session uint64
	request uint64
}

func (m Model) renderRowCopyToast(base string) string {
	if m.layout.width < 6 || m.layout.height < 4 {
		return base
	}
	message := truncateLabel(rowCopyNoticeText, m.layout.width-6)
	toast := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(colorText).
		Background(colorModalBackground).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Render(message)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(toast).
			X(max(0, m.layout.width-lipgloss.Width(toast)-1)).
			Y(max(0, min(m.layout.height, lipgloss.Height(base))-lipgloss.Height(toast)-1)).
			Z(1),
	).Render()
}

func (m *Model) copySelectedRow() tea.Cmd {
	row, ok := m.selectedRowForCopy()
	if !ok {
		return nil
	}
	m.rowCopy.visible = true
	m.rowCopy.session = m.session
	m.rowCopy.request++
	session, request := m.session, m.rowCopy.request
	return tea.Sequence(
		tea.SetClipboard(rowClipboardText(row)),
		tea.Tick(rowCopyNoticeDuration, func(time.Time) tea.Msg {
			return rowCopyNoticeExpiredMsg{session: session, request: request}
		}),
	)
}

func rowClipboardText(row []any) string {
	values := make([]string, len(row))
	for i, value := range row {
		switch value := value.(type) {
		case nil:
			values[i] = "NULL"
		case []byte:
			values[i] = string(value)
		default:
			values[i] = fmt.Sprint(value)
		}
	}
	return strings.Join(values, "\t")
}

func (m Model) selectedRowForCopy() ([]any, bool) {
	if (m.database == nil && m.redis.client == nil) || m.focus != focusData {
		return nil, false
	}
	var rows [][]any
	var selected int
	switch m.panel {
	case panelQuery:
		if m.redis.client != nil || !m.query.resultsFocused || m.query.loading || m.query.err != nil || len(m.query.result.Columns) == 0 {
			return nil, false
		}
		rows, selected = m.query.result.Rows, m.query.selectedRow
	case panelData:
		if m.data.loading || m.data.err != nil || len(m.data.page.Columns) == 0 {
			return nil, false
		}
		if m.redis.client == nil && (!m.activeRelation.set || m.activeFunction.set || m.activeExtensions.set) {
			return nil, false
		}
		rows, selected = m.data.page.Rows, m.data.selected
	default:
		return nil, false
	}
	if selected < 0 || selected >= len(rows) {
		return nil, false
	}
	return rows[selected], true
}
