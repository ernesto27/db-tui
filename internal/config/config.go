// Package config loads db-tui configuration from the current user's home directory.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ernestoponce27/db-tui/internal/db"
)

const (
	configDirectoryMode = 0o700
	configFileMode      = 0o600
	configFileName      = "config.json"
)

type Settings struct {
	Hostname string `json:"hostname"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	Port     string `json:"port"`
	DSN      string `json:"dsn"`
}

type Connection struct {
	Name     string   `json:"name"`
	Engine   string   `json:"engine"`
	Settings Settings `json:"settings"`
	Status   bool     `json:"status"`
}

// Config contains db-tui connection settings.
type Config struct {
	Connections []Connection `json:"connections,omitempty"`
	MaxPageSize int          `json:"maxPageSize"`
}

// PageSize returns the configured page size bounded by the application maximum.
func (config Config) PageSize() int {
	if config.MaxPageSize < 1 {
		return db.MaxPageSize
	}
	return min(config.MaxPageSize, db.MaxPageSize)
}

// Load reads the db-tui configuration from $HOME/.config/db-tui/config.json.
// It creates the configuration directory and an empty configuration file when needed.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	if err := os.MkdirAll(filepath.Dir(path), configDirectoryMode); err != nil {
		return Config{}, fmt.Errorf("create config directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data, err = createEmptyConfig(path)
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	config.MaxPageSize = config.PageSize()

	return config, nil
}

func (config *Config) saveConnection(conn Connection) error {
	config.Connections = append(config.Connections, conn)
	return config.Save()
}

// Save writes config to the db-tui configuration file.
func (config *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	config.MaxPageSize = config.PageSize()
	return writeConfig(path, *config)
}

func createEmptyConfig(path string) ([]byte, error) {
	data, err := encodeConfig(Config{
		MaxPageSize: db.MaxPageSize,
	})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, configFileMode); err != nil {
		return nil, fmt.Errorf("create empty config: %w", err)
	}
	return data, nil
}

func writeConfig(path string, config Config) error {
	data, err := encodeConfig(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, configFileMode); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func encodeConfig(config Config) ([]byte, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return append(data, '\n'), nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "db-tui", configFileName), nil
}
