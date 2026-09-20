package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

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

func TestQueryCommandRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "query without DSN", args: []string{"query", "-q", "select 1"}},
		{name: "DSN without query", args: []string{"query", "--dsn", "postgres://localhost/test"}},
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
