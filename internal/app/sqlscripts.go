package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/utils"
)

const directorySqlScripts = "sql-scripts"

type SqlScript struct {
	name    string
	content string
}

type ListSqlScript struct {
	SqlScript []SqlScript
}

func (lq ListSqlScript) createByConnection(nameConn string, content string) error {
	dirSqlScripts, err := sqlScriptsDirectory(nameConn)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dirSqlScripts, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	randomValue, err := utils.RandomString(10)
	if err != nil {
		return err
	}

	path := filepath.Join(dirSqlScripts, randomValue+".txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return err
	}

	return nil
}

func (lq ListSqlScript) editByConnection(nameConn string, fileName string, content string) error {
	dirSqlScripts, err := sqlScriptsDirectory(nameConn)
	if err != nil {
		return err
	}
	path := filepath.Join(dirSqlScripts, fileName)

	_, err = os.Stat(path)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return err
	}

	return nil
}

func (lq ListSqlScript) getList(nameConn string) ([]SqlScript, error) {
	dirSqlScripts, err := sqlScriptsDirectory(nameConn)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dirSqlScripts)
	if err != nil {
		return nil, err
	}

	items := []SqlScript{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		written, err := os.ReadFile(filepath.Join(dirSqlScripts, entry.Name()))
		if err != nil {
			return nil, err
		}
		items = append(items, SqlScript{
			name:    entry.Name(),
			content: string(written),
		})
	}

	return items, nil
}

func sqlScriptsDirectory(nameConn string) (string, error) {
	if err := validateConnectionName(nameConn); err != nil {
		return "", err
	}

	configDir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, directorySqlScripts, nameConn), nil
}

func validateConnectionName(name string) error {
	if name == "" {
		return fmt.Errorf("connection name is required")
	}
	for _, character := range name {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') {
			return fmt.Errorf("connection name must contain only letters and numbers")
		}
	}
	return nil
}
