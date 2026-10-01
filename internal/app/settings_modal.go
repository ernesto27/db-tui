package app

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type settingsModal struct {
	maxPageSize           textinput.Model
	queryExecutionTimeout textinput.Model
	focusedInput          int
	errorText             string
	saving                bool
}

type saveSettingsMsg struct {
	maxPageSize           int
	queryExecutionTimeout string
}

type cancelSettingsMsg struct{}

type settingsSavedMsg struct {
	maxPageSize           int
	queryExecutionTimeout string
	err                   error
}

func newSettingsModal(maxPageSize int, queryExecutionTimeout string) settingsModal {
	newInput := func(value string) textinput.Model {
		input := textinput.New()
		input.Prompt = ""
		styles := editRowInputStyles()
		styles.Focused.Text = styles.Focused.Text.Foreground(colorAccent)
		styles.Blurred = styles.Focused
		input.SetStyles(styles)
		input.SetValue(value)
		return input
	}

	return settingsModal{
		maxPageSize:           newInput(strconv.Itoa(maxPageSize)),
		queryExecutionTimeout: newInput(queryExecutionTimeout),
	}
}

func (m *settingsModal) focusInput(index int) tea.Cmd {
	m.focusedInput = index
	m.maxPageSize.Blur()
	m.queryExecutionTimeout.Blur()
	if index == 0 {
		return m.maxPageSize.Focus()
	}
	return m.queryExecutionTimeout.Focus()
}

func (m settingsModal) update(msg tea.Msg) (settingsModal, tea.Cmd) {
	if m.saving {
		return m, nil
	}

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "tab":
			return m, (&m).focusInput((m.focusedInput + 1) % 2)
		case "shift+tab":
			return m, (&m).focusInput((m.focusedInput + 1) % 2)
		case "enter":
			maxPageSize, err := parseMaxPageSize(m.maxPageSize.Value())
			if err != nil {
				m.errorText = err.Error()
				return m, nil
			}
			queryExecutionTimeout, err := parseQueryExecutionTimeout(m.queryExecutionTimeout.Value())
			if err != nil {
				m.errorText = err.Error()
				return m, nil
			}
			m.errorText = ""
			return m, func() tea.Msg {
				return saveSettingsMsg{
					maxPageSize:           maxPageSize,
					queryExecutionTimeout: queryExecutionTimeout,
				}
			}
		case "esc":
			return m, func() tea.Msg { return cancelSettingsMsg{} }
		}
	}

	var command tea.Cmd
	if m.focusedInput == 0 {
		m.maxPageSize, command = m.maxPageSize.Update(msg)
	} else {
		m.queryExecutionTimeout, command = m.queryExecutionTimeout.Update(msg)
	}
	return m, command
}

func parseQueryExecutionTimeout(value string) (string, error) {
	value = strings.TrimSpace(value)
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return "", errors.New("query execution timeout must be a positive duration, such as 20m")
	}
	return value, nil
}

func parseMaxPageSize(value string) (int, error) {
	maxPageSize, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || maxPageSize < 1 {
		return 0, errors.New("max page size must be a positive whole number")
	}
	return maxPageSize, nil
}

func (m settingsModal) view(width int) string {
	modalWidth := min(64, max(48, width-8))
	labelStyle := lipgloss.NewStyle().
		Width(30).
		Foreground(colorAccent).
		Background(colorModalBackground)
	fieldView := func(label string, input textinput.Model) string {
		label = labelStyle.Render(label)
		value := lipgloss.NewStyle().
			Height(lipgloss.Height(label)).
			Background(colorModalBackground).
			Render(modalTextInputView(input))
		return lipgloss.JoinHorizontal(lipgloss.Top, label, value)
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Settings"),
		"",
		fieldView("Max page size", m.maxPageSize),
		fieldView("Query timeout: 20m", m.queryExecutionTimeout),
	}
	if m.errorText != "" {
		lines = append(lines, "", lipgloss.NewStyle().
			Foreground(colorWarningForeground).
			Background(colorWarningBackground).
			Width(modalWidth-6).
			Padding(0, 1).
			Render("✕ "+sanitizeText(m.errorText)))
	}
	if m.saving {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(colorAccent).Render("Saving settings…"))
	} else {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("Tab switch field  •  Enter save  •  Esc cancel"))
	}

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}
