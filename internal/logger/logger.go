// Package logger records SQL statements and application messages.
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	configDirectoryMode = 0o700
	logFileMode         = 0o600
	logFileName         = "logs.txt"
)

// Logger writes timestamped SQL statements and messages to an output stream.
type Logger struct {
	mu     sync.Mutex
	logger *log.Logger
	closer io.Closer
}

// Open appends to the SQL log in the db-tui configuration directory.
func Open() (*Logger, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	logDirectory := filepath.Join(configDirectory, "db-tui")
	if err := os.MkdirAll(logDirectory, configDirectoryMode); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filepath.Join(logDirectory, logFileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, logFileMode)
	if err != nil {
		return nil, err
	}

	logger := New(file)
	logger.closer = file
	return logger, nil
}

// New creates a logger that writes to writer.
func New(writer io.Writer) *Logger {
	return &Logger{logger: log.New(writer, "", log.LstdFlags)}
}

// Log records query as a single line of SQL.
func (l *Logger) Log(query string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("SQL: %s", strings.Join(strings.Fields(query), " "))
}

// LogMessage records message as a single line.
func (l *Logger) LogMessage(message string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("MESSAGE: %s", strings.Join(strings.Fields(message), " "))
}

// Close closes the underlying log file when the logger owns one.
func (l *Logger) Close() error {
	if l == nil || l.closer == nil {
		return nil
	}
	return l.closer.Close()
}
