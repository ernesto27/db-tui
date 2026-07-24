package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

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

func (s ConnectionSettings) connectionDSN() (string, error) {
	engine, err := s.normalizedEngine()
	if err != nil {
		return "", err
	}
	if dsn := strings.TrimSpace(s.DSN); dsn != "" {
		return dsn, nil
	}

	host := strings.TrimSpace(s.Host)
	databaseName := strings.TrimSpace(s.DatabaseName)
	username := strings.TrimSpace(s.Username)
	if host == "" {
		return "", errors.New("host is required")
	}
	if databaseName == "" {
		return "", errors.New("database name is required")
	}
	if username == "" {
		return "", errors.New("username is required")
	}
	if s.Port < 1 || s.Port > 65535 {
		return "", errors.New("port must be between 1 and 65535")
	}

	user := url.User(username)
	if s.Password != "" {
		user = url.UserPassword(username, s.Password)
	}
	scheme := "postgres"
	if engine == db.EngineMySQL {
		scheme = "mysql"
	}
	return (&url.URL{
		Scheme: scheme,
		User:   user,
		Host:   net.JoinHostPort(host, strconv.Itoa(s.Port)),
		Path:   "/" + databaseName,
	}).String(), nil
}

func (s ConnectionSettings) normalizedEngine() (string, error) {
	switch s.Engine {
	case db.EnginePostgreSQL:
		return db.EnginePostgreSQL, nil
	case db.EngineMySQL:
		return db.EngineMySQL, nil
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
