// Package app contains the root Bubble Tea application model.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/ansi"

	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	bodyStartRow          = 1
	navigatorListStartRow = 4
	defaultWidth          = 100
	defaultHeight         = 24
	wheelDebounce         = 50 * time.Millisecond
	tableLoadTimeout      = 5 * time.Second
	rowPageSize           = 20
)

type focusPane uint8

const (
	focusNavigator focusPane = iota
	focusData
)

type tablesLoadedMsg struct {
	tables []db.Table
	err    error
}

type rowsLoadedMsg struct {
	tableName   string
	offset      int
	selectedRow int
	page        db.RowPage
	err         error
}

// Model is the root Bubble Tea application model.
type Model struct {
	database        db.Database
	databaseName    string
	tables          []db.Table
	loading         bool
	loadErr         error
	rowPage         db.RowPage
	rowOffset       int
	rowViewport     int
	rowSelected     int
	columnOffset    int
	rowsLoading     bool
	rowsErr         error
	width           int
	height          int
	selected        int
	navigatorOffset int
	focus           focusPane
	lastWheelAt     time.Time
	lastWheelButton tea.MouseButton
}

// New creates the root Bubble Tea application model for databaseName.
func New(database db.Database, databaseName string, connectErr error) Model {
	isLoading := database != nil && connectErr == nil

	return Model{
		database:     database,
		databaseName: databaseName,
		loading:      isLoading,
		loadErr:      connectErr,
		width:        defaultWidth,
		height:       defaultHeight,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.database == nil || m.loadErr != nil {
		return nil
	}
	return loadTables(m.database)
}

func loadTables(database db.Database) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		tables, err := database.ListTables(ctx)
		return tablesLoadedMsg{tables: tables, err: err}
	}
}

func loadRows(database db.Database, table db.Table, offset, selectedRow int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		page, err := database.GetRows(ctx, table, db.PageRequest{
			Offset: offset,
			Limit:  rowPageSize,
		})
		return rowsLoadedMsg{
			tableName:   table.Name,
			offset:      offset,
			selectedRow: selectedRow,
			page:        page,
			err:         err,
		}
	}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var command tea.Cmd

	switch msg := msg.(type) {
	case tablesLoadedMsg:
		m.loading = false
		m.loadErr = msg.err
		m.tables = msg.tables
		m.selected = 0
		m.navigatorOffset = 0
		m.resetRows()
		if msg.err == nil && len(m.tables) > 0 {
			command = m.startRowLoad(0, 0)
		}
	case rowsLoadedMsg:
		if len(m.tables) == 0 || m.selected >= len(m.tables) ||
			msg.tableName != m.tables[m.selected].Name || msg.offset != m.rowOffset {
			return m, nil
		}
		m.rowsLoading = false
		m.rowsErr = msg.err
		if msg.err == nil {
			m.rowPage = msg.page
			m.rowViewport = 0
			m.rowSelected = min(msg.selectedRow, max(0, len(m.rowPage.Rows)-1))
			m.ensureSelectedRowVisible()
			m.columnOffset = 0
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureSelectionVisible()
		m.ensureSelectedRowVisible()
		m.columnOffset = min(m.columnOffset, m.maxColumnOffset())
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "left":
			if m.focus == focusData && m.columnOffset > 0 {
				m.scrollColumns(-1)
			} else {
				m.focus = focusNavigator
			}
		case "right":
			if m.focus == focusNavigator {
				m.focus = focusData
			} else {
				m.scrollColumns(1)
			}
		case "up", "k":
			if m.focus == focusNavigator {
				if m.moveSelection(-1) {
					command = m.startRowLoad(0, 0)
				}
			} else {
				command = m.moveDataUp()
			}
		case "down", "j":
			if m.focus == focusNavigator {
				if m.moveSelection(1) {
					command = m.startRowLoad(0, 0)
				}
			} else {
				command = m.moveDataDown()
			}
		case "pgup":
			if m.focus == focusNavigator {
				if m.moveSelection(-m.visibleNavigatorRows()) {
					command = m.startRowLoad(0, 0)
				}
			} else if m.rowOffset > 0 && !m.rowsLoading {
				command = m.startRowLoad(max(0, m.rowOffset-rowPageSize), 0)
			}
		case "pgdown":
			if m.focus == focusNavigator {
				if m.moveSelection(m.visibleNavigatorRows()) {
					command = m.startRowLoad(0, 0)
				}
			} else if m.rowPage.HasMore && !m.rowsLoading {
				command = m.startRowLoad(m.rowOffset+rowPageSize, 0)
			}
		case "home":
			if m.focus == focusNavigator && len(m.tables) > 0 && m.selected != 0 {
				m.selected = 0
				m.ensureSelectionVisible()
				command = m.startRowLoad(0, 0)
			}
		case "end":
			if m.focus == focusNavigator && len(m.tables) > 0 && m.selected != len(m.tables)-1 {
				m.selected = len(m.tables) - 1
				m.ensureSelectionVisible()
				command = m.startRowLoad(0, 0)
			}
		}
	case tea.MouseClickMsg:
		visibleIndex := msg.Y - bodyStartRow - navigatorListStartRow
		itemIndex := m.navigatorOffset + visibleIndex
		navigatorWidth := navigatorWidth(m.width)
		if msg.Button == tea.MouseLeft && msg.X > 0 && msg.X < navigatorWidth-1 &&
			visibleIndex >= 0 && visibleIndex < m.visibleNavigatorRows() &&
			itemIndex >= 0 && itemIndex < len(m.tables) {
			m.focus = focusNavigator
			if m.selected != itemIndex {
				m.selected = itemIndex
				command = m.startRowLoad(0, 0)
			}
		} else if msg.Button == tea.MouseLeft && msg.X >= navigatorWidth {
			m.focus = focusData
		}
	case tea.MouseWheelMsg:
		if !m.acceptWheel(msg.Button) {
			return m, nil
		}
		if msg.X >= 0 && msg.X < navigatorWidth(m.width) {
			m.focus = focusNavigator
			switch msg.Button {
			case tea.MouseWheelUp:
				if m.moveSelection(-1) {
					command = m.startRowLoad(0, 0)
				}
			case tea.MouseWheelDown:
				if m.moveSelection(1) {
					command = m.startRowLoad(0, 0)
				}
			}
		} else {
			m.focus = focusData
			switch msg.Button {
			case tea.MouseWheelUp:
				command = m.moveDataUp()
			case tea.MouseWheelDown:
				command = m.moveDataDown()
			}
		}
	}

	return m, command
}

