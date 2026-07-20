package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		wantDSN  string
		wantErr  string
	}{
		{
			name:     "valid configuration",
			contents: `{"postgresql":{"dsn":"postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable"}}`,
			wantDSN:  "postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable",
		},
		{
			name:     "malformed JSON",
			contents: `{"postgresql":`,
			wantErr:  "decode config",
		},
		{
			name:     "missing PostgreSQL settings",
			contents: `{}`,
			wantErr:  `config field "postgresql" is required`,
		},
		{
			name:     "blank DSN",
			contents: `{"postgresql":{"dsn":"  "}}`,
			wantErr:  `config field "postgresql.dsn" is required`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := useTemporaryHome(t)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), configDirectoryMode))
			require.NoError(t, os.WriteFile(path, []byte(test.contents), 0o600))

			config, err := Load()
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config.PostgreSQL)
			assert.Equal(t, test.wantDSN, config.PostgreSQL.DSN)
		})
	}
}

func useTemporaryHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, ".config", "db-tui", configFileName)
}
