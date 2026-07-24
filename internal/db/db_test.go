package db_test

import (
	"regexp"
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestTimestampedFilename(t *testing.T) {
	filename := db.TimestampedFilename("chinook", "sql")

	assert.Regexp(t, regexp.MustCompile(`^chinook_\d{8}_\d{6}\.sql$`), filename)
}

func TestSafeFilename(t *testing.T) {
	tests := map[string]string{
		"chinook":       "chinook",
		"..":            "export",
		"database name": "database_name",
		"../../passwd":  "passwd",
		"orders/2024":   "2024",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, want, db.SafeFilename(input))
		})
	}
}

func TestValidateSelectQuery(t *testing.T) {
	assert.NoError(t, db.ValidateSelectQuery("SELECT 1"))
	assert.NoError(t, db.ValidateSelectQuery("-- report\nSELECT 1"))
	assert.NoError(t, db.ValidateSelectQuery("/* report */ SELECT 1"))
	assert.EqualError(t, db.ValidateSelectQuery("UPDATE Album SET Title = 'x'"), "only SELECT queries can be exported")
}
