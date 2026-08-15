package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantColumns []string
		wantRows    int
		wantFirst   any
		wantLast    any
		wantTag     string
		wantErr     bool
	}{
		{
			name:        "returns rows",
			sql:         "SELECT 7 AS number",
			wantColumns: []string{"number"},
			wantRows:    1,
			wantFirst:   int32(7),
			wantLast:    int32(7),
			wantTag:     "SELECT 1",
		},
		{
			name:    "returns command tag",
			sql:     "CREATE TEMPORARY TABLE raw_query_test (id integer)",
			wantTag: "CREATE TABLE",
		},
		{
			name:        "limits rows",
			sql:         "SELECT generate_series(1, 101) AS number",
			wantColumns: []string{"number"},
			wantRows:    db.MaxPageSize,
			wantFirst:   int32(1),
			wantLast:    int32(db.MaxPageSize),
			wantTag:     "SELECT 101",
		},
		{
			name:    "wraps database errors",
			sql:     "SELECT * FROM raw_query_table_that_does_not_exist",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := connectQueryTestDatabase(t)

			result, err := database.Execute(context.Background(), test.sql)

			if test.wantErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "execute PostgreSQL query")
				return
			}
			require.NoError(t, err)
			if len(test.wantColumns) == 0 {
				assert.Empty(t, result.Columns)
			} else {
				assert.Equal(t, test.wantColumns, result.Columns)
			}
			assert.Len(t, result.Rows, test.wantRows)
			assert.Equal(t, test.wantTag, result.CommandTag)
			if test.wantRows > 0 {
				assert.Equal(t, test.wantFirst, result.Rows[0][0])
				assert.Equal(t, test.wantLast, result.Rows[len(result.Rows)-1][0])
			}
		})
	}
}

func connectQueryTestDatabase(t *testing.T) db.Database {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	database, err := postgres.Connect(ctx, chinookDSN)
	require.NoError(t, err)
	t.Cleanup(database.Close)
	return database
}
