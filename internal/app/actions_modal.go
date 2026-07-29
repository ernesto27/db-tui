package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type actionsModalState uint8

const (
	actionsSelecting actionsModalState = iota
	actionsRenameEditing
	actionsRenameSaving
	actionsRenameSuccess
	actionsRenameFailed
)

type actionsModal struct {
	state       actionsModalState
	tableName   string
	connName    string
	selected    int
	renameInput textinput.Model
	renameError string
	// action availability
	ddlAvailable    bool
	renameAvailable bool
}

type selectDDLActionMsg struct{}
type selectRenameActionMsg struct{}
type cancelActionsMsg struct{}
type submitRenameMsg struct {
	newName string
}

func newActionsModal(tableName, connName string) actionsModal {
	input := textinput.New()
	input.Prompt = ""
	input.SetWidth(40)

	m := actionsModal{
		state:           actionsSelecting,
		tableName:       tableName,
		connName:        connName,
		renameInput:     input,
		ddlAvailable:    tableName != "",
		renameAvailable: connName != "",
	}
	return m
}

func (m actionsModal) update(msg tea.Msg) (actionsModal, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch m.state {
	case actionsSelecting:
		return m.updateSelecting(key)
	case actionsRenameEditing:
		return m.updateRenameEditing(key)
	case actionsRenameSuccess, actionsRenameFailed:
		switch key.String() {
		case "enter", "esc":
			m.state = actionsSelecting
			return m, nil
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m actionsModal) updateSelecting(key tea.KeyPressMsg) (actionsModal, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < m.actionCount()-1 {
			m.selected++
		}
	case "enter":
		return m.selectAction()
	case "esc":
		return m, func() tea.Msg { return cancelActionsMsg{} }
	}
	return m, nil
}

func (m actionsModal) actionCount() int {
	count := 0
	if m.ddlAvailable {
		count++
	}
	if m.renameAvailable {
		count++
	}
	return count
}

func (m actionsModal) selectAction() (actionsModal, tea.Cmd) {
	idx := 0
	if m.ddlAvailable {
		if m.selected == idx {
			return m, func() tea.Msg { return selectDDLActionMsg{} }
		}
		idx++
	}
	if m.renameAvailable {
		if m.selected == idx {
			return m, func() tea.Msg { return selectRenameActionMsg{} }
		}
	}
	return m, nil
}

func (m *actionsModal) updateRenameEditing(key tea.KeyPressMsg) (actionsModal, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.state = actionsSelecting
		m.renameError = ""
		return *m, nil
	case "enter":
		return m.submitRename()
	default:
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(key)
		return *m, cmd
	}
}

func (m actionsModal) submitRename() (actionsModal, tea.Cmd) {
	trimmed := strings.TrimSpace(m.renameInput.Value())
	if trimmed == "" {
		m.renameError = "name must not be empty"
		return m, nil
	}
	if trimmed == m.connName {
		// No change, return to selection.
		m.state = actionsSelecting
		m.renameError = ""
		return m, nil
	}
	// Will emit rename command - handled by updateActionsModal.
	return m, func() tea.Msg { return submitRenameMsg{newName: trimmed} }
}

func (m actionsModal) view(width int) string {
	switch m.state {
	case actionsSelecting:
		return m.viewSelecting(width)
	case actionsRenameEditing:
		return m.viewRenameEditing(width)
	case actionsRenameSaving:
		return m.viewRenameSaving(width)
	case actionsRenameSuccess:
		return m.viewRenameSuccess(width)
	case actionsRenameFailed:
		return m.viewRenameFailed(width)
	default:
		return ""
	}
}

func (m actionsModal) viewSelecting(width int) string {
	modalWidth := min(56, max(40, width-8))
	boldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)

	lines := []string{
		boldStyle.Render("Actions"),
		"",
	}

	idx := 0
	if m.ddlAvailable {
		prefix := "  "
		style := normalStyle
		if m.selected == idx {
			prefix = "> "
			style = selectedStyle
		}
		lines = append(lines, prefix+style.Render("View DDL for "+sanitizeText(m.tableName)))
		idx++
	}
	if m.renameAvailable {
		prefix := "  "
		style := normalStyle
		if m.selected == idx {
			prefix = "> "
			style = selectedStyle
		}
		lines = append(lines, prefix+style.Render("Rename connection \""+sanitizeText(m.connName)+"\""))
	}

	lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("↑/↓ or j/k move  •  Enter select  •  Esc close"))

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameEditing(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Rename connection"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("New name:"),
		m.renameInput.View(),
		"",
	}
	if m.renameError != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("✕ "+sanitizeText(m.renameError)))
		lines = append(lines, "")
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Enter confirm  •  Esc back"))
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameSaving(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Rename connection"),
		"",
		"New name: " + sanitizeText(m.renameInput.Value()),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("Saving…"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameSuccess(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Rename connection"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("✓ Connection renamed"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Enter or Esc continue"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameFailed(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Rename connection"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("✕ " + sanitizeText(m.renameError)),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Enter or Esc continue"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("235")).
		Render(strings.Join(lines, "\n"))
}
