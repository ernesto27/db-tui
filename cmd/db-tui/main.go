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
	var savedConnection app.ConnectionSettings
	applicationConfig, startErr := config.Load()
	if startErr == nil {
		savedConnection = appConnectionSettings(*applicationConfig.PostgreSQL)
		var dsn string
		dsn, startErr = applicationConfig.PostgreSQL.ConnectionDSN()
		if startErr == nil {
			ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
			database, startErr = postgres.Connect(ctx, dsn)
			cancel()
			if database != nil {
				databaseName = database.Name()
			}
		}
	}

	model := app.New(database, databaseName, startErr, savedConnection, postgres.Connect, saveConnectionSettings)
	finalModel, err := tea.NewProgram(model).Run()
	if finalApp, ok := finalModel.(app.Model); ok {
		finalApp.Close()
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "db-tui: %v\n", err)
		os.Exit(1)
	}
}

func appConnectionSettings(postgreSQLConfig postgres.PostgreSQLConfig) app.ConnectionSettings {
	return app.ConnectionSettings{
		DSN:          postgreSQLConfig.DSN,
		Host:         postgreSQLConfig.Host,
		Port:         postgreSQLConfig.Port,
		DatabaseName: postgreSQLConfig.DatabaseName,
		Username:     postgreSQLConfig.Username,
		Password:     postgreSQLConfig.Password,
	}
}

func saveConnectionSettings(settings app.ConnectionSettings) error {
	return config.SavePostgreSQLConfig(postgres.PostgreSQLConfig{
		DSN:          settings.DSN,
		Host:         settings.Host,
		Port:         settings.Port,
		DatabaseName: settings.DatabaseName,
		Username:     settings.Username,
		Password:     settings.Password,
	})
}
