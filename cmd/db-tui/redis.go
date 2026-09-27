package main

import (
	"github.com/ernestoponce27/db-tui/internal/redis"
	"github.com/spf13/cobra"
)

func newRedisCmd() *cobra.Command {
	var dsn, command string
	cmd := &cobra.Command{
		Use:   "redis",
		Short: "Run redis command",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			client, err := redis.Connect(cmd.Context(), redis.Settings{DSN: dsn})
			if err != nil {
				return err
			}
			defer client.Close()

			result, err := client.Execute(cmd.Context(), redis.DatabaseFromDSN, command)
			if err != nil {
				return err
			}
			writeResult(cmd.OutOrStdout(), result.Raw)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&dsn, "dsn", "d", "", "DSN")
	flags.StringVarP(&command, "command", "c", "", "command to run")

	return cmd
}
