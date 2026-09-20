// Command db-tui starts the terminal database client.
package main

import (
	"context"
	"os"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := newRootCmd(cliDependencies{
		startInteractive: runInteractive,
		executeQuery:     executeCLI,
	})
	cmd.SetArgs(os.Args[1:])
	cmd.SetErr(os.Stderr)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(commandExitCode(err))
	}
}
