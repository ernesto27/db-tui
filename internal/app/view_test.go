package app

import (
	"errors"
	"strings"
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

func TestBaseViewShowsReadOnlyMode(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook", engine: db.EngineMySQL}
	model.readOnly = true

	view := model.baseView()

	assert.Contains(t, view.Content, readOnlyModeText)
	assert.Contains(t, view.Content, readOnlyHeaderLabel())
}

func TestViewRendersRawQueryDeleteConfirmation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	modal := newRawQueryDeleteModal("DELETE FROM album")
	model.rawQueryDeleteModal = &modal

	view := model.View()

	assert.Contains(t, view.Content, "Confirm DELETE")
	assert.Contains(t, view.Content, "This query will delete data.")
}

func TestBaseViewHidesNavigatorContentWithoutDatabase(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)

	view := model.baseView()

	assert.NotContains(t, view.Content, "● ")
	assert.NotContains(t, view.Content, "Filter:")
	assert.NotContains(t, view.Content, "Tables")
}

func TestBaseViewShowsNavigatorContentWithDatabase(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}

	view := model.baseView()

	assert.Contains(t, view.Content, "● chinook")
	assert.Contains(t, view.Content, "Filter:")
	assert.Contains(t, view.Content, "Tables")
}

func TestBaseViewShowsReconnectFeedback(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.reconnecting = true

	assert.Contains(t, model.baseView().Content, "Reconnecting…")

	model.reconnecting = false
	model.reconnectErr = errors.New("connection failed\x1b")
	view := model.baseView()
	assert.Contains(t, view.Content, "Unable to reconnect")
	assert.Contains(t, view.Content, "connection failed�")
}

func TestBaseViewShowsOracleEngine(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "FREEPDB1", engine: db.EngineOracle}

	view := model.baseView()

	assert.Contains(t, view.Content, "FREEPDB1  /  Oracle")
}

func TestBaseViewOmitsHostSeparatorWhenHostIsEmpty(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook.db", engine: db.EngineSQLite}

	view := model.baseView()

	assert.Contains(t, view.Content, "chinook.db  /  SQLite")
	assert.NotContains(t, view.Content, "SQLite  /  ")
}

func TestBaseViewShowsTestingEnvironment(t *testing.T) {
	environment := config.ConnectionEnvironmentTesting
	model := New(config.Config{Connections: []config.Connection{{Environment: environment}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0

	view := model.baseView()

	assert.Contains(t, view.Content, strings.ToUpper(string(environment)))
	assert.Equal(t, colorTestingHeaderBackground, headerBackgroundForEnvironment(model.activeConnectionEnvironment()))
}

func TestBaseViewShowsProductionEnvironment(t *testing.T) {
	environment := config.ConnectionEnvironmentProduction
	model := New(config.Config{Connections: []config.Connection{{Environment: environment}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0

	view := model.baseView()

	assert.Contains(t, view.Content, strings.ToUpper(string(environment)))
	assert.Equal(t, colorProductionHeaderBackground, headerBackgroundForEnvironment(model.activeConnectionEnvironment()))
}

func TestBaseViewShowsUnrecognizedEnvironmentLabel(t *testing.T) {
	environment := config.ConnectionEnvironmentTesting + config.ConnectionEnvironmentProduction
	model := New(config.Config{Connections: []config.Connection{{Environment: environment}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0

	view := model.baseView()

	assert.Contains(t, view.Content, strings.ToUpper(string(environment)))
	assert.Equal(t, colorHeaderBackground, headerBackgroundForEnvironment(model.activeConnectionEnvironment()))
}

func TestBaseViewKeepsDefaultHeaderForUnclassifiedConnection(t *testing.T) {
	model := New(config.Config{Connections: []config.Connection{{}}}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.activeConnectionIndex = 0

	view := model.baseView()

	assert.NotContains(t, view.Content, strings.ToUpper(string(config.ConnectionEnvironmentTesting)))
	assert.NotContains(t, view.Content, strings.ToUpper(string(config.ConnectionEnvironmentProduction)))
	assert.Equal(t, colorHeaderBackground, headerBackgroundForEnvironment(model.activeConnectionEnvironment()))
}

func TestFooterTextDescribesTabNavigation(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)
	model.database = &fakeDatabase{name: "chinook"}
	model.navigator.tables = []db.Table{{Name: "Album"}}

	assert.Contains(t, model.footerText(), "Tab navigator/data")
	assert.NotContains(t, model.footerText(), "r refresh")
	assert.Contains(t, model.footerText(), "r reconnect")

	model.focus = focusData
	assert.Contains(t, model.footerText(), "r refresh")

	model.panel = panelQuery
	assert.NotContains(t, model.footerText(), "r refresh")
	assert.Equal(t, "Ctrl+P execute  •  Alt+R read only  •  Ctrl+K shortcuts  •  q quit", model.footerText())
}

func TestShortcutsModalDocumentsNavigatorReconnect(t *testing.T) {
	model := New(config.Config{}, ConnectionSettings{}, nil)

	assert.Contains(t, strings.Join(newShortcutsModal(model.layout).lines(model.layout), "\n"), "Reconnect database from navigator")
	assert.Contains(t, strings.Join(newShortcutsModal(model.layout).lines(model.layout), "\n"), "Toggle session read-only mode")
}
