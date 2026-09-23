package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
)

// ConnectFunc opens a database session for an engine and dsn.
type ConnectFunc func(context.Context, string, string) (db.Database, error)

// ConnectionSettings identifies a database connection entered in the TUI.
type ConnectionSettings struct {
	Engine       string
	DSN          string
	Host         string
	Port         int
	DatabaseName string
	Username     string
	Password     string
}

type connectionFinishedMsg struct {
	database db.Database
	settings ConnectionSettings
	attempt  uint64
	err      error
}

type lastConnectionSavedMsg struct {
	name    string
	session uint64
	err     error
}

const lastConnectionOpenErrorText = "Unable to open last used connection"

func saveLastConnection(cfg config.Config, name string, session uint64) tea.Cmd {
	return func() tea.Msg {
		cfg.LastConnectionName = name
		err := cfg.Save()
		return lastConnectionSavedMsg{name: name, session: session, err: err}
	}
}

func (s ConnectionSettings) connectionDSN() (string, error) {
	engine, err := s.normalizedEngine()
	if err != nil {
		return "", err
	}
	_, dsn, err := (config.Connection{
		Engine:   engine,
		Settings: configSettingsFromConnectionSettings(s),
	}).Target()
	return dsn, err
}

func (s ConnectionSettings) normalizedEngine() (string, error) {
	switch s.Engine {
	case db.EnginePostgreSQL:
		return db.EnginePostgreSQL, nil
	case db.EngineMySQL:
		return db.EngineMySQL, nil
	case db.EngineOracle:
		return db.EngineOracle, nil
	case db.EngineSQLite:
		return db.EngineSQLite, nil
	case db.EngineSQLServer:
		return db.EngineSQLServer, nil
	default:
		return "", fmt.Errorf("unsupported database engine %q", s.Engine)
	}
}

func connectConnection(connect ConnectFunc, settings ConnectionSettings, attempt uint64) tea.Cmd {
	return func() tea.Msg {
		engine, err := settings.normalizedEngine()
		if err != nil {
			return connectionFinishedMsg{attempt: attempt, err: err}
		}
		settings.Engine = engine
		dsn, err := settings.connectionDSN()
		if err != nil {
			return connectionFinishedMsg{attempt: attempt, err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		database, err := connect(ctx, engine, dsn)
		if err != nil {
			return connectionFinishedMsg{attempt: attempt, err: err}
		}
		return connectionFinishedMsg{database: database, settings: settings, attempt: attempt}
	}
}

// renameRequestMsg carries a rename completion.
type renameRequestMsg struct {
	request uint64
	index   int
	config  config.Config
	err     error
}

type environmentSaveMsg struct {
	request uint64
	index   int
	config  config.Config
	err     error
}

// saveConnectionName persists the renamed connection via Config.Save.
func saveConnectionName(cfg config.Config, renameRequest uint64, index int) tea.Cmd {
	return func() tea.Msg {
		err := cfg.Save()
		return renameRequestMsg{
			request: renameRequest,
			index:   index,
			config:  cfg,
			err:     err,
		}
	}
}

func saveConnectionEnvironment(cfg config.Config, environmentRequest uint64, index int) tea.Cmd {
	return func() tea.Msg {
		err := cfg.Save()
		return environmentSaveMsg{
			request: environmentRequest,
			index:   index,
			config:  cfg,
			err:     err,
		}
	}
}

func renameConnectionWithSQLScripts(cfg config.Config, oldName, newName string, request uint64, index int) tea.Cmd {
	return func() tea.Msg {
		source, err := sqlScriptsDirectory(oldName)
		if err != nil {
			return renameRequestMsg{request: request, index: index, config: cfg, err: err}
		}
		destination, err := sqlScriptsDirectory(newName)
		if err != nil {
			return renameRequestMsg{request: request, index: index, config: cfg, err: err}
		}
		moved := false
		if source != destination {
			if _, err := os.Stat(destination); err == nil {
				return renameRequestMsg{request: request, index: index, config: cfg, err: fmt.Errorf("saved SQL script library already exists for %q", newName)}
			} else if !errors.Is(err, os.ErrNotExist) {
				return renameRequestMsg{request: request, index: index, config: cfg, err: fmt.Errorf("inspect destination SQL scripts directory: %w", err)}
			}
			if info, err := os.Stat(source); err == nil {
				if !info.IsDir() {
					return renameRequestMsg{request: request, index: index, config: cfg, err: errors.New("source SQL scripts path is not a directory")}
				}
				if err := os.Rename(source, destination); err != nil {
					return renameRequestMsg{request: request, index: index, config: cfg, err: fmt.Errorf("move SQL scripts directory: %w", err)}
				}
				moved = true
			} else if !errors.Is(err, os.ErrNotExist) {
				return renameRequestMsg{request: request, index: index, config: cfg, err: fmt.Errorf("inspect source SQL scripts directory: %w", err)}
			}
		}
		if err := cfg.Save(); err != nil {
			if moved {
				if rollbackErr := os.Rename(destination, source); rollbackErr != nil {
					err = fmt.Errorf("save connection name: %w; restore SQL scripts directory: %v", err, rollbackErr)
				}
			}
			return renameRequestMsg{request: request, index: index, config: cfg, err: err}
		}
		return renameRequestMsg{request: request, index: index, config: cfg}
	}
}
