package app

import (
	"errors"
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ernestoponce27/db-tui/internal/db"
)

type editRowState uint8

const (
	editRowEditing editRowState = iota
	editRowConfirming
	editRowSaving
	editRowSuccess
	editRowFailed
)

const (
	editRowMinWidth = 56
	editRowMaxWidth = 78
	// editRowFieldRows is the vertical cost of one field: label, value, spacer.
	editRowFieldRows = 3
	// editRowChromeRows is everything around the field list: border, padding,
	// title, spacers, scroll hints and the help line.
	editRowChromeRows     = 11
	editRowDefaultVisible = 6
	editRowIndent         = "  "
	editRowIndentWidth    = 2
)

type editRowField struct {
	column   db.Column
	input    textinput.Model
	original any
	errorMsg string
}

// editable reports whether the user may type into the field. Identity columns
// are database-managed, so they are shown read-only.
func (f editRowField) editable() bool {
	return f.column.Identity == ""
}

// originalText is the field value as it was loaded, using the empty string for
// SQL NULL so it compares directly against the input.
func (f editRowField) originalText() string {
	if f.original == nil {
		return ""
	}
	return formatCell(f.original)
}

func (f editRowField) modified() bool {
	return f.editable() && f.input.Value() != f.originalText()
}

type editRowModal struct {
	tableName    string
	fields       []editRowField
	selected     int
	offset       int
	visible      int
	state        editRowState
	err          error
	setColumns   map[string]any
	whereColumns map[string]any
}

func newEditRowModal(table db.Table, columns []db.Column, row []any) editRowModal {
	styles := editRowInputStyles()
	fields := make([]editRowField, 0, len(columns))
	for i, col := range columns {
		var value any
		if i < len(row) {
			value = row[i]
		}
		input := textinput.New()
		input.Prompt = ""
		input.SetStyles(styles)
		input.SetWidth(editRowInputWidth(editRowMinWidth))
		if value != nil {
			input.SetValue(formatCell(value))
		} else {
			input.Placeholder = "NULL"
		}
		fields = append(fields, editRowField{
			column:   col,
			input:    input,
			original: value,
		})
	}
	return editRowModal{
		tableName: table.Name,
		fields:    fields,
		visible:   min(editRowDefaultVisible, max(1, len(fields))),
		state:     editRowEditing,
	}
}

// editRowInputStyles keeps the input on the modal's own background. Without an
// explicit background the padding a textinput adds to reach its width is drawn
// in the terminal's default colour, which shows up as a black bar.
func editRowInputStyles() textinput.Styles {
	styles := textinput.DefaultDarkStyles()
	background := colorModalBackground

	styles.Focused.Text = styles.Focused.Text.Background(background).Foreground(colorText)
	styles.Focused.Placeholder = styles.Focused.Placeholder.Background(background).Foreground(colorTextPlaceholder)
	styles.Focused.Prompt = styles.Focused.Prompt.Background(background)
	styles.Focused.Suggestion = styles.Focused.Suggestion.Background(background)
	styles.Blurred = styles.Focused

	styles.Cursor.Color = colorAccent
	return styles
}

// clamp fits the scrolling field list and the inputs to the terminal size.
func (m *editRowModal) clamp(layout appLayout) {
	fits := (layout.height - editRowChromeRows) / editRowFieldRows
	m.visible = max(1, min(len(m.fields), fits))
	width := editRowInputWidth(editRowContentWidth(layout.width))
	for i := range m.fields {
		m.fields[i].input.SetWidth(width)
	}
	m.ensureFieldVisible(m.visible)
}

// focusInitial selects the first editable field and focuses its input. Without
// it every key press is discarded, because a blurred textinput ignores input.
func (m *editRowModal) focusInitial() tea.Cmd {
	first := m.nextEditable(0, 1)
	if first < 0 {
		return nil
	}
	m.selected = first
	m.ensureFieldVisible(m.visible)
	return m.fields[m.selected].input.Focus()
}

// nextEditable returns the index of the first editable field at or after index,
// walking in the direction of delta, or -1 when there is none.
func (m editRowModal) nextEditable(index, delta int) int {
	for i := index; i >= 0 && i < len(m.fields); i += delta {
		if m.fields[i].editable() {
			return i
		}
	}
	return -1
}

func (m *editRowModal) focus(delta int) tea.Cmd {
	next := m.nextEditable(m.selected+delta, delta)
	if next < 0 {
		return nil
	}
	m.fields[m.selected].input.Blur()
	m.selected = next
	m.ensureFieldVisible(m.visible)
	return m.fields[m.selected].input.Focus()
}

