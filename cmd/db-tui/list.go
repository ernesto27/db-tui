package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/spf13/cobra"
)

func newListConnectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show connectons saved",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			appConfig, err := config.LoadExisting()
			if err != nil {
				return err
			}

			if len(appConfig.Connections) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No saved connections.")
				return err
			}

			table := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(table, "NAME\tENGINE\tENVIRONMENT")

			for _, connection := range appConfig.Connections {
				environment := string(connection.Environment)
				if environment == "" {
					environment = "-"
				}

				fmt.Fprintf(table, "%s\t%s\t%s\n",
					connection.Name,
					connection.Engine,
					environment,
				)

			}
			return table.Flush()
		},
	}

	return cmd
}
