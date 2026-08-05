package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestConnectionSettingsConnectionDSN(t *testing.T) {
	tests := []struct {
		name     string
		settings ConnectionSettings
		want     string
		wantErr  string
	}{
		{
			name:     "explicit DSN takes precedence",
			settings: ConnectionSettings{Engine: db.EnginePostgreSQL, DSN: " postgres://saved "},
			want:     "postgres://saved",
		},
		{
			name: "MySQL fields",
			settings: ConnectionSettings{
				Engine:       "mysql",
				Host:         "127.0.0.1",
				Port:         3306,
				DatabaseName: "chinook",
				Username:     "db_tui",
				Password:     "secret",
			},
			want: "mysql://db_tui:secret@127.0.0.1:3306/chinook",
		},
		{
			name: "valid fields",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Host:         "127.0.0.1",
				Port:         5433,
				DatabaseName: "chinook",
				Username:     "db_tui",
				Password:     "secret",
			},
			want: "postgres://db_tui:secret@127.0.0.1:5433/chinook",
		},
		{
			name: "missing host",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Port:         5432,
				DatabaseName: "chinook",
				Username:     "db_tui",
			},
			wantErr: "host is required",
		},
		{
			name: "missing database",
			settings: ConnectionSettings{
				Engine:   db.EnginePostgreSQL,
				Host:     "127.0.0.1",
				Port:     5432,
				Username: "db_tui",
			},
			wantErr: "database name is required",
		},
		{
			name: "missing username",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Host:         "127.0.0.1",
				Port:         5432,
				DatabaseName: "chinook",
			},
			wantErr: "username is required",
		},
		{
			name: "zero port",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Host:         "127.0.0.1",
				DatabaseName: "chinook",
				Username:     "db_tui",
			},
			wantErr: "port must be between 1 and 65535",
		},
		{
			name: "negative port",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Host:         "127.0.0.1",
				Port:         -1,
				DatabaseName: "chinook",
				Username:     "db_tui",
			},
			wantErr: "port must be between 1 and 65535",
		},
		{
			name: "port above maximum",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Host:         "127.0.0.1",
				Port:         65536,
				DatabaseName: "chinook",
				Username:     "db_tui",
			},
			wantErr: "port must be between 1 and 65535",
		},
		{
			name: "credentials and database are escaped",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Host:         "db.example.com",
				Port:         5432,
				DatabaseName: "sales data",
				Username:     "db user",
				Password:     "p@ss",
			},
			want: "postgres://db%20user:p%40ss@db.example.com:5432/sales%20data",
		},
		{
			name: "IPv6 host",
			settings: ConnectionSettings{
				Engine:       db.EnginePostgreSQL,
				Host:         "2001:db8::1",
				Port:         5432,
				DatabaseName: "chinook",
				Username:     "db_tui",
			},
			want: "postgres://db_tui@[2001:db8::1]:5432/chinook",
		},
		{
			name: "Oracle service name",
			settings: ConnectionSettings{
				Engine:       db.EngineOracle,
				Host:         "127.0.0.1",
				Port:         1522,
				DatabaseName: "FREEPDB1",
				Username:     "db_tui",
				Password:     "p@ss",
			},
			want: "oracle://db_tui:p%40ss@127.0.0.1:1522/FREEPDB1",
		},
		{
			name:     "SQLite database file",
			settings: ConnectionSettings{Engine: "sqlite", DSN: " ./database.db "},
			want:     "./database.db",
		},
		{
			name:     "SQLite requires database file",
			settings: ConnectionSettings{Engine: "sqlite"},
			wantErr:  "SQLite database file is required",
		},
		{
			name:     "blank engine",
			settings: ConnectionSettings{DSN: "postgres://database"},
			wantErr:  `unsupported database engine ""`,
		},
		{
			name:     "PostgreSQL alias is rejected",
			settings: ConnectionSettings{Engine: "postgresql", DSN: "postgres://database"},
			wantErr:  `unsupported database engine "postgresql"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dsn, err := test.settings.connectionDSN()
			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, dsn)
		})
	}
}

func TestConnectConnection(t *testing.T) {
	connectorErr := errors.New("connect failed")
	database := &fakeDatabase{name: "chinook"}
	successSettings := ConnectionSettings{
		Engine:       db.EnginePostgreSQL,
		Host:         "127.0.0.1",
		Port:         5433,
		DatabaseName: "chinook",
		Username:     "db_tui",
	}

	tests := []struct {
		name         string
		settings     ConnectionSettings
		attempt      uint64
		database     db.Database
		connectorErr error
		wantCalls    int
		wantEngine   string
		wantDSN      string
		wantErr      error
		wantErrText  string
	}{
		{
			name:        "validation failure does not call connector",
			settings:    ConnectionSettings{Engine: db.EnginePostgreSQL},
			attempt:     3,
			wantErrText: "host is required",
		},
		{
			name:         "connector error",
			settings:     ConnectionSettings{Engine: db.EnginePostgreSQL, DSN: "postgres://database"},
			attempt:      4,
			connectorErr: connectorErr,
			wantCalls:    1,
			wantEngine:   db.EnginePostgreSQL,
			wantDSN:      "postgres://database",
			wantErr:      connectorErr,
		},
		{
			name:       "success",
			settings:   successSettings,
			attempt:    5,
			database:   database,
			wantCalls:  1,
			wantEngine: db.EnginePostgreSQL,
			wantDSN:    "postgres://db_tui@127.0.0.1:5433/chinook",
		},
		{
			name:       "SQLite passes database file to connector",
			settings:   ConnectionSettings{Engine: "sqlite", DSN: "docker/sqlite/employee.db"},
			attempt:    6,
			database:   database,
			wantCalls:  1,
			wantEngine: "sqlite",
			wantDSN:    "docker/sqlite/employee.db",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var gotDSN string
			var gotEngine string
			var hadDeadline bool
			connect := func(ctx context.Context, engine, dsn string) (db.Database, error) {
				calls++
				gotEngine = engine
				gotDSN = dsn
				_, hadDeadline = ctx.Deadline()
				return test.database, test.connectorErr
			}

			message, ok := connectConnection(connect, test.settings, test.attempt)().(connectionFinishedMsg)
			require.True(t, ok)

			assert.Equal(t, test.wantCalls, calls)
			assert.Equal(t, test.attempt, message.attempt)
			if test.wantCalls == 0 {
				assert.Empty(t, gotDSN)
				assert.False(t, hadDeadline)
			} else {
				assert.Equal(t, test.wantEngine, gotEngine)
				assert.Equal(t, test.wantDSN, gotDSN)
				assert.True(t, hadDeadline)
			}

			if test.wantErrText != "" {
				assert.EqualError(t, message.err, test.wantErrText)
				assert.Nil(t, message.database)
				return
			}
			if test.wantErr != nil {
				assert.ErrorIs(t, message.err, test.wantErr)
				assert.Nil(t, message.database)
				return
			}

			assert.NoError(t, message.err)
			assert.Same(t, test.database, message.database)
			wantSettings := test.settings
			wantSettings.Engine = test.wantEngine
			assert.Equal(t, wantSettings, message.settings)
		})
	}
}
