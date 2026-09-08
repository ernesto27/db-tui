// Command db-tui starts the terminal database client.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/app"
	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/mysql"
	"github.com/ernestoponce27/db-tui/internal/db/oracle"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
	"github.com/ernestoponce27/db-tui/internal/db/sqlite"
	"github.com/ernestoponce27/db-tui/internal/db/sqlserver"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handled, exitCode := runCLI(ctx, os.Args[1:], os.Stdout, os.Stderr)
	if handled {
		os.Exit(exitCode)
	}

	appConfig, err := config.Load()
	if err != nil {
		panic(err)
	}

	model := app.New(appConfig, app.ConnectionSettings{}, connectDatabase)
	finalModel, err := tea.NewProgram(model).Run()
	if finalApp, ok := finalModel.(app.Model); ok {
		finalApp.Close()
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "db-tui: %v\n", err)
		os.Exit(1)
	}
}

func runCLI(ctx context.Context, args []string, stdout, stderr io.Writer) (handled bool, exitCode int) {
	flags := flag.NewFlagSet("db-tui", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	query := flags.String("q", "", "SQL query")
	dsn := flags.String("c", "", "DSN")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(stdout, "Usage: db-tui [-q <SQL> -c <DSN>]")
			return true, 0
		}
		_, _ = fmt.Fprintf(stderr, "db-tui: %v\n", err)
		return true, 2
	}
	if len(flags.Args()) != 0 {
		_, _ = fmt.Fprintf(stderr, "db-tui: unexpected arguments: %s\n", strings.Join(flags.Args(), " "))
		return true, 2
	}

	queryMissing := strings.TrimSpace(*query) == ""
	dsnMissing := strings.TrimSpace(*dsn) == ""
	if queryMissing && dsnMissing {
		return false, 0
	}
	if queryMissing || dsnMissing {
		_, _ = fmt.Fprintln(stderr, "db-tui: -q and -c must be provided together")
		return true, 2
	}

	engine := detectEngine(*dsn)

	var result string
	var err error
	switch engine {
	case db.EnginePostgreSQL:
		result, err = postgres.ExecuteCLI(ctx, *dsn, *query)
	case db.EngineMySQL:
		result, err = mysql.ExecuteCLI(ctx, *dsn, *query)
	default:
		_, _ = fmt.Fprintf(stderr, "db-tui: unsupported DSN: %s\n", *dsn)
		return true, 2
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "db-tui: query: %v\n", err)
		return true, 1
	}

	_, _ = fmt.Fprintln(stdout, result)
	return true, 0
}

func detectEngine(dsn string) string {
	if strings.HasPrefix(strings.ToLower(dsn), "mysql://") {
		return db.EngineMySQL
	}
	// Default to PostgreSQL for backward compatibility
	return db.EnginePostgreSQL
}

func connectDatabase(ctx context.Context, engine, dsn string) (db.Database, error) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case db.EnginePostgreSQL:
		return postgres.Connect(ctx, dsn)
	case db.EngineMySQL:
		return mysql.Connect(ctx, dsn)
	case db.EngineOracle:
		return oracle.Connect(ctx, dsn)
	case db.EngineSQLite:
		return sqlite.Connect(ctx, dsn)
	case db.EngineSQLServer:
		return sqlserver.Connect(ctx, dsn)
	default:
		return nil, fmt.Errorf("unsupported database engine %q", engine)
	}
}
