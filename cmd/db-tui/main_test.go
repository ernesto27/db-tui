package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommandRoutesWorkflows(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		queryResult     string
		wantInteractive bool
		wantQuery       bool
		wantFormat      string
		wantOutput      string
	}{
		{
			name:            "starts interactive client without arguments",
			wantInteractive: true,
		},
		{
			name:        "runs query subcommand with short flags",
			args:        []string{"query", "-q", "select 1", "-c", "postgres://localhost/test"},
			queryResult: `[{"value":1}]`,
			wantQuery:   true,
			wantFormat:  db.ExportTypeJSON,
			wantOutput:  "[{\"value\":1}]\n",
		},
		{
			name:        "runs query subcommand with long flags",
			args:        []string{"query", "--query", "select 1", "--dsn", "postgres://localhost/test", "--format", "csv"},
			queryResult: "value\n1\n",
			wantQuery:   true,
			wantFormat:  db.ExportTypeCSV,
			wantOutput:  "value\n1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			interactiveStarted := false
			queryExecuted := false
			var receivedEngine string
			var receivedFormat string
			cmd := newRootCmd(cliDependencies{
				startInteractive: func() error {
					interactiveStarted = true
					return nil
				},
				executeQuery: func(_ context.Context, engine, _, _, format string) (string, error) {
					queryExecuted = true
					receivedEngine = engine
					receivedFormat = format
					return tt.queryResult, nil
				},
			})
			cmd.SetArgs(tt.args)
			cmd.SetOut(&output)
			cmd.SetErr(io.Discard)

			err := cmd.ExecuteContext(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tt.wantInteractive, interactiveStarted)
			assert.Equal(t, tt.wantQuery, queryExecuted)
			assert.Equal(t, tt.wantOutput, output.String())
			if tt.wantQuery {
				assert.Equal(t, db.EnginePostgreSQL, receivedEngine)
				assert.Equal(t, tt.wantFormat, receivedFormat)
			}
		})
	}
}

func TestListCommand(t *testing.T) {
	tests := []struct {
		name         string
		connections  []config.Connection
		wantContains []string
		wantOutput   string
		wantAbsent   []string
	}{
		{
			name: "shows saved connections",
			connections: []config.Connection{
				{
					Name:        "reporting",
					Engine:      db.EnginePostgreSQL,
					Environment: config.ConnectionEnvironmentProduction,
					Settings: config.Settings{
						Password: "test-secret",
						DSN:      "postgres://reader:test-secret@localhost/app",
					},
				},
			},
			wantContains: []string{"NAME", "ENGINE", "ENVIRONMENT", "reporting", db.EnginePostgreSQL, string(config.ConnectionEnvironmentProduction)},
			wantAbsent:   []string{"test-secret", "postgres://reader:"},
		},
		{
			name:       "shows message for empty config",
			wantOutput: "No saved connections.\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			appConfig, err := config.Load()
			require.NoError(t, err)
			appConfig.Connections = test.connections
			require.NoError(t, appConfig.Save())

			var output bytes.Buffer
			cmd := newRootCmd(cliDependencies{})
			cmd.SetArgs([]string{"list"})
			cmd.SetOut(&output)
			cmd.SetErr(io.Discard)

			require.NoError(t, cmd.ExecuteContext(context.Background()))
			if test.wantOutput != "" {
				assert.Equal(t, test.wantOutput, output.String())
			}
			for _, value := range test.wantContains {
				assert.Contains(t, output.String(), value)
			}
			for _, value := range test.wantAbsent {
				assert.NotContains(t, output.String(), value)
			}
		})
	}
}

func TestQueryCommandReadsQueryFromFile(t *testing.T) {
	const query = "SELECT 1;"
	queryFile := filepath.Join(t.TempDir(), "query.sql")
	require.NoError(t, os.WriteFile(queryFile, []byte(query), 0o600))

	var receivedQuery string
	cmd := newQueryCmd(func(_ context.Context, _, _, gotQuery, _ string) (string, error) {
		receivedQuery = gotQuery
		return "[]", nil
	})
	cmd.SetArgs([]string{"--dsn", "postgres://localhost/test", "--fileQuery", queryFile})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.ExecuteContext(context.Background())

	require.NoError(t, err)
	assert.Equal(t, query, receivedQuery)
}

func TestQueryCommandRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "query without DSN", args: []string{"query", "-q", "select 1"}},
		{name: "DSN without query", args: []string{"query", "--dsn", "postgres://localhost/test"}},
		{name: "query and file query", args: []string{"query", "--query", "select 1", "--fileQuery", "query.sql", "--dsn", "postgres://localhost/test"}},
		{name: "format without query and DSN", args: []string{"query", "--format", "csv"}},
		{name: "unsupported format", args: []string{"query", "-q", "select 1", "-c", "postgres://localhost/test", "-t", "xml"}},
		{name: "unsupported DSN", args: []string{"query", "-q", "select 1", "-c", "redis://localhost"}},
		{name: "positional argument", args: []string{"query", "extra"}},
		{name: "unknown flag", args: []string{"query", "--unknown"}},
		{name: "root query flag", args: []string{"--query", "select 1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interactiveStarted := false
			queryExecuted := false
			cmd := newRootCmd(cliDependencies{
				startInteractive: func() error {
					interactiveStarted = true
					return nil
				},
				executeQuery: func(context.Context, string, string, string, string) (string, error) {
					queryExecuted = true
					return "", nil
				},
			})
			cmd.SetArgs(tt.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.ExecuteContext(context.Background())

			require.Error(t, err)
			assert.Equal(t, exitCodeUsage, commandExitCode(err))
			assert.False(t, interactiveStarted)
			assert.False(t, queryExecuted)
		})
	}
}

func TestDumpCommandUsesDSNFlag(t *testing.T) {
	const dsn = "postgres://localhost/test"
	var receivedDSN string
	cmd := newDumpCmd(func(_ context.Context, _, received string) error {
		receivedDSN = received
		return nil
	})
	cmd.SetArgs([]string{"--dsn", dsn})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.ExecuteContext(context.Background())

	require.NoError(t, err)
	assert.Equal(t, dsn, receivedDSN)
}

func TestSavedConnectionCommands(t *testing.T) {
	const (
		formDSN   = "postgres://reader@localhost:5432/app"
		nativeDSN = "reader@tcp(localhost:3306)/app"
		directDSN = "postgres://localhost/direct"
	)
	tests := []struct {
		name         string
		args         []string
		noConfig     bool
		wantEngine   string
		wantDSN      string
		wantError    string
		wantExitCode int
	}{
		{
			name:       "query uses saved form fields",
			args:       []string{"query", "--connection", "form", "-q", "SELECT 1"},
			wantEngine: db.EnginePostgreSQL,
			wantDSN:    formDSN,
		},
		{
			name:       "dump uses saved explicit DSN and engine",
			args:       []string{"dump", "--connection", "native-mysql"},
			wantEngine: db.EngineMySQL,
			wantDSN:    nativeDSN,
		},
		{
			name:       "duplicate name uses first entry",
			args:       []string{"query", "--connection", "duplicate", "-q", "SELECT 1"},
			wantEngine: db.EnginePostgreSQL,
			wantDSN:    formDSN,
		},
		{
			name:       "nonempty DSN takes precedence without reading config",
			args:       []string{"query", "--connection", "missing", "--dsn", directDSN, "-q", "SELECT 1"},
			noConfig:   true,
			wantEngine: db.EnginePostgreSQL,
			wantDSN:    directDSN,
		},
		{
			name:       "empty DSN falls back to saved name",
			args:       []string{"query", "--connection", "form", "--dsn", "", "-q", "SELECT 1"},
			wantEngine: db.EnginePostgreSQL,
			wantDSN:    formDSN,
		},
		{
			name:         "name is case sensitive",
			args:         []string{"query", "--connection", "Form", "-q", "SELECT 1"},
			wantError:    `saved connection "Form" not found`,
			wantExitCode: exitCodeUsage,
		},
		{
			name:         "missing config does not create a file",
			args:         []string{"dump", "--connection", "form"},
			noConfig:     true,
			wantError:    "load saved connections",
			wantExitCode: exitCodeRuntime,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			configDir, err := config.ConfigDir()
			require.NoError(t, err)
			if !test.noConfig {
				require.NoError(t, os.MkdirAll(configDir, 0o700))
				appConfig := config.Config{Connections: []config.Connection{
					{Name: "form", Engine: db.EnginePostgreSQL, Settings: config.Settings{
						Hostname: "localhost", Port: "5432", Database: "app", Username: "reader",
					}},
					{Name: "native-mysql", Engine: db.EngineMySQL, Settings: config.Settings{DSN: nativeDSN}},
					{Name: "duplicate", Engine: db.EnginePostgreSQL, Settings: config.Settings{DSN: formDSN}},
					{Name: "duplicate", Engine: db.EngineMySQL, Settings: config.Settings{DSN: nativeDSN}},
				}}
				require.NoError(t, appConfig.Save())
			}

			called := false
			var gotEngine, gotDSN string
			capture := func(engine, dsn string) {
				called = true
				gotEngine, gotDSN = engine, dsn
			}
			cmd := newRootCmd(cliDependencies{
				startInteractive: func() error { return nil },
				executeQuery: func(_ context.Context, engine, dsn, _, _ string) (string, error) {
					capture(engine, dsn)
					return "[]", nil
				},
				dumpDatabase: func(_ context.Context, engine, dsn string) error {
					capture(engine, dsn)
					return nil
				},
			})
			cmd.SetArgs(test.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err = cmd.ExecuteContext(context.Background())
			if test.wantError != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, test.wantError)
				assert.Equal(t, test.wantExitCode, commandExitCode(err))
				assert.False(t, called)
				if test.noConfig {
					_, statErr := os.Stat(filepath.Join(configDir, "config.json"))
					assert.ErrorIs(t, statErr, os.ErrNotExist)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, called)
			assert.Equal(t, test.wantEngine, gotEngine)
			assert.Equal(t, test.wantDSN, gotDSN)
		})
	}
}

