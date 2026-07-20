// Package app contains the root Bubble Tea application model.
package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

const (
	bodyStartRow          = 1
	navigatorListStartRow = 4
	defaultWidth          = 100
	defaultHeight         = 24
	wheelDebounce         = 50 * time.Millisecond
	noModal               = -1
)

var mockTableNames = buildMockTableNames(200)

var mockUsers = buildMockUsers(200)

var mockUserHeaders = []string{
	"ID", "USERNAME", "NAME", "EMAIL", "ROLE", "DEPARTMENT", "STATUS", "CREATED", "LAST LOGIN",
	"PHONE", "COUNTRY", "CITY", "TZ", "LANG", "PLAN", "LOGINS", "API KEYS", "2FA", "UPDATED",
}

type focusPane uint8

const (
	focusNavigator focusPane = iota
	focusData
)

// Model is the root Bubble Tea model.
type Model struct {
	width            int
	height           int
	selected         int
	navigatorOffset  int
	userOffset       int
	userColumnOffset int
	focus            focusPane
	lastWheelAt      time.Time
	lastWheelButton  tea.MouseButton
	modalTable       int
	notice           string
}

// New creates the root application model with mock data.
func New() Model {
	return Model{
		width:      defaultWidth,
		height:     defaultHeight,
		selected:   0,
		modalTable: noModal,
	}
}

