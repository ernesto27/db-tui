package app

import "charm.land/bubbles/v2/key"

type keyMap struct {
	connect    key.Binding
	quit       key.Binding
	focusLeft  key.Binding
	focusRight key.Binding
	up         key.Binding
	down       key.Binding
	pageUp     key.Binding
	pageDown   key.Binding
	home       key.Binding
	end        key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		connect:    key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "connect")),
		quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		focusLeft:  key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left/focus tables")),
		focusRight: key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right/focus data")),
		up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		pageUp:     key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "previous page")),
		pageDown:   key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "next page")),
		home:       key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "first table")),
		end:        key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "last table")),
	}
}
