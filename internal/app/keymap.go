package app

import "charm.land/bubbles/v2/key"

type keyMap struct {
	help          key.Binding
	connections   key.Binding
	newConnection key.Binding
	query         key.Binding
	tableData     key.Binding
	tableSearch   key.Binding
	executeQuery  key.Binding
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
}

func defaultKeyMap() keyMap {
	return keyMap{
		help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keyboard shortcuts")),
		connections:   key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "connections")),
		newConnection: key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new connection")),
		query:         key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "raw query")),
		tableData:     key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "table data")),
		tableSearch:   key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "search tables")),
		executeQuery:  key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "execute query")),
		queryFocus:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch focus / next field")),
		quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q/ctrl+c", "quit")),
		focusLeft:     key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "focus / scroll left")),
		focusRight:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "focus / scroll right")),
		up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous item / row")),
		down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next item / row")),
		pageUp:        key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "previous page")),
		pageDown:      key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "next page")),
		home:          key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "first table")),
		end:           key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "last table")),
		dump:          key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "dump database")),
		export:        key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "export / edit connection")),
	}
}
