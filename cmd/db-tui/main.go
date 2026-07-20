// Command db-tui starts the terminal database client.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/app"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
)

const (
	chinookDSN        = "postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable"
	connectionTimeout = 5 * time.Second
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	database, connectErr := postgres.Connect(ctx, chinookDSN)
	cancel()
	if database != nil {
		defer database.Close()
	}

	if _, err := tea.NewProgram(app.New(database, "chinook", connectErr)).Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "db-tui: %v\n", err)
		os.Exit(1)
	}
}
