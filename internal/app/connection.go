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

// ConnectFunc opens a database session for dsn.
type ConnectFunc func(context.Context, string) (db.Database, error)

// ConnectionSettings identifies a PostgreSQL connection entered in the TUI.
type ConnectionSettings struct {
	DSN          string
	Host         string
	Port         int
	DatabaseName string
	Username     string
	Password     string
}

// SaveConnectionFunc persists a successfully verified database connection.
type SaveConnectionFunc func(ConnectionSettings) error

type connectionFinishedMsg struct {
	database db.Database
	settings ConnectionSettings
	attempt  uint64
	err      error
}

func (s ConnectionSettings) connectionDSN() (string, error) {
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
	return (&url.URL{
		Scheme: "postgres",
		User:   user,
		Host:   net.JoinHostPort(host, strconv.Itoa(s.Port)),
		Path:   "/" + databaseName,
	}).String(), nil
}

func connectAndSave(connect ConnectFunc, saveConnection SaveConnectionFunc, settings ConnectionSettings, attempt uint64) tea.Cmd {
	return func() tea.Msg {
		dsn, err := settings.connectionDSN()
		if err != nil {
			return connectionFinishedMsg{attempt: attempt, err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
		defer cancel()

		database, err := connect(ctx, dsn)
		if err != nil {
			return connectionFinishedMsg{attempt: attempt, err: err}
		}
		if err := saveConnection(settings); err != nil {
			database.Close()
			return connectionFinishedMsg{
				attempt: attempt,
				err:     fmt.Errorf("save connection: %w", err),
			}
		}

		return connectionFinishedMsg{database: database, settings: settings, attempt: attempt}
	}
}
