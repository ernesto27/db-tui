package main

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
)

const (
	exitCodeRuntime = 1
	exitCodeUsage   = 2
)

type cliDependencies struct {
	startInteractive func() error
	executeQuery     queryExecutor
	dumpDatabase     dumpExecutor
}

type commandError struct {
	err      error
	exitCode int
}

func (err *commandError) Error() string {
	return err.err.Error()
}

func (err *commandError) Unwrap() error {
	return err.err
}

func newRootCmd(dependencies cliDependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db-tui",
		Short: "Start the terminal database client",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			if err := dependencies.startInteractive(); err != nil {
				return runtimeError(err)
			}
			return nil
		},
	}
	cmd.AddCommand(
		newQueryCmd(dependencies.executeQuery),
		newDumpCmd(dependencies.dumpDatabase),
	)

	return cmd
}

func usageError(err error) error {
	return &commandError{err: err, exitCode: exitCodeUsage}
}

func runtimeError(err error) error {
	return &commandError{err: err, exitCode: exitCodeRuntime}
}

func commandExitCode(err error) int {
	var commandErr *commandError
	if errors.As(err, &commandErr) {
		return commandErr.exitCode
	}
	return exitCodeUsage
}

type queryExecutor func(context.Context, string, string, string, string) (string, error)
