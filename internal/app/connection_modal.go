package app

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
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

var connectionEngines = []string{db.EnginePostgreSQL, db.EngineMySQL, db.EngineOracle, db.EngineSQLite}

type connectionModal struct {
	inputs      [connectionInputCount]textinput.Model
	engineIndex int
	focused     connectionInput
	errorText   string
	connecting  bool
}

type submitConnectionMsg struct{}
type cancelConnectionMsg struct{}

func newConnectionModal(saved ConnectionSettings) connectionModal {
	modal := connectionModal{}
	modal.inputs[hostInput] = newConnectionInput("127.0.0.1")
	modal.inputs[databaseNameInput] = newConnectionInput("database name")
	modal.inputs[portInput] = newConnectionInput("5432 or 3306")
	modal.inputs[usernameInput] = newConnectionInput("username")
	modal.inputs[passwordInput] = newConnectionInput("password")
	modal.inputs[passwordInput].EchoMode = textinput.EchoPassword
	modal.inputs[dsnInput] = newConnectionInput("engine-specific DSN")
	engine, err := saved.normalizedEngine()
	if err != nil {
		engine = db.EnginePostgreSQL
	}
	modal.setEngine(engine)
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
		if m.focused == engineInput {
			switch key.String() {
			case "left", "up":
				m.selectEngine(-1)
				return m, nil
			case "right", "down":
				m.selectEngine(1)
				return m, nil
			}
		}

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

	if m.focused == engineInput {
		return m, nil
	}

	var command tea.Cmd
	m.inputs[m.focused], command = m.inputs[m.focused].Update(msg)
	return m, command
}

func (m *connectionModal) focus(delta int) tea.Cmd {
	if m.focused != engineInput {
		m.inputs[m.focused].Blur()
	}
	inputs := connectionInputsForEngine(m.engine())
	current := 0
	for index, input := range inputs {
		if input == m.focused {
			current = index
			break
		}
	}
	m.focused = inputs[(current+delta+len(inputs))%len(inputs)]
	if m.focused == engineInput {
		return nil
	}
	return m.inputs[m.focused].Focus()
}

func (m connectionModal) engine() string {
	return connectionEngines[m.engineIndex]
}

func (m *connectionModal) selectEngine(delta int) {
	previousEngine := m.engine()
	m.engineIndex = (m.engineIndex + delta + len(connectionEngines)) % len(connectionEngines)
	m.setDSNPlaceholder()
	previousPort := defaultPortForEngine(previousEngine)
	if port := m.inputs[portInput].Value(); port == "" || port == previousPort {
		m.inputs[portInput].SetValue(defaultPortForEngine(m.engine()))
	}
}

func (m *connectionModal) setEngine(engine string) {
	for index, candidate := range connectionEngines {
		if engine == candidate {
			m.engineIndex = index
			break
		}
	}
	m.setDSNPlaceholder()
	if m.inputs[portInput].Value() == "" {
		m.inputs[portInput].SetValue(defaultPortForEngine(m.engine()))
	}
}

func (m *connectionModal) setDSNPlaceholder() {
	if m.engine() == db.EngineSQLite {
		m.inputs[dsnInput].Placeholder = "path/to/database.db"
		return
	}
	if m.engine() == db.EngineOracle {
		m.inputs[dsnInput].Placeholder = "oracle://user:password@host:1521/service"
		return
	}
	m.inputs[dsnInput].Placeholder = "engine-specific DSN"
}

func defaultPortForEngine(engine string) string {
	if engine == db.EngineMySQL {
		return "3306"
	}
	if engine == db.EngineOracle {
		return "1521"
	}
	if engine == db.EngineSQLite {
		return ""
	}
	return "5432"
}

func connectionInputsForEngine(engine string) []connectionInput {
	if engine == db.EngineSQLite {
		return []connectionInput{engineInput, dsnInput}
	}
	return []connectionInput{engineInput, hostInput, databaseNameInput, portInput, usernameInput, passwordInput, dsnInput}
}

func (m connectionModal) connectionSettings() (ConnectionSettings, error) {
	if m.engine() == db.EngineSQLite {
		settings := ConnectionSettings{
			Engine: db.EngineSQLite,
			DSN:    strings.TrimSpace(m.inputs[dsnInput].Value()),
		}
		_, err := settings.connectionDSN()
		return settings, err
	}
	if dsn := strings.TrimSpace(m.inputs[dsnInput].Value()); dsn != "" {
		settings := ConnectionSettings{
			Engine: m.engine(),
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
		Engine:       m.engine(),
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
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Connect to database"),
		"",
	}
	engineStyle := fieldStyle
	if m.focused == engineInput {
		engineStyle = engineStyle.Foreground(colorTitle).Bold(true)
	}
	lines = append(lines,
		labelStyle.Render("Engine"),
		engineStyle.Render("◀ "+engineDisplayName(m.engine())+" ▶"),
	)
	fields := []struct {
		label string
		input connectionInput
	}{
		{"Host", hostInput},
		{"Database name", databaseNameInput},
		{"Port", portInput},
		{"Username", usernameInput},
		{"Password", passwordInput},
		{"DSN (optional)", dsnInput},
	}
	if m.engine() == db.EngineSQLite {
		fields = []struct {
			label string
			input connectionInput
		}{{"Database file", dsnInput}}
	}
	for _, field := range fields {
		lines = append(lines, labelStyle.Render(field.label), fieldStyle.Render(m.inputs[field.input].View()))
	}
	if m.errorText != "" {
		lines = append(lines, "", lipgloss.NewStyle().
			Foreground(colorWarningForeground).
			Background(colorWarningBackground).
			Width(modalWidth-6).
			Padding(0, 1).
			Render("✕ "+sanitizeText(m.errorText)))
	}
	if m.connecting {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(colorAccent).Render("Checking connection…"))
	} else {
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(colorTextMuted).Render("Tab move  •  ←/→ select engine  •  Enter connect  •  Esc cancel"),
		)
	}

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}
