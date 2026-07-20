package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     postgres.PostgreSQLConfig
		wantErr  string
	}{
		{
			name:     "valid configuration",
			contents: `{"postgresql":{"dsn":"postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable"}}`,
			want:     postgres.PostgreSQLConfig{DSN: "postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable"},
		},
		{
			name:     "valid form configuration",
			contents: `{"postgresql":{"host":"127.0.0.1","port":5433,"databaseName":"chinook","username":"db_tui","password":"secret"}}`,
			want: postgres.PostgreSQLConfig{
				Host:         "127.0.0.1",
				Port:         5433,
				DatabaseName: "chinook",
				Username:     "db_tui",
				Password:     "secret",
			},
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
			name:     "blank connection settings",
			contents: `{"postgresql":{"dsn":"  "}}`,
			wantErr:  `config field "postgresql.host" is required`,
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
			assert.Equal(t, test.want, *config.PostgreSQL)
		})
	}
}

func TestSavePostgreSQLConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   postgres.PostgreSQLConfig
		wantErr  string
		wantJSON string
	}{
		{
			name:     "writes PostgreSQL DSN",
			config:   postgres.PostgreSQLConfig{DSN: "postgres://db_tui:secret@127.0.0.1:5433/chinook?sslmode=disable"},
			wantJSON: `{"postgresql":{"dsn":"postgres://db_tui:secret@127.0.0.1:5433/chinook?sslmode=disable"}}`,
		},
		{
			name: "writes PostgreSQL form settings without DSN",
			config: postgres.PostgreSQLConfig{
				Host:         "127.0.0.1",
				Port:         5433,
				DatabaseName: "chinook",
				Username:     "db_tui",
				Password:     "secret",
			},
			wantJSON: `{"postgresql":{"host":"127.0.0.1","port":5433,"databaseName":"chinook","username":"db_tui","password":"secret"}}`,
		},
		{
			name:    "rejects blank connection settings",
			config:  postgres.PostgreSQLConfig{},
			wantErr: `config field "postgresql.host" is required`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := useTemporaryHome(t)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), configDirectoryMode))

			err := SavePostgreSQLConfig(test.config)
			if test.wantErr != "" {
				assert.ErrorContains(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			config, err := Load()
			require.NoError(t, err)
			require.NotNil(t, config.PostgreSQL)
			assert.Equal(t, test.config, *config.PostgreSQL)

			contents, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.JSONEq(t, test.wantJSON, string(contents))

			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(configFileMode), info.Mode().Perm())
		})
	}
}

func useTemporaryHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, ".config", "db-tui", configFileName)
}
