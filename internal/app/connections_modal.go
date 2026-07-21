package app

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
)

type connectionsModal struct {
	connections []config.Connection
	selected    int
}

type cancelConnectionsMsg struct{}
type selectConnectionMsg struct {
	connection config.Connection
}
type editConnectionMsg struct {
	index      int
	connection config.Connection
}

const (
	visibleConnectionRows = 10
	connectionEngineWidth = 14
)

func newConnectionsModal(appConfig config.Config) connectionsModal {
	return connectionsModal{
		connections: appConfig.Connections,
	}
}

func connectionSettingsFromConfig(connection config.Connection) ConnectionSettings {
	port, _ := strconv.Atoi(strings.TrimSpace(connection.Settings.Port))

	return ConnectionSettings{
		DSN:          strings.TrimSpace(connection.Settings.DSN),
		Host:         strings.TrimSpace(connection.Settings.Hostname),
		Port:         port,
		DatabaseName: strings.TrimSpace(connection.Settings.Database),
		Username:     strings.TrimSpace(connection.Settings.Username),
		Password:     connection.Settings.Password,
	}
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
	name := strings.TrimSpace(settings.DatabaseName)
	if name == "" {
		name = "PostgreSQL connection"
	}
	if host := strings.TrimSpace(settings.Host); host != "" {
		name += " (" + host + ")"
	}

	return config.Connection{
		Name:     name,
		Engine:   "postgres",
		Settings: configSettingsFromConnectionSettings(settings),
	}
}

func (m connectionsModal) update(msg tea.Msg) (connectionsModal, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "up":
			if len(m.connections) > 0 {
				m.selected = max(0, m.selected-1)
			}
		case "down":
			if len(m.connections) > 0 {
				m.selected = min(len(m.connections)-1, m.selected+1)
			}
		case "enter":
			if len(m.connections) == 0 {
				return m, nil
			}
			connection := m.connections[m.selected]
			return m, func() tea.Msg { return selectConnectionMsg{connection: connection} }
		case "ctrl+e":
			if len(m.connections) == 0 {
				return m, nil
			}
			connection := m.connections[m.selected]
			return m, func() tea.Msg { return editConnectionMsg{index: m.selected, connection: connection} }
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
	engineStyle := lipgloss.NewStyle().Width(connectionEngineWidth).Foreground(lipgloss.Color("245"))
	selectedNameStyle := nameStyle.Copy().Foreground(lipgloss.Color("230")).Bold(true)
	selectedEngineStyle := engineStyle.Copy().Foreground(lipgloss.Color("230")).Bold(true)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Connections"),
		"",
	}
	first := max(0, min(m.selected-visibleConnectionRows+1, len(m.connections)-visibleConnectionRows))
	last := min(first+visibleConnectionRows, len(m.connections))
	for index, connection := range m.connections[first:last] {
		name := truncateLabel(connection.Name, nameWidth)
		engine := truncateLabel(connection.Engine, connectionEngineWidth)
		if first+index == m.selected {
			lines = append(lines, "> "+selectedNameStyle.Render(name)+selectedEngineStyle.Render(engine))
			continue
		}
		lines = append(lines, "  "+nameStyle.Render(name)+engineStyle.Render(engine))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("↑/↓ move  •  Enter connect  •  Ctrl+E edit  •  Esc close"))

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Render(strings.Join(lines, "\n"))
}
