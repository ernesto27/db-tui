package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
)

type actionsModalState uint8

const (
	actionsSelecting actionsModalState = iota
	actionsRenameEditing
	actionsRenameSaving
	actionsRenameSuccess
	actionsRenameFailed
)

type environmentModal struct {
	selected  int
	saving    bool
	succeeded bool
	err       string
}

type actionsModal struct {
	state       actionsModalState
	tableName   string
	connName    string
	selected    int
	renameInput textinput.Model
	renameError string
	environment *environmentModal
	// action availability
	ddlAvailable         bool
	columnsAvailable     bool
	indexesAvailable     bool
	renameAvailable      bool
	environmentAvailable bool
}

type selectDDLActionMsg struct{}
type selectColumnsActionMsg struct{}
type selectIndexesActionMsg struct{}
type selectRenameActionMsg struct{}
type selectEnvironmentActionMsg struct{}
type cancelActionsMsg struct{}
type submitRenameMsg struct {
	newName string
}
type submitEnvironmentMsg struct {
	environment config.ConnectionEnvironment
}

func newActionsModal(tableName, connName string) actionsModal {
	input := textinput.New()
	input.Prompt = ""
	input.SetWidth(40)

	m := actionsModal{
		state:            actionsSelecting,
		tableName:        tableName,
		connName:         connName,
		renameInput:      input,
		ddlAvailable:     tableName != "",
		columnsAvailable: tableName != "",
		renameAvailable:  connName != "",
	}
	return m
}

func (m actionsModal) update(msg tea.Msg) (actionsModal, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if m.environment != nil {
		return m.updateEnvironment(key)
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
	if m.columnsAvailable {
		count++
	}
	if m.indexesAvailable {
		count++
	}
	if m.renameAvailable {
		count++
	}
	if m.environmentAvailable {
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
	if m.columnsAvailable {
		if m.selected == idx {
			return m, func() tea.Msg { return selectColumnsActionMsg{} }
		}
		idx++
	}
	if m.indexesAvailable {
		if m.selected == idx {
			return m, func() tea.Msg { return selectIndexesActionMsg{} }
		}
		idx++
	}
	if m.renameAvailable {
		if m.selected == idx {
			return m, func() tea.Msg { return selectRenameActionMsg{} }
		}
		idx++
	}
	if m.environmentAvailable {
		if m.selected == idx {
			return m, func() tea.Msg { return selectEnvironmentActionMsg{} }
		}
	}
	return m, nil
}

func (m actionsModal) updateEnvironment(key tea.KeyPressMsg) (actionsModal, tea.Cmd) {
	if m.environment.saving {
		return m, nil
	}
	if m.environment.succeeded || m.environment.err != "" {
		switch key.String() {
		case "enter", "esc":
			m.environment = nil
		}
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		if m.environment.selected > 0 {
			m.environment.selected--
		}
	case "down", "j":
		if m.environment.selected < len(connectionEnvironmentOptions)-1 {
			m.environment.selected++
		}
	case "enter":
		environment := connectionEnvironmentOptions[m.environment.selected].environment
		return m, func() tea.Msg { return submitEnvironmentMsg{environment: environment} }
	case "esc":
		m.environment = nil
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
	if m.environment != nil {
		return m.environment.view(width)
	}

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
	boldStyle := lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
	normalStyle := lipgloss.NewStyle().Foreground(colorTextMuted)
	selectedStyle := lipgloss.NewStyle().Foreground(colorTitle).Bold(true)

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
	if m.columnsAvailable {
		prefix := "  "
		style := normalStyle
		if m.selected == idx {
			prefix = "> "
			style = selectedStyle
		}
		lines = append(lines, prefix+style.Render("Inspect columns for "+sanitizeText(m.tableName)))
		idx++
	}
	if m.indexesAvailable {
		prefix := "  "
		style := normalStyle
		if m.selected == idx {
			prefix = "> "
			style = selectedStyle
		}
		lines = append(lines, prefix+style.Render("Inspect indexes for "+sanitizeText(m.tableName)))
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
		idx++
	}
	if m.environmentAvailable {
		prefix := "  "
		style := normalStyle
		if m.selected == idx {
			prefix = "> "
			style = selectedStyle
		}
		lines = append(lines, prefix+style.Render("Set connection environment…"))
	}

	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("↑/↓ or j/k move  •  Enter select  •  Esc close"))

	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameEditing(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Rename connection"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("New name:"),
		m.renameInput.View(),
		"",
	}
	if m.renameError != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorError).Render("✕ "+sanitizeText(m.renameError)))
		lines = append(lines, "")
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(colorTextMuted).Render("Enter confirm  •  Esc back"))
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameSaving(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Rename connection"),
		"",
		"New name: " + sanitizeText(m.renameInput.Value()),
		"",
		lipgloss.NewStyle().Foreground(colorAccent).Render("Saving…"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameSuccess(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Rename connection"),
		"",
		lipgloss.NewStyle().Foreground(colorAccent).Render("✓ Connection renamed"),
		"",
		lipgloss.NewStyle().Foreground(colorTextMuted).Render("Enter or Esc continue"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m actionsModal) viewRenameFailed(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Rename connection"),
		"",
		lipgloss.NewStyle().Foreground(colorError).Render("✕ " + sanitizeText(m.renameError)),
		"",
		lipgloss.NewStyle().Foreground(colorTextMuted).Render("Enter or Esc continue"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

type connectionEnvironmentOption struct {
	environment config.ConnectionEnvironment
	label       string
}

var connectionEnvironmentOptions = []connectionEnvironmentOption{
	{label: "No environment color"},
	{environment: config.ConnectionEnvironmentTesting, label: "Testing — green header"},
	{environment: config.ConnectionEnvironmentProduction, label: "Production — red header"},
}

func newEnvironmentModal(environment config.ConnectionEnvironment) environmentModal {
	modal := environmentModal{}
	for index, option := range connectionEnvironmentOptions {
		if option.environment == environment {
			modal.selected = index
			break
		}
	}
	return modal
}

func (m environmentModal) view(width int) string {
	if m.saving {
		return m.viewSaving(width)
	}
	if m.succeeded {
		return m.viewSuccess(width)
	}
	if m.err != "" {
		return m.viewFailed(width)
	}
	return m.viewSelecting(width)
}

func (m environmentModal) viewSelecting(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Set connection environment"),
		"",
	}
	for index, option := range connectionEnvironmentOptions {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(colorTextMuted)
		if m.selected == index {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(colorTitle).Bold(true)
		}
		lines = append(lines, prefix+style.Render(option.label))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorTextMuted).Render("↑/↓ or j/k move  •  Enter select  •  Esc back"))
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m environmentModal) viewSaving(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Set connection environment"),
		"",
		lipgloss.NewStyle().Foreground(colorAccent).Render("Saving…"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m environmentModal) viewSuccess(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Set connection environment"),
		"",
		lipgloss.NewStyle().Foreground(colorAccent).Render("✓ Connection environment updated"),
		"",
		lipgloss.NewStyle().Foreground(colorTextMuted).Render("Enter or Esc continue"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}

func (m environmentModal) viewFailed(width int) string {
	modalWidth := min(56, max(40, width-8))
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorTitle).Render("Set connection environment"),
		"",
		lipgloss.NewStyle().Foreground(colorError).Render("✕ " + sanitizeText(m.err)),
		"",
		lipgloss.NewStyle().Foreground(colorTextMuted).Render("Enter or Esc continue"),
	}
	return lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground).
		Render(strings.Join(lines, "\n"))
}
