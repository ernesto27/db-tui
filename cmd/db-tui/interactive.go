package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ernestoponce27/db-tui/internal/app"
	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/redis"
)

func runInteractive() error {
	appConfig, err := config.Load()
	if err != nil {
		panic(err)
	}

	model := app.New(appConfig, app.ConnectionSettings{}, connectDatabase, redis.Connect)
	finalModel, err := tea.NewProgram(model).Run()
	if finalApp, ok := finalModel.(app.Model); ok {
		finalApp.Close()
	}
	return err
}
