package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAppLayout(t *testing.T) {
	tests := []struct {
		name               string
		width              int
		height             int
		wantWidth          int
		wantHeight         int
		wantNavigatorWidth int
		wantNavigatorRows  int
	}{
		{
			name:               "normal terminal",
			width:              100,
			height:             24,
			wantWidth:          100,
			wantHeight:         24,
			wantNavigatorWidth: 26,
			wantNavigatorRows:  16,
		},
		{
			name:               "narrow terminal",
			width:              79,
			height:             20,
			wantWidth:          79,
			wantHeight:         20,
			wantNavigatorWidth: 20,
			wantNavigatorRows:  12,
		},
		{
			name:               "minimum dimensions",
			width:              20,
			height:             5,
			wantWidth:          64,
			wantHeight:         16,
			wantNavigatorWidth: 20,
			wantNavigatorRows:  8,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			layout := newAppLayout(test.width, test.height)

			assert.Equal(t, test.wantWidth, layout.width)
			assert.Equal(t, test.wantHeight, layout.height)
			assert.Equal(t, test.wantNavigatorWidth, layout.navigator.width)
			assert.Equal(t, test.wantNavigatorRows, layout.navigatorListRows)
			assert.Equal(t, layout.height-4, layout.bodyHeight)
			assert.Equal(t, layout.width-layout.navigator.width-1, layout.data.width)
			assert.Equal(t, layout.bodyHeight, layout.navigator.height)
			assert.Equal(t, layout.bodyHeight, layout.data.height)
		})
	}
}

func TestAppLayoutMouseInNavigator(t *testing.T) {
	layout := newAppLayout(100, 24)

	tests := []struct {
		name string
		x    int
		want bool
	}{
		{name: "left of terminal", x: -1},
		{name: "left border", x: 0, want: true},
		{name: "last navigator cell", x: layout.navigator.width - 1, want: true},
		{name: "data pane", x: layout.navigator.width},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, layout.mouseInNavigator(test.x))
		})
	}
}

func TestAppLayoutClickableNavigatorX(t *testing.T) {
	layout := newAppLayout(100, 24)

	tests := []struct {
		name string
		x    int
		want bool
	}{
		{name: "left of terminal", x: -1},
		{name: "left border", x: 0},
		{name: "first content cell", x: 1, want: true},
		{name: "last content cell", x: layout.navigator.width - 2, want: true},
		{name: "right border", x: layout.navigator.width - 1},
		{name: "data pane", x: layout.navigator.width},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, layout.clickableNavigatorX(test.x))
		})
	}
}
