package app

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type connectionInput uint8

const (
	engineInput connectionInput = iota
	hostInput
	databaseNameInput
	portInput
	usernameInput
	passwordInput
	dsnInput
	connectionInputCount
)

const connectionModalInputWidth = 42

type connectionModal struct {
	inputs     [connectionInputCount]textinput.Model
	focused    connectionInput
	errorText  string
	connecting bool
}

type submitConnectionMsg struct{}
type cancelConnectionMsg struct{}

func newConnectionModal(saved ConnectionSettings) connectionModal {
	modal := connectionModal{}
	modal.inputs[engineInput] = newConnectionInput("postgres or mysql")
	modal.inputs[hostInput] = newConnectionInput("127.0.0.1")
	modal.inputs[databaseNameInput] = newConnectionInput("database name")
	modal.inputs[portInput] = newConnectionInput("5432 or 3306")
	modal.inputs[usernameInput] = newConnectionInput("username")
	modal.inputs[passwordInput] = newConnectionInput("password")
	modal.inputs[passwordInput].EchoMode = textinput.EchoPassword
	modal.inputs[dsnInput] = newConnectionInput("engine-specific DSN")
	engine, err := saved.normalizedEngine()
	if err != nil {
		engine = saved.Engine
	}
	modal.inputs[engineInput].SetValue(engine)
	modal.inputs[hostInput].SetValue(saved.Host)
	modal.inputs[databaseNameInput].SetValue(saved.DatabaseName)
	if saved.Port > 0 {
		modal.inputs[portInput].SetValue(strconv.Itoa(saved.Port))
	}
	modal.inputs[usernameInput].SetValue(saved.Username)
	modal.inputs[passwordInput].SetValue(saved.Password)
	modal.inputs[dsnInput].SetValue(saved.DSN)
	return modal
}

func newConnectionInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.SetWidth(connectionModalInputWidth)
	return input
}

func (m connectionModal) update(msg tea.Msg) (connectionModal, tea.Cmd) {
	if m.connecting {
		return m, nil
	}

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "tab":
			return m, m.focus(1)
		case "shift+tab":
			return m, m.focus(-1)
		case "enter":
			return m, func() tea.Msg { return submitConnectionMsg{} }
		case "esc":
			return m, func() tea.Msg { return cancelConnectionMsg{} }
		}
	}

	var command tea.Cmd
	m.inputs[m.focused], command = m.inputs[m.focused].Update(msg)
	return m, command
}

func (m *connectionModal) focus(delta int) tea.Cmd {
	m.inputs[m.focused].Blur()
	m.focused = connectionInput((int(m.focused) + delta + int(connectionInputCount)) % int(connectionInputCount))
	return m.inputs[m.focused].Focus()
}

func (m connectionModal) connectionSettings() (ConnectionSettings, error) {
	if dsn := strings.TrimSpace(m.inputs[dsnInput].Value()); dsn != "" {
		settings := ConnectionSettings{
			Engine: strings.TrimSpace(m.inputs[engineInput].Value()),
			DSN:    dsn,
		}
		_, err := settings.normalizedEngine()
		return settings, err
	}

	port, err := strconv.Atoi(strings.TrimSpace(m.inputs[portInput].Value()))
	if err != nil {
		port = 0
	}
	settings := ConnectionSettings{
		Engine:       strings.TrimSpace(m.inputs[engineInput].Value()),
		Host:         strings.TrimSpace(m.inputs[hostInput].Value()),
		Port:         port,
		DatabaseName: strings.TrimSpace(m.inputs[databaseNameInput].Value()),
		Username:     strings.TrimSpace(m.inputs[usernameInput].Value()),
		Password:     m.inputs[passwordInput].Value(),
	}
	_, err = settings.connectionDSN()
	return settings, err
}

func (m connectionModal) view(width int) string {
	modalWidth := min(64, max(52, width-8))
	fieldStyle := lipgloss.NewStyle().Width(modalWidth - 6)
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Connect to database"),
		"",
	}
	for _, field := range []struct {
		label string
		input connectionInput
	}{
		{"Engine", engineInput},
		{"Host", hostInput},
		{"Database name", databaseNameInput},
		{"Port", portInput},
		{"Username", usernameInput},
		{"Password", passwordInput},
		{"DSN (optional)", dsnInput},
	} {
		lines = append(lines, labelStyle.Render(field.label), fieldStyle.Render(m.inputs[field.input].View()))
	}
	if m.errorText != "" {
		lines = append(lines, "", lipgloss.NewStyle().
			Foreground(lipgloss.Color("210")).
			Background(lipgloss.Color("52")).
			Width(modalWidth-6).
			Padding(0, 1).
			Render("✕ "+sanitizeText(m.errorText)))
	}
	if m.connecting {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("Checking connection…"))
	} else {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Tab/Shift+Tab move  •  Enter connect  •  Esc cancel"),
		)
	}

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}
