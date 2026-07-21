package version

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if got := Version(); strings.TrimSpace(got) == "" {
		t.Error("Version() returned an empty version")
	}
}
