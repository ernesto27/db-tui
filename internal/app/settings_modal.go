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
	maxPageSize textinput.Model
	errorText   string
	saving      bool
}

type saveSettingsMsg struct {
	maxPageSize int
}

type cancelSettingsMsg struct{}

type settingsSavedMsg struct {
	maxPageSize int
	err         error
}

func newSettingsModal(maxPageSize int) settingsModal {
	input := textinput.New()
	input.Prompt = ""
	styles := editRowInputStyles()
	styles.Focused.Text = styles.Focused.Text.Foreground(colorAccent)
	styles.Blurred = styles.Focused
	styles.Cursor.Color = colorModalBackground
	input.SetStyles(styles)
	input.SetValue(strconv.Itoa(maxPageSize))
	return settingsModal{maxPageSize: input}
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
			return m, func() tea.Msg { return saveSettingsMsg{maxPageSize: maxPageSize} }
		case "esc":
			return m, func() tea.Msg { return cancelSettingsMsg{} }
		}
	}

	var command tea.Cmd
	m.maxPageSize, command = m.maxPageSize.Update(msg)
	return m, command
}

func parseMaxPageSize(value string) (int, error) {
	maxPageSize, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || maxPageSize < 1 {
		return 0, errors.New("max page size must be a positive whole number")
	}
	return maxPageSize, nil
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
		lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("Enter save  •  Esc cancel"))
	}

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}
