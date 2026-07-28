// Command db-tui starts the terminal database client.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/app"
	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/mysql"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
	"github.com/ernestoponce27/db-tui/internal/db/sqlite"
)

func main() {
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

func connectDatabase(ctx context.Context, engine, dsn string) (db.Database, error) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case db.EnginePostgreSQL:
		return postgres.Connect(ctx, dsn)
	case db.EngineMySQL:
		return mysql.Connect(ctx, dsn)
	case db.EngineSQLite:
		return sqlite.Connect(ctx, dsn)
	default:
		return nil, fmt.Errorf("unsupported database engine %q", engine)
	}
}
