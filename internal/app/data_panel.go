package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type rowLoadRequest struct {
	offset      int
	selectedRow int
}

type dataModel struct {
	page         db.RowPage
	offset       int
	viewport     int
	selected     int
	columnOffset int
	loading      bool
	err          error
}

type dataStatus struct {
	tableName     string
	disconnected  bool
	tablesLoading bool
	tableLoadErr  error
	noTables      bool
	spinner       string
}

func (m *dataModel) reset() {
	*m = dataModel{}
}

func (m *dataModel) beginLoad(offset int) {
	m.page = db.RowPage{}
	m.offset = offset
	m.viewport = 0
	m.selected = 0
	m.columnOffset = 0
	m.loading = true
	m.err = nil
}

func (m *dataModel) finishLoad(page db.RowPage, selectedRow int, err error, layout appLayout) {
	m.loading = false
	m.err = err
	if err != nil {
		return
	}
	m.page = page
	m.viewport = 0
	m.selected = min(selectedRow, max(0, len(page.Rows)-1))
	m.columnOffset = 0
	m.ensureSelectedVisible(layout)
}

func (m *dataModel) moveUp(layout appLayout) (rowLoadRequest, bool) {
	if m.loading {
		return rowLoadRequest{}, false
	}
	if m.selected > 0 {
		m.selected--
		m.ensureSelectedVisible(layout)
		return rowLoadRequest{}, false
	}
	if m.offset > 0 {
		return rowLoadRequest{offset: max(0, m.offset-rowPageSize), selectedRow: rowPageSize - 1}, true
	}
	return rowLoadRequest{}, false
}

func (m *dataModel) moveDown(layout appLayout) (rowLoadRequest, bool) {
	if m.loading {
		return rowLoadRequest{}, false
	}
	if m.selected < len(m.page.Rows)-1 {
		m.selected++
		m.ensureSelectedVisible(layout)
		return rowLoadRequest{}, false
	}
	if m.page.HasMore {
		return rowLoadRequest{offset: m.offset + rowPageSize}, true
	}
	return rowLoadRequest{}, false
}

func (m *dataModel) scrollColumns(delta int, layout appLayout) {
	m.columnOffset = min(max(m.columnOffset+delta, 0), m.maxColumnOffset())
	m.ensureSelectedVisible(layout)
}

func (m *dataModel) ensureSelectedVisible(layout appLayout) {
	if len(m.page.Rows) == 0 {
		m.selected = 0
		m.viewport = 0
		return
	}
	m.selected = min(max(m.selected, 0), len(m.page.Rows)-1)
	if m.selected < m.viewport {
		m.viewport = m.selected
	}
	firstColumn, lastColumn := m.visibleColumnRange(layout.data.width)
	for m.visibleDataEnd(layout.data.width, layout.data.height, firstColumn, lastColumn, m.viewport) <= m.selected && m.viewport < m.selected {
		m.viewport++
	}
	m.viewport = min(max(m.viewport, 0), len(m.page.Rows)-1)
}

func (m dataModel) maxColumnOffset() int {
	return max(0, len(m.page.Columns)-1)
}

func (m dataModel) view(status dataStatus, layout appLayout, focused bool) string {
	switch {
	case status.disconnected:
		return panelStyle(layout.data.width, layout.data.height, focused).Render("Welcome to db-tui\n\nPress Ctrl+N to create a connection.\nPress Ctrl+L to open saved connections.")
	case status.tablesLoading:
		return panelStyle(layout.data.width, layout.data.height, focused).Render("Loading database tables…")
	case status.tableLoadErr != nil:
		return panelStyle(layout.data.width, layout.data.height, focused).Render("Unable to load database tables:\n" + sanitizeText(status.tableLoadErr.Error()))
	case status.noTables:
		return panelStyle(layout.data.width, layout.data.height, focused).Render("No tables found.")
	case m.loading:
		return panelStyle(layout.data.width, layout.data.height, focused).Render(status.tableName + "\n\n" + status.spinner + " Query executing…")
	case m.err != nil:
		return panelStyle(layout.data.width, layout.data.height, focused).Render(status.tableName + "\n\nUnable to load rows:\n" + sanitizeText(m.err.Error()))
	case len(m.page.Rows) == 0:
		return panelStyle(layout.data.width, layout.data.height, focused).Render(status.tableName + "\n\nNo rows in this page.")
	}

	firstColumn, lastColumn := m.visibleColumnRange(layout.data.width)
	firstVisibleRow := m.viewport
	lastVisibleRow := m.visibleDataEnd(layout.data.width, layout.data.height, firstColumn, lastColumn, firstVisibleRow)
	grid := m.dataGrid(layout.data.width, firstColumn, lastColumn, firstVisibleRow, lastVisibleRow)

	firstRow := m.offset + 1
	lastRow := m.offset + len(m.page.Rows)
	title := fmt.Sprintf("%s  •  rows %d–%d  •  columns %d–%d/%d", status.tableName, firstRow, lastRow, firstColumn+1, lastColumn, len(m.page.Columns))
	if m.page.HasMore {
		title += "  •  PgUp/PgDown page"
	}
	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(title),
		"",
		grid.String(),
	}, "\n")
	return panelStyle(layout.data.width, layout.data.height, focused).Render(content)
}
