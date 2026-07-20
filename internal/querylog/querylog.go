// Package querylog records executed SQL statements.
package querylog

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Logger writes timestamped SQL statements to an output stream.
type Logger struct {
	mu     sync.Mutex
	logger *log.Logger
	closer io.Closer
}

// Open creates or truncates the query log at path.
func Open(path string) (*Logger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}

	logger := New(file)
	logger.closer = file
	return logger, nil
}

// New creates a query logger that writes to writer.
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

// Close closes the underlying log file when the logger owns one.
func (l *Logger) Close() error {
	if l == nil || l.closer == nil {
		return nil
	}
	return l.closer.Close()
}
