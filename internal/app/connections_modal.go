package app

import (
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

type connectionsModal struct {
	connections        []config.Connection
	search             textinput.Model
	searchFocused      bool
	selected           int
	confirmingDeletion bool
	deletionError      string
}

type visibleConnection struct {
	index      int
	connection config.Connection
}

type cancelConnectionsMsg struct{}
type selectConnectionMsg struct {
	index      int
	connection config.Connection
}
type editConnectionMsg struct {
	index      int
	connection config.Connection
}
type deleteConnectionMsg struct {
	index      int
	connection config.Connection
}

const (
	visibleConnectionRows = 10
	connectionEngineWidth = 14
)

func newConnectionsModal(appConfig config.Config) connectionsModal {
	search := textinput.New()
	search.Prompt = ""
	search.Placeholder = "Search"
	search.SetWidth(connectionModalInputWidth)
	search.Focus()
	return connectionsModal{
		connections:   appConfig.Connections,
		search:        search,
		searchFocused: true,
	}
}

func connectionSettingsFromConfig(connection config.Connection) ConnectionSettings {
	port, _ := strconv.Atoi(strings.TrimSpace(connection.Settings.Port))

	settings := ConnectionSettings{
		Engine:       strings.TrimSpace(connection.Engine),
		DSN:          strings.TrimSpace(connection.Settings.DSN),
		Host:         strings.TrimSpace(connection.Settings.Hostname),
		Port:         port,
		DatabaseName: strings.TrimSpace(connection.Settings.Database),
		Username:     strings.TrimSpace(connection.Settings.Username),
		Password:     connection.Settings.Password,
	}
	if engine, err := settings.normalizedEngine(); err == nil {
		settings.Engine = engine
	}
	return settings
}

func configSettingsFromConnectionSettings(settings ConnectionSettings) config.Settings {
	port := ""
	if settings.DSN == "" && settings.Port > 0 {
		port = strconv.Itoa(settings.Port)
	}

	return config.Settings{
		Hostname: settings.Host,
		Database: settings.DatabaseName,
		Username: settings.Username,
		Password: settings.Password,
		Port:     port,
		DSN:      settings.DSN,
	}
}

func newConfigConnection(settings ConnectionSettings) config.Connection {
	engine, err := settings.normalizedEngine()
	if err != nil {
		engine = strings.ToLower(strings.TrimSpace(settings.Engine))
	}
	name := strings.TrimSpace(settings.DatabaseName)
	if name == "" && engine == db.EngineSQLite && strings.TrimSpace(settings.DSN) != "" {
		name = filepath.Base(strings.TrimSpace(settings.DSN)) + "-" + strconv.Itoa(rand.IntN(1_000_000))
	}
	if name == "" {
		name = engineDisplayName(engine) + " connection"
	}
	if host := strings.TrimSpace(settings.Host); host != "" {
		name += " (" + host + ")"
	}

	return config.Connection{
		Name:     name,
		Engine:   engine,
		Settings: configSettingsFromConnectionSettings(settings),
	}
}

func (m connectionsModal) visibleConnections() []visibleConnection {
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	connections := make([]visibleConnection, 0, len(m.connections))
	for index, connection := range m.connections {
		if query != "" && !strings.Contains(strings.ToLower(connection.Name), query) {
			continue
		}
		connections = append(connections, visibleConnection{index: index, connection: connection})
	}
	return connections
}

func (m connectionsModal) selectedConnection() (visibleConnection, bool) {
	connections := m.visibleConnections()
	if m.selected < 0 || m.selected >= len(connections) {
		return visibleConnection{}, false
	}
	return connections[m.selected], true
}

func (m connectionsModal) update(msg tea.Msg) (connectionsModal, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if m.confirmingDeletion {
			switch key.String() {
			case "y":
				connection, ok := m.selectedConnection()
				if !ok {
					m.confirmingDeletion = false
					return m, nil
				}
				return m, func() tea.Msg {
					return deleteConnectionMsg{index: connection.index, connection: connection.connection}
				}
			case "n", "esc":
				m.confirmingDeletion = false
				m.deletionError = ""
			}
			return m, nil
		}

		if m.searchFocused {
			switch key.String() {
			case "enter":
				connection, ok := m.selectedConnection()
				if !ok {
					return m, nil
				}
				return m, func() tea.Msg {
					return selectConnectionMsg{index: connection.index, connection: connection.connection}
				}
			case "down":
				if len(m.visibleConnections()) > 0 {
					m.searchFocused = false
					m.search.Blur()
				}
				return m, nil
			case "esc":
				return m, func() tea.Msg { return cancelConnectionsMsg{} }
			}

			previousQuery := m.search.Value()
			var command tea.Cmd
			m.search, command = m.search.Update(msg)
			if m.search.Value() != previousQuery {
				m.selected = 0
			}
			return m, command
		}

		switch key.String() {
		case "f":
			m.searchFocused = true
			return m, m.search.Focus()
		case "up":
			if m.selected == 0 {
				m.searchFocused = true
				return m, m.search.Focus()
			}
			m.selected--
		case "down":
			if connections := m.visibleConnections(); len(connections) > 0 {
				m.selected = min(len(connections)-1, m.selected+1)
			}
		case "enter":
			connection, ok := m.selectedConnection()
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return selectConnectionMsg{index: connection.index, connection: connection.connection} }
		case "ctrl+e":
			connection, ok := m.selectedConnection()
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return editConnectionMsg{index: connection.index, connection: connection.connection} }
		case "d":
			if _, ok := m.selectedConnection(); !ok {
				return m, nil
			}
			m.confirmingDeletion = true
			m.deletionError = ""
		case "esc":
			return m, func() tea.Msg { return cancelConnectionsMsg{} }
		}
	}
	return m, nil
}

