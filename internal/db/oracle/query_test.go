package oracle_test

import (
	"context"
	"testing"
	"time"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/db/oracle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantColumns []string
		wantRows    int
		wantTag     string
		wantErr     bool
	}{
		{
			name:        "returns rows",
			sql:         "SELECT 7 AS value FROM dual",
			wantColumns: []string{"VALUE"},
			wantRows:    1,
			wantTag:     "SELECT",
		},
		{
			name:    "returns command tag",
			sql:     "CREATE TABLE raw_query_test (id NUMBER)",
			wantTag: "CREATE TABLE",
		},
		{
			name:        "limits rows",
			sql:         "SELECT level AS value FROM dual CONNECT BY level <= 101",
			wantColumns: []string{"VALUE"},
			wantRows:    db.MaxPageSize,
			wantTag:     "SELECT",
		},
		{
			name:    "wraps database errors",
			sql:     "SELECT * FROM raw_query_table_that_does_not_exist",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			database := connectQueryTestDatabase(t)
			if test.name == "returns command tag" {
				_, _ = database.Execute(ctx, "DROP TABLE raw_query_test PURGE")
				t.Cleanup(func() {
					cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cleanupCancel()
					_, _ = database.Execute(cleanupCtx, "DROP TABLE raw_query_test PURGE")
				})
			}

			result, err := database.Execute(ctx, test.sql)

			if test.wantErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "execute Oracle query")
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
		})
	}
}

func connectQueryTestDatabase(t *testing.T) db.Database {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	database, err := oracle.Connect(ctx, freePDBDSN)
	require.NoError(t, err)
	t.Cleanup(database.Close)
	return database
}
