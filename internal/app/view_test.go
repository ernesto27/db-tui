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
	model.database = &fakeDatabase{name: "chinook", engine: db.EngineMySQL}

	view := model.baseView()

	assert.Contains(t, view.Content, "chinook  /  MySQL")
}

func TestBaseViewOmitsHostSeparatorWhenHostIsEmpty(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook.db", engine: db.EngineSQLite}

	view := model.baseView()

	assert.Contains(t, view.Content, "chinook.db  /  SQLite")
	assert.NotContains(t, view.Content, "SQLite  /  ")
}

func TestFooterTextDescribesTabNavigation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}

	assert.Contains(t, model.footerText(), "Tab table/search/data")

	model.panel = panelQuery
	assert.Contains(t, model.footerText(), "Tab editor/results")
}