func (m editRowModal) update(msg tea.Msg) (editRowModal, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch m.state {
	case editRowEditing:
		return m.updateEditing(key)
	case editRowConfirming:
		switch key.String() {
		case "enter":
			return m.confirmSave()
		case "esc":
			m.state = editRowEditing
		}
	case editRowSuccess, editRowFailed:
		switch key.String() {
		case "enter", "esc":
			return m, func() tea.Msg { return editRowCancelMsg{} }
		}
	}
	return m, nil
}

func (m editRowModal) updateEditing(key tea.KeyPressMsg) (editRowModal, tea.Cmd) {
	switch key.String() {
	case "esc":
		return m, func() tea.Msg { return editRowCancelMsg{} }
	case "enter":
		return m.prepareSave()
	case "up", "shift+tab":
		return m, m.focus(-1)
	case "down", "tab":
		return m, m.focus(1)
	default:
		if m.selected >= len(m.fields) || !m.fields[m.selected].editable() {
			return m, nil
		}
		field := &m.fields[m.selected]
		var cmd tea.Cmd
		field.input, cmd = field.input.Update(key)
		return m, cmd
	}
}

func (m *editRowModal) ensureFieldVisible(visible int) {
	if visible < 1 {
		visible = 1
	}
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visible {
		m.offset = m.selected - visible + 1
	}
}

func (m editRowModal) prepareSave() (editRowModal, tea.Cmd) {
	hasPK := false
	for _, f := range m.fields {
		if f.column.IsPrimaryKey {
			hasPK = true
			break
		}
	}
	if !hasPK {
		m.state = editRowFailed
		m.err = errors.New("cannot edit row: table has no primary key")
		return m, nil
	}

	// Validate first so every offending field is flagged in one pass.
	invalid := false
	for i := range m.fields {
		f := &m.fields[i]
		f.errorMsg = ""
		if !f.editable() {
			continue
		}
		if f.column.NotNull && f.column.Default == "" && strings.TrimSpace(f.input.Value()) == "" {
			f.errorMsg = "NOT NULL: a value is required"
			invalid = true
		}
	}
	if invalid {
		return m, nil
	}

	setColumns := make(map[string]any)
	whereColumns := make(map[string]any)

	for _, f := range m.fields {
		// The WHERE clause always uses the primary key values the row was
		// loaded with, so it cannot update an ambiguous matching row.
		if f.column.IsPrimaryKey {
			whereColumns[f.column.Name] = f.original
		}
		if !f.modified() {
			continue
		}
		value := f.input.Value()
		if strings.TrimSpace(value) == "" {
			setColumns[f.column.Name] = nil
		} else {
			setColumns[f.column.Name] = value
		}
	}

	if len(setColumns) == 0 {
		return m, func() tea.Msg { return editRowCancelMsg{} }
	}
	if len(whereColumns) == 0 {
		m.state = editRowFailed
		m.err = errors.New("cannot edit row: table has no primary key")
		return m, nil
	}

	m.setColumns = setColumns
	m.whereColumns = whereColumns
	m.state = editRowConfirming
	return m, nil
}

func (m editRowModal) confirmSave() (editRowModal, tea.Cmd) {
	m.state = editRowSaving
	return m, func() tea.Msg {
		return editRowSaveMsg{table: db.Table{Name: m.tableName}, setColumns: m.setColumns, whereColumns: m.whereColumns}
	}
}

// editRowStyles keeps every span on the modal's own background so no segment
// falls back to the terminal default and streaks the dialog.
type editRowStyles struct {
	base     lipgloss.Style
	title    lipgloss.Style
	accent   lipgloss.Style
	name     lipgloss.Style
	dim      lipgloss.Style
	value    lipgloss.Style
	fail     lipgloss.Style
	readOnly lipgloss.Style
}

func newEditRowStyles() editRowStyles {
	base := lipgloss.NewStyle().Background(colorModalBackground)
	return editRowStyles{
		base:     base,
		title:    base.Bold(true).Foreground(colorTitle),
		accent:   base.Foreground(colorAccent),
		name:     base.Foreground(colorText),
		dim:      base.Foreground(colorTextMuted),
		value:    base.Foreground(colorText),
		fail:     base.Foreground(colorError),
		readOnly: base.Foreground(colorTextPlaceholder).Italic(true),
	}
}

func (s editRowStyles) container(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Background(colorModalBackground)
}

func editRowContentWidth(width int) int {
	return min(editRowMaxWidth, max(editRowMinWidth, width-12))
}

