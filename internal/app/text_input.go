package app

import "charm.land/bubbles/v2/textinput"

// modalTextInputView renders plain-text inputs on their styled background,
// including the virtual cursor's blank cell while blinking. Blurred inputs
// render without the virtual cursor, which otherwise leaves a black block.
func modalTextInputView(input textinput.Model) string {
	styles := input.Styles()
	if input.Focused() {
		return styles.Focused.Text.Render(input.View())
	}
	value := input.Value()
	style := styles.Blurred.Text
	if value == "" {
		value = input.Placeholder
		style = styles.Blurred.Placeholder
	}
	if input.Width() > 0 {
		value = truncateLabel(value, input.Width())
		style = style.Width(input.Width())
	}
	return style.Render(value)
}
