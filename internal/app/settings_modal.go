package app

import (
	"errors"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type settingsModal struct {
	maxPageSize  textinput.Model
	queryTimeout textinput.Model
	errorText    string
	saving       bool
}

type saveSettingsMsg struct {
	maxPageSize         int
	queryTimeoutSeconds int
}

type cancelSettingsMsg struct{}

type settingsSavedMsg struct {
	maxPageSize         int
	queryTimeoutSeconds int
	err                 error
}

func newSettingsModal(maxPageSize, queryTimeoutSeconds int) settingsModal {
	input := textinput.New()
	input.Prompt = ""
	styles := editRowInputStyles()
	styles.Focused.Text = styles.Focused.Text.Foreground(colorAccent)
	styles.Blurred = styles.Focused
	styles.Cursor.Color = colorModalBackground
	input.SetStyles(styles)
	input.SetValue(strconv.Itoa(maxPageSize))
	timeout := textinput.New()
	timeout.Prompt = ""
	timeout.SetStyles(styles)
	timeout.SetValue(strconv.Itoa(queryTimeoutSeconds))
	return settingsModal{maxPageSize: input, queryTimeout: timeout}
}

func (m settingsModal) update(msg tea.Msg) (settingsModal, tea.Cmd) {
	if m.saving {
		return m, nil
	}

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter":
			maxPageSize, err := parseMaxPageSize(m.maxPageSize.Value())
			if err != nil {
				m.errorText = err.Error()
				return m, nil
			}
			m.errorText = ""
			queryTimeoutSeconds, err := parseQueryTimeout(m.queryTimeout.Value())
			if err != nil {
				m.errorText = err.Error()
				return m, nil
			}
			return m, func() tea.Msg {
				return saveSettingsMsg{maxPageSize: maxPageSize, queryTimeoutSeconds: queryTimeoutSeconds}
			}
		case "esc":
			return m, func() tea.Msg { return cancelSettingsMsg{} }
		}
	}

	var command tea.Cmd
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "tab" {
		if m.maxPageSize.Focused() {
			m.maxPageSize.Blur()
			command = m.queryTimeout.Focus()
		} else {
			m.queryTimeout.Blur()
			command = m.maxPageSize.Focus()
		}
		return m, command
	}
	if m.queryTimeout.Focused() {
		m.queryTimeout, command = m.queryTimeout.Update(msg)
	} else {
		m.maxPageSize, command = m.maxPageSize.Update(msg)
	}
	return m, command
}

func parseMaxPageSize(value string) (int, error) {
	maxPageSize, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || maxPageSize < 1 {
		return 0, errors.New("max page size must be a positive whole number")
	}
	return maxPageSize, nil
}

func parseQueryTimeout(value string) (int, error) {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 1 {
		return 0, errors.New("query timeout must be a positive whole number of seconds")
	}
	return seconds, nil
}

func (m settingsModal) view(width int) string {
	modalWidth := min(56, max(40, width-8))
	labelStyle := lipgloss.NewStyle().
		Width(24).
		Foreground(colorAccent).
		Background(colorModalBackground)
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Settings"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Max page size"),
			m.maxPageSize.View(),
		),
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Query timeout (sec)"),
			m.queryTimeout.View(),
		),
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
		lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("Tab switch  •  Enter save  •  Esc cancel"))
	}

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}
