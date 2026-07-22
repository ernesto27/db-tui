package app

import "charm.land/bubbles/v2/key"

type keyMap struct {
	connections   key.Binding
	newConnection key.Binding
	query         key.Binding
	tableData     key.Binding
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
}

func defaultKeyMap() keyMap {
	return keyMap{
		connections:   key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "connections")),
		newConnection: key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new connection")),
		query:         key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "raw query")),
		tableData:     key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "table data")),
		executeQuery:  key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "execute query")),
		queryFocus:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch editor/results")),
		quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		focusLeft:     key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left/focus tables")),
		focusRight:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right/focus data")),
		up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		pageUp:        key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "previous page")),
		pageDown:      key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "next page")),
		home:          key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "first table")),
		end:           key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "last table")),
		dump:          key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "dump database")),
	}
}
