// Package config loads db-tui configuration from the current user's home directory.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

type ConnectionEnvironment string

const (
	ConnectionEnvironmentTesting    ConnectionEnvironment = "testing"
	ConnectionEnvironmentProduction ConnectionEnvironment = "production"
)

type Connection struct {
	Name        string                `json:"name"`
	Engine      string                `json:"engine"`
	Settings    Settings              `json:"settings"`
	Environment ConnectionEnvironment `json:"environment,omitempty"`
	Status      bool                  `json:"status"`
}

// Config contains db-tui connection settings.
type Config struct {
	Connections []Connection `json:"connections,omitempty"`
	MaxPageSize int          `json:"maxPageSize"`
}

// PageSize returns the configured page size or the default when it is invalid.
func (config Config) PageSize() int {
	if config.MaxPageSize < 1 {
		return db.MaxPageSize
	}
	return config.MaxPageSize
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
	return decodeConfig(data)
}

// LoadExisting reads the current user's config without creating it when absent.
func LoadExisting() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	return decodeConfig(data)
}

func decodeConfig(data []byte) (Config, error) {
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

// FindConnection returns the first saved connection with the exact name.
func (config Config) FindConnection(name string) (Connection, error) {
	for _, connection := range config.Connections {
		if connection.Name == name {
			return connection, nil
		}
	}
	return Connection{}, fmt.Errorf("saved connection %q not found", name)
}

// Target returns the adapter engine and DSN for a saved connection.
func (connection Connection) Target() (string, string, error) {
	engine := strings.TrimSpace(connection.Engine)
	switch engine {
	case db.EnginePostgreSQL, db.EngineMySQL, db.EngineOracle, db.EngineSQLite, db.EngineSQLServer:
	default:
		return "", "", fmt.Errorf("unsupported database engine %q", engine)
	}

	settings := connection.Settings
	if dsn := strings.TrimSpace(settings.DSN); dsn != "" {
		return engine, dsn, nil
	}
	if engine == db.EngineSQLite {
		return "", "", errors.New("SQLite database file is required")
	}

	host := strings.TrimSpace(settings.Hostname)
	databaseName := strings.TrimSpace(settings.Database)
	username := strings.TrimSpace(settings.Username)
	if host == "" {
		return "", "", errors.New("host is required")
	}
	if databaseName == "" {
		return "", "", errors.New("database name is required")
	}
	if username == "" {
		return "", "", errors.New("username is required")
	}
	port, err := strconv.Atoi(strings.TrimSpace(settings.Port))
	if err != nil || port < 1 || port > 65535 {
		return "", "", errors.New("port must be between 1 and 65535")
	}

	user := url.User(username)
	if settings.Password != "" {
		user = url.UserPassword(username, settings.Password)
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	if engine == db.EngineSQLServer {
		// SQL Server interprets a URL path as an instance name.
		return engine, (&url.URL{
			Scheme:   "sqlserver",
			User:     user,
			Host:     address,
			RawQuery: url.Values{"database": {databaseName}}.Encode(),
		}).String(), nil
	}

	scheme := "postgres"
	if engine == db.EngineMySQL {
		scheme = "mysql"
	}
	if engine == db.EngineOracle {
		scheme = "oracle"
	}
	return engine, (&url.URL{
		Scheme: scheme,
		User:   user,
		Host:   address,
		Path:   "/" + databaseName,
	}).String(), nil
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

// ConfigDir returns the db-tui configuration directory.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "db-tui"), nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "db-tui", configFileName), nil
}
