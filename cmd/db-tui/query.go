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
	query  string
	dsn    string
	format string
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
	flags.StringVarP(&options.format, "format", "t", db.ExportTypeJSON, "output format: json or csv")
	cmd.MarkFlagsRequiredTogether("query", "dsn")
}

func runQuery(ctx context.Context, cmd *cobra.Command, options queryOptions, execute queryExecutor) error {
	if strings.TrimSpace(options.query) == "" || strings.TrimSpace(options.dsn) == "" {
		return usageError(errors.New("--query and --dsn must be provided together"))
	}
	if err := validateOutputFormat(options.format); err != nil {
		return usageError(err)
	}

	engine, err := detectEngine(options.dsn)
	if err != nil {
		return usageError(err)
	}
	result, err := execute(ctx, engine, options.dsn, options.query, options.format)
	if err != nil {
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

func detectEngine(dsn string) (string, error) {
	dsn = strings.TrimSpace(dsn)
	lowerDSN := strings.ToLower(dsn)

	switch {
	case strings.HasPrefix(lowerDSN, "postgres://"), strings.HasPrefix(lowerDSN, "postgresql://"):
		return db.EnginePostgreSQL, nil
	case strings.HasPrefix(lowerDSN, "mysql://"):
		return db.EngineMySQL, nil
	case strings.HasPrefix(lowerDSN, "oracle://"):
		return db.EngineOracle, nil
	case strings.HasPrefix(lowerDSN, "sqlserver://"):
		return db.EngineSQLServer, nil
	case isRegularFile(dsn):
		return db.EngineSQLite, nil
	default:
		return "", errors.New("unsupported DSN")
	}
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
