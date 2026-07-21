// Package version provides the db-tui release version.
package version

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed version.json
var manifest []byte

type releaseManifest struct {
	Version string `json:"version"`
}

// Version returns the release version defined in version.json.
func Version() string {
	var release releaseManifest
	if err := json.Unmarshal(manifest, &release); err != nil {
		return "dev"
	}

	if version := strings.TrimSpace(release.Version); version != "" {
		return version
	}
	return "dev"
}
