package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestEditRowFieldDoesNotRenderDataType(t *testing.T) {
	modal := newEditRowModal(
		db.Table{Name: "Album"},
		[]db.Column{{Name: "AlbumId", DataType: "int4"}},
		[]any{11},
	)
	contentWidth := 60
	inputWidth := editRowInputWidth(contentWidth)

	lines := modal.viewField(newEditRowStyles(), 0, contentWidth, inputWidth)

	assert.NotContains(t, ansi.Strip(strings.Join(lines, "\n")), "int4")
}