func TestSavedConnectionAdapterErrorsHideCredentials(t *testing.T) {
	const (
		password = "FAKE_PASSWORD"
		dsn      = "mysql://review:" + password + "@localhost:bad/db"
	)
	t.Setenv("HOME", t.TempDir())
	configDir, err := config.ConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	appConfig := config.Config{Connections: []config.Connection{{
		Name:     "reporting",
		Engine:   db.EngineMySQL,
		Settings: config.Settings{DSN: dsn},
	}}}
	require.NoError(t, appConfig.Save())

	tests := []struct {
		name string
		args []string
	}{
		{name: "query", args: []string{"query", "--connection", "reporting", "-q", "SELECT 1"}},
		{name: "dump", args: []string{"dump", "--connection", "reporting"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := newRootCmd(cliDependencies{
				executeQuery: executeCLI,
				dumpDatabase: runDump,
			})
			cmd.SetArgs(test.args)
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.ExecuteContext(context.Background())

			require.Error(t, err)
			assert.Equal(t, exitCodeRuntime, commandExitCode(err))
			assert.ErrorContains(t, err, test.name+" using saved connection failed")
			assert.Contains(t, stderr.String(), test.name+" using saved connection failed")
			assert.NotContains(t, err.Error(), password)
			assert.NotContains(t, stderr.String(), password)
			assert.NotContains(t, stderr.String(), dsn)
			assert.Empty(t, stdout.String())
		})
	}
}

func TestQueryCommandHelpDoesNotStartWorkflow(t *testing.T) {
	var output bytes.Buffer
	interactiveStarted := false
	queryExecuted := false
	cmd := newRootCmd(cliDependencies{
		startInteractive: func() error {
			interactiveStarted = true
			return nil
		},
		executeQuery: func(context.Context, string, string, string, string) (string, error) {
			queryExecuted = true
			return "", nil
		},
	})
	cmd.SetArgs([]string{"query", "--help"})
	cmd.SetOut(&output)
	cmd.SetErr(io.Discard)

	err := cmd.ExecuteContext(context.Background())

	require.NoError(t, err)
	assert.Contains(t, output.String(), "--query")
	assert.Contains(t, output.String(), "--dsn")
	assert.Contains(t, output.String(), "--format")
	assert.False(t, interactiveStarted)
	assert.False(t, queryExecuted)
}

func TestQueryCommandClassifiesQueryFailures(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newRootCmd(cliDependencies{
		startInteractive: func() error {
			return nil
		},
		executeQuery: func(context.Context, string, string, string, string) (string, error) {
			return "", errors.New("connection lost")
		},
	})
	cmd.SetArgs([]string{"query", "--query", "select 1", "--dsn", "postgres://localhost/test"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.ExecuteContext(context.Background())

	require.Error(t, err)
	assert.Equal(t, exitCodeRuntime, commandExitCode(err))
	assert.ErrorContains(t, err, "query: connection lost")
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "Error: query: connection lost")
	assert.NotContains(t, stderr.String(), "Usage:")
}