func editRowInputWidth(contentWidth int) int {
	return max(10, contentWidth-editRowIndentWidth)
}

func (m editRowModal) view(layout appLayout) string {
	switch m.state {
	case editRowEditing:
		return m.viewEditing(layout)
	case editRowConfirming:
		return m.viewConfirmation(layout)
	case editRowSaving:
		return m.viewStatus(layout, "Saving…", colorAccent)
	case editRowSuccess:
		return m.viewStatus(layout, "✓ Row updated", colorAccent)
	case editRowFailed:
		return m.viewStatus(layout, "✕ "+sanitizeText(m.err.Error()), colorError)
	}
	return ""
}

func (m editRowModal) viewEditing(layout appLayout) string {
	s := newEditRowStyles()
	contentWidth := editRowContentWidth(layout.width)
	inputWidth := editRowInputWidth(contentWidth)

	lines := []string{
		s.title.Render("Edit row") + s.dim.Render("  ·  ") + s.accent.Render(sanitizeText(m.tableName)),
		"",
	}

	if m.offset > 0 {
		lines = append(lines, s.dim.Render(fmt.Sprintf("  ↑ %d more above", m.offset)))
	}

	last := min(m.offset+m.visible, len(m.fields))
	for i := m.offset; i < last; i++ {
		if i > m.offset {
			lines = append(lines, "")
		}
		lines = append(lines, m.viewField(s, i, contentWidth, inputWidth)...)
	}

	if last < len(m.fields) {
		lines = append(lines, s.dim.Render(fmt.Sprintf("  ↓ %d more below", len(m.fields)-last)))
	}

	lines = append(lines,
		"",
		s.dim.Render("↑/↓ or Tab move  •  Enter review  •  Esc cancel"),
	)

	return s.container(contentWidth).Render(strings.Join(lines, "\n"))
}

func (m editRowModal) viewConfirmation(layout appLayout) string {
	s := newEditRowStyles()
	contentWidth := editRowContentWidth(layout.width)

	lines := []string{
		s.title.Render("Edit row") + s.dim.Render("  ·  ") + s.accent.Render(sanitizeText(m.tableName)),
		"",
		s.base.Render("Save changes to this row?"),
		"",
		s.dim.Render("Enter confirm  •  Esc back"),
	}
	return s.container(contentWidth).Render(strings.Join(lines, "\n"))
}

// viewField renders the column name with its value below.
func (m editRowModal) viewField(s editRowStyles, index, contentWidth, inputWidth int) []string {
	f := m.fields[index]
	focused := index == m.selected

	marker := s.base.Render("  ")
	nameStyle := s.name
	if focused {
		marker = s.accent.Bold(true).Render("▸ ")
		nameStyle = s.accent.Bold(true)
	}

	nameWidth := max(1, contentWidth-editRowIndentWidth)
	header := marker +
		nameStyle.Render(truncateLabel(f.column.Name, nameWidth))

	// Only the focused input is rendered as a widget. A blurred textinput still
	// draws its virtual cursor, and that cursor carries no background, so it
	// would show up as a black block at the end of every other value.
	value := s.base.Render(editRowIndent)
	switch {
	case !f.editable():
		value += s.readOnly.Width(inputWidth).Render(truncateLabel(f.input.Value(), inputWidth))
	case focused:
		value += s.base.Width(inputWidth).Render(f.input.View())
	case f.input.Value() == "":
		value += s.dim.Width(inputWidth).Render(truncateLabel(f.input.Placeholder, inputWidth))
	default:
		value += s.value.Width(inputWidth).Render(truncateLabel(f.input.Value(), inputWidth))
	}

	lines := []string{header, value}
	if f.errorMsg != "" {
		lines = append(lines, s.base.Render(editRowIndent)+
			s.fail.Width(inputWidth).Render(truncateLabel("✕ "+f.errorMsg, inputWidth)))
	}
	return lines
}

func (m editRowModal) viewStatus(layout appLayout, message string, messageColor color.Color) string {
	s := newEditRowStyles()
	contentWidth := editRowContentWidth(layout.width)
	message = lipgloss.Wrap(sanitizeText(message), contentWidth, "")

	lines := []string{
		s.title.Render("Edit row") + s.dim.Render("  ·  ") + s.accent.Render(sanitizeText(m.tableName)),
		"",
		s.base.Foreground(messageColor).Render(message),
	}
	if m.state != editRowSaving {
		lines = append(lines, "", s.dim.Render("Enter or Esc continue"))
	}
	return s.container(contentWidth).Render(strings.Join(lines, "\n"))
}
