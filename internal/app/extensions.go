package app

import "github.com/ernestoponce27/db-tui/internal/db"

const (
	extensionPanelTitle    = "Extensions"
	extensionLoadingText   = "Loading extensions…"
	extensionLoadErrorText = "Unable to load extensions:"
	noExtensionsText       = "No extensions found."
	extensionNameColumn    = "Name"
	extensionSchemaColumn  = "Schema"
	extensionVersionColumn = "Version"
)

// activeExtensions owns PostgreSQL extension metadata shown in the data panel.
// It is separate from activeRelation because extensions are not relations.
type activeExtensions struct {
	request uint64
	set     bool
}

func extensionRowPage(extensions []db.ExtensionData) db.RowPage {
	page := db.RowPage{
		Columns: []string{extensionNameColumn, extensionSchemaColumn, extensionVersionColumn},
		Rows:    make([][]any, 0, len(extensions)),
	}
	for _, extension := range extensions {
		page.Rows = append(page.Rows, []any{extension.Name, extension.Schema, extension.Version})
	}
	return page
}
