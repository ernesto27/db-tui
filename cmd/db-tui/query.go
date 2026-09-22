package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/mysql"
	"github.com/ernestoponce27/db-tui/internal/db/oracle"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
	"github.com/ernestoponce27/db-tui/internal/db/sqlite"
	"github.com/ernestoponce27/db-tui/internal/db/sqlserver"
)

type queryOptions struct {
	query      string
	dsn        string
	connection string
	format     string
	fileQuery  string
}

func newQueryCmd(execute queryExecutor) *cobra.Command {
	options := queryOptions{}
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Run a read-only query and write its result to standard output",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return runQuery(cmd.Context(), cmd, options, execute)
		},
	}
	configureQueryFlags(cmd, &options)

	return cmd
}

func configureQueryFlags(cmd *cobra.Command, options *queryOptions) {
	flags := cmd.Flags()
	flags.StringVarP(&options.query, "query", "q", "", "SQL query")
	flags.StringVarP(&options.dsn, "dsn", "c", "", "DSN")
	flags.StringVar(&options.connection, "connection", "", "saved connection name")
	flags.StringVarP(&options.format, "format", "t", db.ExportTypeJSON, "output format: json or csv")
	flags.StringVarP(&options.fileQuery, "fileQuery", "f", "", "path of file with query content")
	cmd.MarkFlagsMutuallyExclusive("query", "fileQuery")
}

func runQuery(ctx context.Context, cmd *cobra.Command, options queryOptions, execute queryExecutor) error {
	if err := validateOutputFormat(options.format); err != nil {
		return usageError(err)
	}

	query := options.query
	if options.fileQuery != "" {
		data, err := os.ReadFile(options.fileQuery)
		if err != nil {
			return err
		}
		query = string(data)
	}
	if strings.TrimSpace(query) == "" {
		return usageError(errors.New("--query or --fileQuery must be provided"))
	}

	engine, dsn, err := resolveCLIConnection(options.dsn, options.connection)
	if err != nil {
		return err
	}
	result, err := execute(ctx, engine, dsn, query, options.format)
	if err != nil {
		if options.dsn == "" {
			return savedConnectionError("query")
		}
		return runtimeError(fmt.Errorf("query: %w", err))
	}
	writeResult(cmd.OutOrStdout(), result)
	return nil
}

func executeCLI(ctx context.Context, engine, dsn, query, format string) (string, error) {
	switch engine {
	case db.EnginePostgreSQL:
		return postgres.ExecuteCLI(ctx, dsn, query, format)
	case db.EngineMySQL:
		return mysql.ExecuteCLI(ctx, dsn, query, format)
	case db.EngineOracle:
		return oracle.ExecuteCLI(ctx, dsn, query, format)
	case db.EngineSQLServer:
		return sqlserver.ExecuteCLI(ctx, dsn, query, format)
	case db.EngineSQLite:
		return sqlite.ExecuteCLI(ctx, dsn, query, format)
	default:
		return "", errors.New("unsupported DSN")
	}
}

func validateOutputFormat(format string) error {
	switch format {
	case db.ExportTypeJSON, db.ExportTypeCSV:
		return nil
	default:
		return fmt.Errorf("unsupported output format %q; want %q or %q", format, db.ExportTypeJSON, db.ExportTypeCSV)
	}
}

func writeResult(stdout io.Writer, result string) {
	if strings.HasSuffix(result, "\n") {
		_, _ = fmt.Fprint(stdout, result)
		return
	}
	_, _ = fmt.Fprintln(stdout, result)
}
