package main

import (
	"context"

	"github.com/spf13/cobra"
)

type dumpExecutor func(context.Context, string, string) error

func newDumpCmd(dump dumpExecutor) *cobra.Command {
	var dsn, connection string
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump the database identified by a DSN or saved connection",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			engine, resolvedDSN, err := resolveCLIConnection(dsn, connection)
			if err != nil {
				return err
			}
			if err := dump(cmd.Context(), engine, resolvedDSN); err != nil {
				if dsn == "" {
					return savedConnectionError("dump")
				}
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&dsn, "dsn", "d", "", "DSN")
	flags.StringVar(&connection, "connection", "", "saved connection name")
	return cmd
}

func runDump(ctx context.Context, engine, dsn string) error {
	dbInstance, err := connectDatabase(ctx, engine, dsn)
	if err != nil {
		return usageError(err)
	}
	defer dbInstance.Close()

	return dbInstance.Dump(ctx)
}
