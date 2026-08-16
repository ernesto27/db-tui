package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     Config
		wantErr  string
	}{
		{
			name:     "valid configuration",
			contents: `{"connections":[{"name":"local","engine":"postgres","settings":{"hostname":"127.0.0.1","database":"chinook","username":"db_tui","password":"secret","port":"5433","dsn":""},"status":true}]}`,
			want: Config{MaxPageSize: db.MaxPageSize, Connections: []Connection{{
				Name:   "local",
				Engine: "postgres",
				Settings: Settings{
					Hostname: "127.0.0.1",
					Database: "chinook",
					Username: "db_tui",
					Password: "secret",
					Port:     "5433",
				},
				Status: true,
			}}},
		},
		{
			name:     "empty configuration",
			contents: `{}`,
			want:     Config{MaxPageSize: db.MaxPageSize},
		},
		{
			name:     "malformed JSON",
			contents: `{"connections":`,
			wantErr:  "decode config",
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
			assert.Equal(t, test.want, config)
		})
	}
}

func TestLoadPreservesMaxPageSizeAboveDefault(t *testing.T) {
	path := useTemporaryHome(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), configDirectoryMode))
	require.NoError(t, os.WriteFile(path, []byte(`{"maxPageSize":250}`), 0o600))

	config, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 250, config.MaxPageSize)
}

func TestSavePreservesMaxPageSizeAboveDefault(t *testing.T) {
	path := useTemporaryHome(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), configDirectoryMode))
	config := Config{MaxPageSize: 250}

	require.NoError(t, config.Save())
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"maxPageSize":250}`, string(contents))
}

func TestConfigSaveConnection(t *testing.T) {
	path := useTemporaryHome(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), configDirectoryMode))

	config := Config{
		Connections: []Connection{{Name: "local", Engine: "postgres"}},
	}
	connection := Connection{
		Name:   "production",
		Engine: "postgres",
		Settings: Settings{
			Hostname: "db.example.com",
			Database: "chinook",
			Username: "db_tui",
			Port:     "5432",
		},
	}

	require.NoError(t, config.saveConnection(connection))
	assert.Equal(t, []Connection{
		{Name: "local", Engine: "postgres"},
		connection,
	}, config.Connections)

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{
		"maxPageSize": %d,
		"connections": [
			{"name":"local","engine":"postgres","settings":{"hostname":"","database":"","username":"","password":"","port":"","dsn":""},"status":false},
			{"name":"production","engine":"postgres","settings":{"hostname":"db.example.com","database":"chinook","username":"db_tui","password":"","port":"5432","dsn":""},"status":false}
		]
	}`, db.MaxPageSize), string(contents))
}

func TestConfigSave(t *testing.T) {
	path := useTemporaryHome(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), configDirectoryMode))

	config := Config{Connections: []Connection{{
		Name:   "local",
		Engine: "postgres",
		Settings: Settings{
			Hostname: "127.0.0.1",
			Database: "chinook",
			Username: "db_tui",
			Port:     "5433",
		},
	}}}
	config.Connections[0].Settings.Username = "updated_user"

	require.NoError(t, config.Save())

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{
		"maxPageSize": %d,
		"connections": [
			{"name":"local","engine":"postgres","settings":{"hostname":"127.0.0.1","database":"chinook","username":"updated_user","password":"","port":"5433","dsn":""},"status":false}
		]
	}`, db.MaxPageSize), string(contents))
}

func TestConfigPersistsSQLiteDatabasePathInExistingDSNField(t *testing.T) {
	path := useTemporaryHome(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), configDirectoryMode))

	config := Config{Connections: []Connection{{
		Name:   "Employee",
		Engine: "sqlite",
		Settings: Settings{
			DSN: "docker/sqlite/employee.db",
		},
	}}}

	require.NoError(t, config.Save())
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"maxPageSize":%d,"connections":[{"name":"Employee","engine":"sqlite","settings":{"hostname":"","database":"","username":"","password":"","port":"","dsn":"docker/sqlite/employee.db"},"status":false}]}`, db.MaxPageSize), string(contents))
}

func useTemporaryHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, ".config", "db-tui", configFileName)
}