func (m connectionsModal) view(width int) string {
	modalWidth := min(56, max(40, width-8))
	nameWidth := max(12, modalWidth-24)
	nameStyle := lipgloss.NewStyle().Width(nameWidth)
	engineStyle := lipgloss.NewStyle().Width(connectionEngineWidth).Foreground(colorTextMuted)
	selectedNameStyle := nameStyle.Copy().Foreground(colorTitle).Bold(true)
	selectedEngineStyle := engineStyle.Copy().Foreground(colorTitle).Bold(true)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Connections"),
		"",
	}
	if m.confirmingDeletion {
		connection, ok := m.selectedConnection()
		if !ok {
			return ""
		}
		lines = append(lines,
			"Remove "+truncateLabel(connection.connection.Name, modalWidth-8)+"?",
			"",
			lipgloss.NewStyle().Foreground(colorTextMuted).Render("y confirm  •  n/Esc cancel"),
		)
		if m.deletionError != "" {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(colorError).Render(sanitizeText(m.deletionError)))
		}
		return lipgloss.NewStyle().
			Width(modalWidth).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderActive).
			Render(strings.Join(lines, "\n"))
	}

	search := m.search
	search.SetWidth(modalWidth - 6)
	lines = append(lines, search.View(), "")
	connections := m.visibleConnections()
	if len(connections) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorTextMuted).Render("No matching connections"))
	}
	first := max(0, min(m.selected-visibleConnectionRows+1, len(connections)-visibleConnectionRows))
	last := min(first+visibleConnectionRows, len(connections))
	for index, item := range connections[first:last] {
		connection := item.connection
		name := truncateLabel(connection.Name, nameWidth)
		engine := truncateLabel(connection.Engine, connectionEngineWidth)
		if first+index == m.selected {
			lines = append(lines, "> "+selectedNameStyle.Render(name)+selectedEngineStyle.Render(engine))
			continue
		}
		lines = append(lines, "  "+nameStyle.Render(name)+engineStyle.Render(engine))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("↑/↓ move  •  f search  •  Enter connect  •  Ctrl+E edit  •  d remove  •  Esc close"))

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Render(strings.Join(lines, "\n"))
}
