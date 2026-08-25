package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestActiveFunctionRendersHorizontalMetadataTable(t *testing.T) {
	layout := newAppLayout(100, 24)
	function := activeFunction{}
	function.activate(db.FunctionColumns{
		Name:       "customer_total",
		Arguments:  "customer_id integer",
		ReturnType: "numeric",
		Language:   "sql",
		Definition: "CREATE FUNCTION customer_total(customer_id integer) RETURNS numeric AS $$ SELECT 1 $$ LANGUAGE sql;",
	}, layout)

	view := function.view(layout, true)

	assert.Contains(t, view, "customer_total")
	assert.Contains(t, view, "Arguments")
	assert.Contains(t, view, "Return type")
	assert.Contains(t, view, "Definition")
	assert.NotContains(t, view, "Language │")
}

func TestActiveFunctionScrollClampsToDefinition(t *testing.T) {
	layout := newAppLayout(64, 16)
	function := activeFunction{}
	function.activate(db.FunctionColumns{Name: "long_function", Definition: strings.Repeat("SELECT 1\n", 40)}, layout)

	function.scroll(100, layout)
	assert.Greater(t, function.offset, 0)
	function.scroll(-100, layout)
	assert.Zero(t, function.offset)
}
