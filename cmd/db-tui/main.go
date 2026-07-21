// Command db-tui starts the terminal database client.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/app"
	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		panic(err)
	}

	model := app.New(appConfig, app.ConnectionSettings{}, postgres.Connect)
	finalModel, err := tea.NewProgram(model).Run()
	if finalApp, ok := finalModel.(app.Model); ok {
		finalApp.Close()
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "db-tui: %v\n", err)
		os.Exit(1)
	}
}
