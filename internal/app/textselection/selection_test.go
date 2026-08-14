package textselection

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestSelectionExtractsForwardAndReverseMultilineRanges(t *testing.T) {
	rendered := "012345\nabcdef\nUVWXYZ"
	region := Region{Left: 1, Right: 4, Top: 0, Bottom: 2}

	for _, test := range []struct {
		name  string
		start Point
		end   Point
	}{
		{name: "forward", start: Point{X: 2, Y: 0}, end: Point{X: 3, Y: 2}},
		{name: "reverse", start: Point{X: 3, Y: 2}, end: Point{X: 2, Y: 0}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var selection Selection
			assert.True(t, selection.Start(test.start, region))
			assert.True(t, selection.Update(test.end))
			assert.True(t, selection.Release(test.end))
			assert.Equal(t, "234\nbcde\nVWX", selection.Text(rendered))
		})
	}
}

func TestSelectionClampsToRegionAndIgnoresClickWithoutDrag(t *testing.T) {
	region := Region{Left: 1, Right: 3, Top: 1, Bottom: 1}
	var selection Selection

	assert.True(t, selection.Start(Point{X: 2, Y: 1}, region))
	assert.False(t, selection.Release(Point{X: 2, Y: 1}))
	assert.False(t, selection.Active())

	assert.True(t, selection.Start(Point{X: 2, Y: 1}, region))
	assert.True(t, selection.Update(Point{X: 100, Y: 100}))
	assert.True(t, selection.Release(Point{X: 100, Y: 100}))
	assert.Equal(t, "cd", selection.Text("0000\nabcd\n0000"))
}

func TestSelectionRendersHighlightWithoutChangingPlainText(t *testing.T) {
	rendered := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("hello")
	var selection Selection
	assert.True(t, selection.Start(Point{X: 1, Y: 0}, Region{Left: 0, Right: 4, Top: 0, Bottom: 0}))
	assert.True(t, selection.Release(Point{X: 3, Y: 0}))

	highlighted := selection.Render(rendered, lipgloss.NewStyle().Background(lipgloss.Color("153")))

	assert.NotEqual(t, rendered, highlighted)
	assert.Equal(t, ansi.Strip(rendered), ansi.Strip(highlighted))
	assert.Equal(t, "ell", selection.Text(rendered))
}
