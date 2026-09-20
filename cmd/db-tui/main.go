// Command db-tui starts the terminal database client.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

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

	cmd := newRootCmd(cliDependencies{
		startInteractive: runInteractive,
		executeQuery:     executeCLI,
		dumpDatabase:     runDump,
	})
	cmd.SetArgs(os.Args[1:])
	cmd.SetErr(os.Stderr)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(commandExitCode(err))
	}
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
