package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	type scriptFile struct {
		script     SqlScript
		modifiedAt int64
	}
	files := make([]scriptFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		written, err := os.ReadFile(filepath.Join(dirSqlScripts, entry.Name()))
		if err != nil {
			return nil, err
		}
		files = append(files, scriptFile{
			script: SqlScript{
				name:    entry.Name(),
				content: string(written),
			},
			modifiedAt: info.ModTime().UnixNano(),
		})
	}

	sort.SliceStable(files, func(left, right int) bool {
		return files[left].modifiedAt > files[right].modifiedAt
	})
	items := make([]SqlScript, len(files))
	for index, file := range files {
		items[index] = file.script
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
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("connection name is required")
	}
	if name == "." || name == ".." ||
		filepath.IsAbs(name) ||
		filepath.VolumeName(name) != "" ||
		strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("connection name must be a single directory name")
	}
	if strings.ContainsRune(name, '\x00') {
		return fmt.Errorf("connection name must not contain a null character")
	}
	return nil
}