// Init implements tea.Model.
func (Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.modalTable != noModal {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
		case tea.KeyPressMsg:
			switch msg.String() {
			case "esc", "q":
				m.modalTable = noModal
			case "1":
				m.notice = "Mock action: preview rows"
				m.modalTable = noModal
			case "2":
				m.notice = "Mock action: describe table"
				m.modalTable = noModal
			case "3":
				m.notice = "Mock action: copy table name"
				m.modalTable = noModal
			}
		case tea.MouseClickMsg:
			m.modalTable = noModal
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureSelectionVisible()
		m.userOffset = min(m.userOffset, m.maxUserOffset())
		m.userColumnOffset = min(m.userColumnOffset, m.maxUserColumnOffset())
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "left":
			if m.focus == focusData && m.userColumnOffset > 0 {
				m.scrollUserColumns(-1)
			} else {
				m.focus = focusNavigator
			}
		case "right":
			if m.focus == focusNavigator {
				m.focus = focusData
			} else {
				m.scrollUserColumns(1)
			}
		case "up", "k":
			if m.focus == focusNavigator {
				m.moveSelection(-1)
			} else {
				m.scrollUsers(-1)
			}
		case "down", "j":
			if m.focus == focusNavigator {
				m.moveSelection(1)
			} else {
				m.scrollUsers(1)
			}
		case "pgup":
			if m.focus == focusNavigator {
				m.moveSelection(-m.visibleNavigatorRows())
			} else {
				m.scrollUsers(-m.visibleUserRows())
			}
		case "pgdown":
			if m.focus == focusNavigator {
				m.moveSelection(m.visibleNavigatorRows())
			} else {
				m.scrollUsers(m.visibleUserRows())
			}
		case "home":
			if m.focus == focusNavigator {
				m.selected = 0
				m.ensureSelectionVisible()
			} else {
				m.userOffset = 0
			}
		case "end":
			if m.focus == focusNavigator {
				m.selected = len(mockTableNames) - 1
				m.ensureSelectionVisible()
			} else {
				m.userOffset = m.maxUserOffset()
			}
		}
	case tea.MouseClickMsg:
		visibleIndex := msg.Y - bodyStartRow - navigatorListStartRow
		itemIndex := m.navigatorOffset + visibleIndex
		navigatorWidth := navigatorWidth(m.width)
		if (msg.Button == tea.MouseLeft || msg.Button == tea.MouseRight) &&
			msg.X > 0 && msg.X < navigatorWidth-1 &&
			visibleIndex >= 0 && visibleIndex < m.visibleNavigatorRows() &&
			itemIndex >= 0 && itemIndex < len(mockTableNames) {
			m.focus = focusNavigator
			m.selected = itemIndex
			m.userOffset = 0
			m.userColumnOffset = 0
			if msg.Button == tea.MouseRight {
				m.modalTable = itemIndex
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
				m.moveSelection(-1)
			case tea.MouseWheelDown:
				m.moveSelection(1)
			}
		} else if m.selected == 0 {
			m.focus = focusData
			switch msg.Button {
			case tea.MouseWheelUp:
				m.scrollUsers(-1)
			case tea.MouseWheelDown:
				m.scrollUsers(1)
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	width := max(m.width, 64)
	height := max(m.height, 16)
	bodyHeight := height - 2
	leftWidth := navigatorWidth(width)
	rightWidth := width - leftWidth - 1

	header := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Render("db-tui  /  demo_store  /  PostgreSQL (mock)")

	navigator := m.renderNavigator(leftWidth, bodyHeight, m.focus == focusNavigator)
	data := m.renderData(rightWidth, bodyHeight, m.focus == focusData)
	body := lipgloss.JoinHorizontal(lipgloss.Top, navigator, " ", data)

	firstTable := m.navigatorOffset + 1
	lastTable := min(m.navigatorOffset+m.visibleNavigatorRows(), len(mockTableNames))
	focusLabel := "tables"
	if m.focus == focusData {
		focusLabel = "data"
	}
	footerText := fmt.Sprintf("%s  •  tables %d–%d/%d  •  focus: %s  •  ←/→ switch  •  q quit", mockTableNames[m.selected], firstTable, lastTable, len(mockTableNames), focusLabel)
	if m.selected == 0 {
		firstUser := m.userOffset + 1
		lastUser := min(m.userOffset+m.visibleUserRows(), len(mockUsers))
		firstColumn := m.userColumnOffset + 1
		lastColumn := min(m.userColumnOffset+m.visibleUserColumns(m.dataPaneWidth()), len(mockUserHeaders))
		footerText = fmt.Sprintf("users %d–%d/%d  •  cols %d–%d/%d  •  focus: %s  •  ←/→ scroll  •  q quit", firstUser, lastUser, len(mockUsers), firstColumn, lastColumn, len(mockUserHeaders), focusLabel)
	}
	footer := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Foreground(lipgloss.Color("245")).
		Render(footerText)

	content := strings.Join([]string{header, body, footer}, "\n")
	if m.notice != "" {
		content += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(m.notice)
	}
	if m.modalTable != noModal {
		content = m.renderModal(content, width, height)
	}

	view := tea.NewView(content)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "db-tui prototype"
	return view
}

func (m Model) renderModal(base string, width, height int) string {
	tableName := mockTableNames[m.modalTable]
	dialog := lipgloss.NewStyle().
		Width(38).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("236")).
		Render(strings.Join([]string{
			"Table options",
			"",
			"Table: " + tableName,
			"",
			"1. Preview rows",
			"2. Describe table",
			"3. Copy table name",
			"",
			"Esc, q, or click to close",
		}, "\n"))

	dialogLayer := lipgloss.NewLayer(dialog).
		X(max(0, (width-lipgloss.Width(dialog))/2)).
		Y(max(0, (height-lipgloss.Height(dialog))/2)).
		Z(1)
	return lipgloss.NewCompositor(lipgloss.NewLayer(base), dialogLayer).Render()
}

func (m Model) renderNavigator(width, height int, focused bool) string {
	visibleRows := max(1, height-5)
	lastVisible := min(m.navigatorOffset+visibleRows, len(mockTableNames))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("● demo_store"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Render(
			fmt.Sprintf("TABLES (%d)", len(mockTableNames)),
		),
	}

	for index := m.navigatorOffset; index < lastVisible; index++ {
		name := mockTableNames[index]
		itemWidth := max(1, width-4)
		marker := "  "
		style := lipgloss.NewStyle().Width(itemWidth)
		if index == m.selected {
			marker = "> "
			style = style.Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62"))
		}
		lines = append(lines, style.Render(marker+truncateLabel(name, max(0, itemWidth-len(marker)))))
	}

	return panelStyle(width, height, focused).Render(strings.Join(lines, "\n"))
}

func (m Model) renderData(width, height int, focused bool) string {
	selectedTable := mockTableNames[m.selected]
	if selectedTable != "users" {
		content := strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(selectedTable + "  •  mock table"),
			"",
			"No preview rows in this prototype.",
			"Select users to view the mock data grid.",
		}, "\n")
		return panelStyle(width, height, focused).Render(content)
	}

	columnCount := m.visibleUserColumns(width)
	firstColumn := min(m.userColumnOffset, len(mockUserHeaders)-columnCount)
	lastColumn := min(firstColumn+columnCount, len(mockUserHeaders))
	headers := mockUserHeaders[firstColumn:lastColumn]
	rows := make([][]string, len(mockUsers))
	for index, row := range mockUsers {
		rows[index] = row[firstColumn:lastColumn]
	}

	grid := table.New().
		Headers(headers...).
		Rows(rows...).
		Width(max(20, width-4)).
		Height(max(5, height-4)).
		YOffset(m.userOffset).
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, _ int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(lipgloss.Color("86"))
			}
			if row%2 == 1 {
				return style.Foreground(lipgloss.Color("252"))
			}
			return style.Foreground(lipgloss.Color("250"))
		})

	content := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(
			fmt.Sprintf("users  •  200 rows  •  columns %d-%d/%d  •  mock data", firstColumn+1, lastColumn, len(mockUserHeaders)),
		),
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
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)
}

