package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ernestoponce27/db-tui/internal/config"
	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/version"
)

func TestBaseViewShowsVersion(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)

	view := model.baseView()

	assert.Contains(t, view.Content, "db-tui v"+version.Version())
}

func TestBaseViewShowsConnectedEngine(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.databaseName = "chinook"
	model.databaseEngine = db.EngineMySQL

	view := model.baseView()

	assert.Contains(t, view.Content, "chinook  /  MySQL")
}
