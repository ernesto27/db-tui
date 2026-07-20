// Command db-tui starts the terminal database client.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/app"
	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
)

const connectionTimeout = 5 * time.Second

func main() {
	var database db.Database
	var databaseName string
	config, startErr := config.Load()
	if startErr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
		database, startErr = postgres.Connect(ctx, config.PostgreSQL.DSN)
		cancel()
		if database != nil {
			databaseName = database.Name()
		}
	}
	if database != nil {
		defer database.Close()
	}

	if _, err := tea.NewProgram(app.New(database, databaseName, startErr)).Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "db-tui: %v\n", err)
		os.Exit(1)
	}
}
