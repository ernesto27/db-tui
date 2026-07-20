// Package config loads db-tui configuration from the current user's home directory.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDirectoryMode = 0o700
	configFileMode      = 0o600
	configFileName      = "config.json"
)

// PostgreSQLConfig contains the PostgreSQL connection settings.
type PostgreSQLConfig struct {
	DSN string `json:"dsn"`
}

// Config contains db-tui connection settings.
type Config struct {
	PostgreSQL *PostgreSQLConfig `json:"postgresql"`
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
	if config.PostgreSQL == nil {
		return Config{}, errors.New(`config field "postgresql" is required`)
	}
	if strings.TrimSpace(config.PostgreSQL.DSN) == "" {
		return Config{}, errors.New(`config field "postgresql.dsn" is required`)
	}

	return config, nil
}

func createEmptyConfig(path string) ([]byte, error) {
	data, err := json.MarshalIndent(Config{
		PostgreSQL: &PostgreSQLConfig{},
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode empty config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, configFileMode); err != nil {
		return nil, fmt.Errorf("create empty config: %w", err)
	}
	return data, nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "db-tui", configFileName), nil
}
