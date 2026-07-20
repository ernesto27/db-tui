// Package config loads db-tui configuration from the current user's home directory.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ernestoponce27/db-tui/internal/db/postgres"
)

const (
	configDirectoryMode = 0o700
	configFileMode      = 0o600
	configFileName      = "config.json"
)

// Config contains db-tui connection settings.
type Config struct {
	PostgreSQL *postgres.PostgreSQLConfig `json:"postgresql"`
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
	if _, err := config.PostgreSQL.ConnectionDSN(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// SavePostgreSQLDSN writes dsn as the saved PostgreSQL connection.
func SavePostgreSQLDSN(dsn string) error {
	return SavePostgreSQLConfig(postgres.PostgreSQLConfig{DSN: dsn})
}

// SavePostgreSQLConfig writes PostgreSQL connection settings.
func SavePostgreSQLConfig(postgreSQLConfig postgres.PostgreSQLConfig) error {
	if _, err := postgreSQLConfig.ConnectionDSN(); err != nil {
		return err
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	return writeConfig(path, Config{PostgreSQL: &postgreSQLConfig})
}

func createEmptyConfig(path string) ([]byte, error) {
	data, err := encodeConfig(Config{PostgreSQL: &postgres.PostgreSQLConfig{}})
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