// View implements tea.Model.
func (m Model) View() tea.View {
	width := max(m.width, 64)
	height := max(m.height, 16)
	bodyHeight := height - 2
	leftWidth := navigatorWidth(width)
	rightWidth := width - leftWidth - 1

	header := lipgloss.NewStyle().Width(width).Padding(0, 1).Bold(true).
		Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).
		Render(fmt.Sprintf("db-tui  /  %s  /  PostgreSQL", sanitizeText(m.databaseName)))
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderNavigator(leftWidth, bodyHeight, m.focus == focusNavigator), " ",
		m.renderData(rightWidth, bodyHeight, m.focus == focusData),
	)
	footer := lipgloss.NewStyle().Width(width).Padding(0, 1).
		Foreground(lipgloss.Color("245")).Render(m.footerText())

	view := tea.NewView(strings.Join([]string{header, body, footer}, "\n"))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "db-tui"
	return view
}

func (m Model) footerText() string {
	if m.loading {
		return "loading tables  •  q quit"
	}
	if m.loadErr != nil {
		return "unable to load tables  •  q quit"
	}
	if len(m.tables) == 0 {
		return "no public tables  •  q quit"
	}

	firstTable := m.navigatorOffset + 1
	lastTable := min(m.navigatorOffset+m.visibleNavigatorRows(), len(m.tables))
	focusLabel := "tables"
	if m.focus == focusData {
		focusLabel = "data"
	}
	rowStatus := ""
	if m.rowsLoading {
		rowStatus = "  •  query executing…"
	} else if m.rowsErr != nil {
		rowStatus = "  •  row load failed"
	} else if len(m.rowPage.Rows) > 0 {
		firstRow := m.rowOffset + 1
		lastRow := m.rowOffset + len(m.rowPage.Rows)
		rowStatus = fmt.Sprintf("  •  rows %d–%d", firstRow, lastRow)
		if m.rowPage.HasMore {
			rowStatus += "  •  PgDown next"
		}
	}
	return fmt.Sprintf("%s  •  tables %d–%d/%d  •  focus: %s%s  •  ←/→ switch  •  q quit", m.selectedTableName(), firstTable, lastTable, len(m.tables), focusLabel, rowStatus)
}

