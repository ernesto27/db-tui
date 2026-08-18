package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/config"
)

func TestListSqlScriptCreateByConnection(t *testing.T) {
	const (
		connectionName = "testconnection"
		content        = "SELECT id, name, email FROM users WHERE active = TRUE ORDER BY name;"
	)

	lsql := ListSqlScript{}
	err := lsql.createByConnection(connectionName, content)
	require.NoError(t, err)

	configDir, err := config.ConfigDir()
	require.NoError(t, err)
	directory := filepath.Join(configDir, directorySqlScripts, connectionName)
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.False(t, entries[0].IsDir())
	assert.Equal(t, ".txt", filepath.Ext(entries[0].Name()))
	assert.Len(t, entries[0].Name(), 14)

	written, err := os.ReadFile(filepath.Join(directory, entries[0].Name()))
	require.NoError(t, err)
	assert.Equal(t, content, string(written))
}

func TestListSqlScriptEditByConnection(t *testing.T) {
	const (
		connectionName = "edittestconnection"
		oldContent     = "SELECT id, name FROM users;"
		content        = "SELECT id, name, email FROM users WHERE active = TRUE ORDER BY name;"
	)

	lsql := ListSqlScript{}
	lsql.createByConnection(connectionName, oldContent)

	configDir, err := config.ConfigDir()
	require.NoError(t, err)
	directory := filepath.Join(configDir, directorySqlScripts, connectionName)
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	err = lsql.editByConnection(connectionName, entries[0].Name(), content)
	require.NoError(t, err)

	written, err := os.ReadFile(filepath.Join(directory, entries[0].Name()))
	require.NoError(t, err)
	assert.Equal(t, content, string(written))
}

func TestListSqlScriptGetList(t *testing.T) {
	const connectionName = "listtestconnection"

	list := ListSqlScript{}
	require.NoError(t, list.createByConnection(connectionName, "SELECT id FROM users;"))
	require.NoError(t, list.createByConnection(connectionName, "SELECT id FROM roles;"))

	scripts, err := list.getList(connectionName)
	require.NoError(t, err)
	require.Len(t, scripts, 2)

	contents := make([]string, 0, len(scripts))
	for _, script := range scripts {
		assert.NotEmpty(t, script.name)
		contents = append(contents, script.content)
	}
	assert.ElementsMatch(t, []string{
		"SELECT id FROM users;",
		"SELECT id FROM roles;",
	}, contents)
}

func TestSQLScriptsDirectoryValidatesConnectionName(t *testing.T) {
	tests := []struct {
		name           string
		connectionName string
		errorMessage   string
	}{
		{name: "random string characters", connectionName: "aB7z"},
		{name: "empty", errorMessage: "connection name is required"},
		{name: "current directory", connectionName: "./scripts", errorMessage: "connection name must contain only letters and numbers"},
		{name: "parent directory", connectionName: "../scripts", errorMessage: "connection name must contain only letters and numbers"},
		{name: "path separator", connectionName: "scripts/other", errorMessage: "connection name must contain only letters and numbers"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := sqlScriptsDirectory(test.connectionName)
			if test.errorMessage == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, test.errorMessage)
		})
	}
}
