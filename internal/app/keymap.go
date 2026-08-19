package app

import "charm.land/bubbles/v2/key"

type keyMap struct {
	connections   key.Binding
	newConnection key.Binding
	query         key.Binding
	tableData     key.Binding
	tableDDL      key.Binding
	tableSearch   key.Binding
	settings      key.Binding
	activate      key.Binding
	executeQuery  key.Binding
	sqlScripts    key.Binding
	queryFocus    key.Binding
	quit          key.Binding
	focusLeft     key.Binding
	focusRight    key.Binding
	up            key.Binding
	down          key.Binding
	pageUp        key.Binding
	pageDown      key.Binding
	home          key.Binding
	end           key.Binding
	dump          key.Binding
	export        key.Binding
	editRow       key.Binding
	deleteRow     key.Binding
	refreshTable  key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		connections:   key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "connections")),
		newConnection: key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new connection")),
		query:         key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "raw query")),
		tableData:     key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "table data")),
		tableDDL:      key.NewBinding(key.WithKeys("ctrl+g"), key.WithHelp("ctrl+g", "actions")),
		tableSearch:   key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "search tables")),
		settings:      key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "settings")),
		activate:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "load rows")),
		executeQuery:  key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "execute query")),
		sqlScripts:    key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "saved scripts")),
		queryFocus:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch editor/results")),
		quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		focusLeft:     key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
		focusRight:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
		up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		pageUp:        key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "previous page")),
		pageDown:      key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "next page")),
		home:          key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "first table")),
		end:           key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "last table")),
		dump:          key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "dump database")),
		export:        key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "export")),
		editRow:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit row")),
		deleteRow:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete row")),
		refreshTable:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh table")),
	}
}
