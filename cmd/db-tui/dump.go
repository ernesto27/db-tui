package main

import (
	"context"

	"github.com/spf13/cobra"
)

type dumpExecutor func(context.Context, string) error

func newDumpCmd(dump dumpExecutor) *cobra.Command {
	var dsn string
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump the database identified by a DSN",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return dump(cmd.Context(), dsn)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&dsn, "dsn", "d", "", "DSN")
	_ = cmd.MarkFlagRequired("dsn")
	return cmd
}

func runDump(ctx context.Context, dsn string) error {
	engine, err := detectEngine(dsn)
	if err != nil {
		return usageError(err)
	}

	dbInstance, err := connectDatabase(ctx, engine, dsn)
	if err != nil {
		return usageError(err)
	}
	defer dbInstance.Close()

	return dbInstance.Dump(ctx)
}