func (m Model) renderNavigator(width, height int, focused bool) string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("● " + sanitizeText(m.databaseName)), ""}
	switch {
	case m.loading:
		lines = append(lines, "Loading tables…")
	case m.loadErr != nil:
		lines = append(lines, "Unable to load tables", truncateLabel(m.loadErr.Error(), max(1, width-4)))
	case len(m.tables) == 0:
		lines = append(lines, "No public tables.")
	default:
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Render(fmt.Sprintf("TABLES (%d)", len(m.tables))))
		lastVisible := min(m.navigatorOffset+m.visibleNavigatorRows(), len(m.tables))
		for index := m.navigatorOffset; index < lastVisible; index++ {
			itemWidth := max(1, width-4)
			marker := "  "
			style := lipgloss.NewStyle().Width(itemWidth)
			if index == m.selected {
				marker = "> "
				style = style.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
			}
			lines = append(lines, style.Render(marker+truncateLabel(m.tables[index].Name, max(0, itemWidth-len(marker)))))
		}
	}
	return panelStyle(width, height, focused).Render(strings.Join(lines, "\n"))
}

func (m Model) renderData(width, height int, focused bool) string {
	tableName := m.selectedTableName()
	switch {
	case m.loading:
		return panelStyle(width, height, focused).Render("Loading PostgreSQL tables…")
	case m.loadErr != nil:
		return panelStyle(width, height, focused).Render("Unable to load PostgreSQL tables:\n" + sanitizeText(m.loadErr.Error()))
	case len(m.tables) == 0:
		return panelStyle(width, height, focused).Render("No public tables found.")
	case m.rowsLoading:
		return panelStyle(width, height, focused).Render(tableName + "\n\nQuery executing…")
	case m.rowsErr != nil:
		return panelStyle(width, height, focused).Render(tableName + "\n\nUnable to load rows:\n" + sanitizeText(m.rowsErr.Error()))
	case len(m.rowPage.Rows) == 0:
		return panelStyle(width, height, focused).Render(tableName + "\n\nNo rows in this page.")
	}

	columnCount := m.visibleDataColumns(width)
	firstColumn := min(m.columnOffset, len(m.rowPage.Columns)-columnCount)
	lastColumn := min(firstColumn+columnCount, len(m.rowPage.Columns))
	headers := make([]string, 0, lastColumn-firstColumn)
	for _, column := range m.rowPage.Columns[firstColumn:lastColumn] {
		headers = append(headers, sanitizeText(column))
	}
	rows := make([][]string, len(m.rowPage.Rows))
	for rowIndex, row := range m.rowPage.Rows {
		rows[rowIndex] = make([]string, 0, lastColumn-firstColumn)
		for columnIndex := firstColumn; columnIndex < lastColumn; columnIndex++ {
			var value any
			if columnIndex < len(row) {
				value = row[columnIndex]
			}
			rows[rowIndex] = append(rows[rowIndex], formatCell(value))
		}
	}

	grid := table.New().
		Headers(headers...).
		Rows(rows...).
		Width(max(20, width-4)).
		Height(max(5, height-5)).
		YOffset(m.rowViewport).
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, _ int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(lipgloss.Color("86"))
			}
			if row == m.rowSelected {
				return style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
			}
			if row%2 == 1 {
				return style.Foreground(lipgloss.Color("252"))
			}
			return style.Foreground(lipgloss.Color("250"))
		})

	firstRow := m.rowOffset + 1
	lastRow := m.rowOffset + len(m.rowPage.Rows)
	title := fmt.Sprintf("%s  •  rows %d–%d  •  columns %d–%d/%d", tableName, firstRow, lastRow, firstColumn+1, lastColumn, len(m.rowPage.Columns))
	if m.rowPage.HasMore {
		title += "  •  PgUp/PgDown page"
	}
	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(title),
		"",
		grid.String(),
	}, "\n")
	return panelStyle(width, height, focused).Render(content)
}

func panelStyle(width, height int, focused bool) lipgloss.Style {
	borderColor := lipgloss.Color("240")
	if focused {
		borderColor = lipgloss.Color("62")
	}
	return lipgloss.NewStyle().Width(width).Height(height).Padding(0, 1).
		Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
}

func navigatorWidth(totalWidth int) int {
	if totalWidth < 80 {
		return 20
	}
	return 26
}

func (m *Model) startRowLoad(offset, selectedRow int) tea.Cmd {
	if m.database == nil || len(m.tables) == 0 || m.selected >= len(m.tables) {
		return nil
	}
	m.rowOffset = offset
	m.rowViewport = 0
	m.rowSelected = 0
	m.columnOffset = 0
	m.rowPage = db.RowPage{}
	m.rowsLoading = true
	m.rowsErr = nil
	return loadRows(m.database, m.tables[m.selected], offset, selectedRow)
}

