package app

import (
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

type connectionsModal struct {
	connections        []config.Connection
	selected           int
	confirmingDeletion bool
	deletionError      string
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
	return connectionsModal{
		connections: appConfig.Connections,
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

func (m connectionsModal) update(msg tea.Msg) (connectionsModal, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if m.confirmingDeletion {
			switch key.String() {
			case "y":
				connection := m.connections[m.selected]
				return m, func() tea.Msg {
					return deleteConnectionMsg{index: m.selected, connection: connection}
				}
			case "n", "esc":
				m.confirmingDeletion = false
				m.deletionError = ""
			}
			return m, nil
		}

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
			return m, func() tea.Msg { return selectConnectionMsg{index: m.selected, connection: connection} }
		case "ctrl+e":
			if len(m.connections) == 0 {
				return m, nil
			}
			connection := m.connections[m.selected]
			return m, func() tea.Msg { return editConnectionMsg{index: m.selected, connection: connection} }
		case "d":
			if len(m.connections) == 0 {
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
		connection := m.connections[m.selected]
		lines = append(lines,
			"Remove "+truncateLabel(connection.Name, modalWidth-8)+"?",
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
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("↑/↓ move  •  Enter connect  •  Ctrl+E edit  •  d remove  •  Esc close"))

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Render(strings.Join(lines, "\n"))
}
