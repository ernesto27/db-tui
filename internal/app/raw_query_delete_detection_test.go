package app

import "testing"

import "github.com/stretchr/testify/assert"

func TestRawQueryContainsDelete(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{name: "direct delete", sql: "DELETE FROM album", want: true},
		{name: "mixed case and comments", sql: "-- remove old rows\nDeLeTe FROM album", want: true},
		{name: "cte final delete", sql: "WITH stale AS (SELECT id FROM album) DELETE FROM album WHERE id IN (SELECT id FROM stale)", want: true},
		{name: "cte with column list", sql: "WITH stale(id) AS (SELECT id FROM album) DELETE FROM album WHERE id IN (SELECT id FROM stale)", want: true},
		{name: "data modifying cte", sql: "WITH removed AS (DELETE FROM album RETURNING id) SELECT * FROM removed", want: true},
		{name: "batch delete", sql: "SELECT 1; DELETE FROM album", want: true},
		{name: "nested cte delete", sql: "WITH outer_cte AS (WITH inner_cte AS (SELECT 1) DELETE FROM album) SELECT * FROM outer_cte", want: true},
		{name: "comment only", sql: "/* DELETE FROM album */ SELECT 1", want: false},
		{name: "string literal", sql: "SELECT 'DELETE FROM album'", want: false},
		{name: "dollar quoted string", sql: "SELECT $$DELETE FROM album$$", want: false},
		{name: "quoted identifier", sql: "SELECT \"DELETE\" FROM album", want: false},
		{name: "unquoted identifier", sql: "SELECT delete FROM audit_log", want: false},
		{name: "cte identifier", sql: "WITH delete AS (SELECT 1) SELECT * FROM delete", want: false},
		{name: "other command", sql: "UPDATE album SET title = 'DELETE'", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, rawQueryContainsDelete(test.sql))
		})
	}
}