func (m *Model) resetRows() {
	m.rowPage = db.RowPage{}
	m.rowOffset = 0
	m.rowViewport = 0
	m.rowSelected = 0
	m.columnOffset = 0
	m.rowsLoading = false
	m.rowsErr = nil
}

func (m *Model) moveSelection(delta int) bool {
	if len(m.tables) == 0 {
		m.selected = 0
		m.navigatorOffset = 0
		return false
	}
	previous := m.selected
	m.selected = min(max(m.selected+delta, 0), len(m.tables)-1)
	m.ensureSelectionVisible()
	return m.selected != previous
}

func (m *Model) ensureSelectionVisible() {
	if len(m.tables) == 0 {
		m.selected = 0
		m.navigatorOffset = 0
		return
	}
	m.selected = min(max(m.selected, 0), len(m.tables)-1)
	visibleRows := m.visibleNavigatorRows()
	if m.selected < m.navigatorOffset {
		m.navigatorOffset = m.selected
	}
	if m.selected >= m.navigatorOffset+visibleRows {
		m.navigatorOffset = m.selected - visibleRows + 1
	}
	m.navigatorOffset = min(max(m.navigatorOffset, 0), m.maxNavigatorOffset())
}

func (m Model) maxNavigatorOffset() int {
	return max(0, len(m.tables)-m.visibleNavigatorRows())
}

func (m Model) visibleNavigatorRows() int {
	bodyHeight := max(m.height, 16) - 2
	return max(1, bodyHeight-5)
}

func (m *Model) moveDataUp() tea.Cmd {
	if m.rowsLoading {
		return nil
	}
	if m.rowSelected > 0 {
		m.rowSelected--
		m.ensureSelectedRowVisible()
		return nil
	}
	if m.rowOffset > 0 {
		return m.startRowLoad(max(0, m.rowOffset-rowPageSize), rowPageSize-1)
	}
	return nil
}

func (m *Model) moveDataDown() tea.Cmd {
	if m.rowsLoading {
		return nil
	}
	if m.rowSelected < len(m.rowPage.Rows)-1 {
		m.rowSelected++
		m.ensureSelectedRowVisible()
		return nil
	}
	if m.rowPage.HasMore {
		return m.startRowLoad(m.rowOffset+rowPageSize, 0)
	}
	return nil
}

func (m *Model) ensureSelectedRowVisible() {
	if len(m.rowPage.Rows) == 0 {
		m.rowSelected = 0
		m.rowViewport = 0
		return
	}
	m.rowSelected = min(max(m.rowSelected, 0), len(m.rowPage.Rows)-1)
	visibleRows := m.visibleDataRows()
	if m.rowSelected < m.rowViewport {
		m.rowViewport = m.rowSelected
	}
	if m.rowSelected >= m.rowViewport+visibleRows {
		m.rowViewport = m.rowSelected - visibleRows + 1
	}
	m.rowViewport = min(max(m.rowViewport, 0), m.maxRowViewport())
}

func (m Model) maxRowViewport() int {
	return max(0, len(m.rowPage.Rows)-m.visibleDataRows())
}

func (m Model) visibleDataRows() int {
	bodyHeight := max(m.height, 16) - 2
	gridHeight := max(5, bodyHeight-5)
	return max(1, gridHeight-5)
}

func (m *Model) scrollColumns(delta int) {
	m.columnOffset = min(max(m.columnOffset+delta, 0), m.maxColumnOffset())
}

func (m Model) maxColumnOffset() int {
	return max(0, len(m.rowPage.Columns)-m.visibleDataColumns(m.dataPaneWidth()))
}

func (m Model) visibleDataColumns(dataWidth int) int {
	return min(len(m.rowPage.Columns), max(1, (dataWidth-4)/14))
}

func (m Model) dataPaneWidth() int {
	width := max(m.width, 64)
	return width - navigatorWidth(width) - 1
}

func (m *Model) acceptWheel(button tea.MouseButton) bool {
	now := time.Now()
	if button == m.lastWheelButton && now.Sub(m.lastWheelAt) < wheelDebounce {
		return false
	}
	m.lastWheelAt = now
	m.lastWheelButton = button
	return true
}

func (m Model) selectedTableName() string {
	if len(m.tables) == 0 || m.selected >= len(m.tables) {
		return ""
	}
	return sanitizeText(m.tables[m.selected].Name)
}

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

var _ tea.Model = Model{}
