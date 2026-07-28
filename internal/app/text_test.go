package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestFormatCell(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "SQL NULL", value: nil, want: "NULL"},
		{name: "binary data", value: []byte("hello"), want: "BINARY DATA"},
		{name: "number", value: 42, want: "42"},
		{name: "control character", value: "safe\x1b[31m", want: "safe�[31m"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, formatCell(test.value))
		})
	}
}

func TestSanitizeText(t *testing.T) {
	assert.Equal(t, "printable 世界", sanitizeText("printable 世界"))
	assert.Equal(t, "a�����b", sanitizeText("a\n\t\r\x1b\x00b"))
}

func TestTruncateLabelReturnsEmptyForNonPositiveWidth(t *testing.T) {
	assert.Empty(t, truncateLabel("label", 0))
	assert.Empty(t, truncateLabel("label", -1))
}

func TestTruncateLabelKeepsShortLabel(t *testing.T) {
	assert.Equal(t, "Album", truncateLabel("Album", 10))
}

func TestTruncateLabelTruncatesASCIIWithEllipsis(t *testing.T) {
	got := truncateLabel("very-long-label", 8)

	assert.LessOrEqual(t, ansi.StringWidth(got), 8)
	assert.True(t, strings.HasSuffix(got, "…"))
}

func TestTruncateLabelUsesDisplayWidthForUnicode(t *testing.T) {
	got := truncateLabel("界界界", 4)

	assert.LessOrEqual(t, ansi.StringWidth(got), 4)
	assert.True(t, strings.HasSuffix(got, "…"))
}

func TestTruncateLabelSanitizesBeforeTruncating(t *testing.T) {
	got := truncateLabel("safe\x1b[31m", 20)

	assert.NotContains(t, got, "\x1b")
	assert.Equal(t, "safe�[31m", got)
}