func navigatorWidth(totalWidth int) int {
	if totalWidth < 80 {
		return 20
	}
	return 26
}

func (m *Model) moveSelection(delta int) {
	m.selected = min(max(m.selected+delta, 0), len(mockTableNames)-1)
	m.ensureSelectionVisible()
}

func (m *Model) ensureSelectionVisible() {
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
	return max(0, len(mockTableNames)-m.visibleNavigatorRows())
}

func (m Model) visibleNavigatorRows() int {
	bodyHeight := max(m.height, 16) - 2
	return max(1, bodyHeight-5)
}

func (m *Model) scrollUsers(delta int) {
	if m.selected != 0 {
		return
	}
	m.userOffset = min(max(m.userOffset+delta, 0), m.maxUserOffset())
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

func (m Model) maxUserOffset() int {
	return max(0, len(mockUsers)-m.visibleUserRows())
}

func (m Model) visibleUserRows() int {
	return max(1, max(m.height, 16)-9)
}

func (m *Model) scrollUserColumns(delta int) {
	if m.selected != 0 {
		return
	}
	m.userColumnOffset = min(max(m.userColumnOffset+delta, 0), m.maxUserColumnOffset())
}

func (m Model) maxUserColumnOffset() int {
	return max(0, len(mockUserHeaders)-m.visibleUserColumns(m.dataPaneWidth()))
}

func (m Model) visibleUserColumns(dataWidth int) int {
	return min(len(mockUserHeaders), max(3, (dataWidth-4)/13))
}

func (m Model) dataPaneWidth() int {
	width := max(m.width, 64)
	return width - navigatorWidth(width) - 1
}

func buildMockTableNames(count int) []string {
	commonNames := []string{
		"users", "accounts", "profiles", "sessions", "roles",
		"permissions", "orders", "order_items", "products", "categories",
		"inventory", "warehouses", "payments", "invoices", "shipments",
		"addresses", "reviews", "coupons", "audit_logs", "events",
	}

	names := make([]string, 0, count)
	for index := range count {
		if index < len(commonNames) {
			names = append(names, commonNames[index])
			continue
		}
		names = append(names, fmt.Sprintf("archive_table_%03d", index+1))
	}
	return names
}

func buildMockUsers(count int) [][]string {
	firstNames := []string{"Ada", "Grace", "Alan", "Katherine", "Edsger", "Margaret", "Donald", "Barbara"}
	lastNames := []string{"Lovelace", "Hopper", "Turing", "Johnson", "Dijkstra", "Hamilton", "Knuth", "Liskov"}
	roles := []string{"admin", "editor", "viewer", "support"}
	departments := []string{"Engineering", "Operations", "Analytics", "Support"}
	countries := []string{"Argentina", "Canada", "Germany", "Japan"}
	cities := []string{"Buenos Aires", "Toronto", "Berlin", "Tokyo"}
	timeZones := []string{"UTC-3", "UTC-4", "UTC+2", "UTC+9"}
	languages := []string{"es", "en", "de", "ja"}
	plans := []string{"free", "team", "business", "enterprise"}

	rows := make([][]string, 0, count)
	for index := range count {
		id := index + 1
		status := "active"
		if id%7 == 0 {
			status = "inactive"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%03d", id),
			fmt.Sprintf("user%03d", id),
			firstNames[index%len(firstNames)] + " " + lastNames[(index/len(firstNames))%len(lastNames)],
			fmt.Sprintf("user%03d@example.com", id),
			roles[index%len(roles)],
			departments[index%len(departments)],
			status,
			fmt.Sprintf("2026-01-%02d", index%28+1),
			fmt.Sprintf("2026-07-%02d 10:%02d", index%19+1, index%60),
			fmt.Sprintf("+1-555-%04d", id),
			countries[index%len(countries)],
			cities[index%len(cities)],
			timeZones[index%len(timeZones)],
			languages[index%len(languages)],
			plans[index%len(plans)],
			fmt.Sprintf("%d", id*3),
			fmt.Sprintf("%d", index%5),
			map[bool]string{true: "on", false: "off"}[id%2 == 0],
			fmt.Sprintf("2026-07-%02d", index%19+1),
		})
	}
	return rows
}

func truncateLabel(label string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(label) <= width {
		return label
	}
	if width == 1 {
		return "…"
	}
	return label[:width-1] + "…"
}

var _ tea.Model = Model{}
