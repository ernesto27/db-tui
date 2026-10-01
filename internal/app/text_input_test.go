package app

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestTextInputsUseLightCursorColors(t *testing.T) {
	settings := newSettingsModal(100, "20m")
	connections := newConnectionsModal(config.Config{})
	actions := newActionsModal("", "connection")
	navigator := newNavigatorModel()
	row := newEditRowModal(db.Table{}, []db.Column{{Name: "Title"}}, []any{"value"})
	inputs := map[string]textinput.Model{
		"max page size":     settings.maxPageSize,
		"query timeout":     settings.queryExecutionTimeout,
		"connection search": connections.search,
		"rename connection": actions.renameInput,
		"navigator filter":  navigator.filter,
		"connection field":  newConnectionInput("host"),
		"row field":         row.fields[0].input,
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, colorAccent, input.Styles().Cursor.Color)
			assert.NotEqual(t, colorModalBackground, input.Styles().Cursor.Color)
			input.SetValue("value")
			input.CursorEnd()
			input.Focus()
			cursor := lipgloss.NewStyle().Foreground(colorAccent).Reverse(true).Render(" ")
			assert.Contains(t, input.View(), cursor)
		})
	}
	query := newQueryModel(newAppLayout(80, 24))
	assert.Equal(t, colorText, query.editor.Styles().Cursor.Color)
}

func TestModalTextInputViewOmitsBlurredCursor(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "settings value", value: "100", want: "100"},
		{name: "padded value", value: "100", width: 5, want: "100  "},
		{name: "empty value", width: 6, want: "Search"},
		{name: "truncated value", value: "long connection", width: 5, want: "long…"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := textinput.New()
			input.SetStyles(editRowInputStyles())
			input.Placeholder = "Search"
			input.SetWidth(test.width)
			input.SetValue(test.value)
			input.Focus()
			input.Blur()
			view := modalTextInputView(input)
			assert.Equal(t, test.want, ansi.Strip(view))
			style := input.Styles().Blurred.Text
			if test.value == "" {
				style = input.Styles().Blurred.Placeholder
			}
			if test.width > 0 {
				style = style.Width(test.width)
			}
			assert.Equal(t, style.Render(strings.TrimRight(test.want, " ")), view)
		})
	}
}
