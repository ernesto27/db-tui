package db_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestSerializeCLIResult(t *testing.T) {
	t.Parallel()

	columns := []string{"id", "name"}
	rows := [][]any{{1, "Doe, Jane"}, {2, nil}}

	tests := []struct {
		name    string
		engine  string
		columns []string
		rows    [][]any
		format  string
		want    string
		wantErr string
	}{
		{
			name:    "serializes JSON",
			engine:  db.EngineMySQL,
			columns: columns,
			rows:    rows,
			format:  db.ExportTypeJSON,
			want: `[
  {
    "id": 1,
    "name": "Doe, Jane"
  },
  {
    "id": 2,
    "name": null
  }
]`,
		},
		{
			name:    "serializes CSV",
			engine:  db.EngineMySQL,
			columns: columns,
			rows:    rows,
			format:  db.ExportTypeCSV,
			want:    "id,name\n1,\"Doe, Jane\"\n2,\n",
		},
		{
			name:    "serializes an empty result",
			engine:  db.EngineSQLite,
			columns: columns,
			format:  db.ExportTypeCSV,
			want:    "id,name\n",
		},
		{
			name:    "names PostgreSQL in an unsupported format error",
			engine:  db.EnginePostgreSQL,
			columns: columns,
			rows:    rows,
			format:  "xml",
			wantErr: `unsupported ` + db.EnginePostgreSQL + ` CLI output format "xml"`,
		},
		{
			name:    "names MySQL in an unsupported format error",
			engine:  db.EngineMySQL,
			columns: columns,
			rows:    rows,
			format:  "xml",
			wantErr: `unsupported ` + db.EngineMySQL + ` CLI output format "xml"`,
		},
		{
			name:    "names Oracle in an unsupported format error",
			engine:  db.EngineOracle,
			columns: columns,
			rows:    rows,
			format:  "xml",
			wantErr: `unsupported ` + db.EngineOracle + ` CLI output format "xml"`,
		},
		{
			name:    "names SQLite in an unsupported format error",
			engine:  db.EngineSQLite,
			columns: columns,
			rows:    rows,
			format:  "xml",
			wantErr: `unsupported ` + db.EngineSQLite + ` CLI output format "xml"`,
		},
		{
			name:    "names SQL Server in an unsupported format error",
			engine:  db.EngineSQLServer,
			columns: columns,
			rows:    rows,
			format:  "xml",
			wantErr: `unsupported ` + db.EngineSQLServer + ` CLI output format "xml"`,
		},
		{
			name:    "rejects an empty format",
			engine:  db.EngineSQLite,
			columns: columns,
			rows:    rows,
			wantErr: `unsupported ` + db.EngineSQLite + ` CLI output format ""`,
		},
		{
			name:    "names the engine in a CSV serialization error",
			engine:  db.EngineSQLServer,
			columns: columns,
			rows:    [][]any{{1}},
			format:  db.ExportTypeCSV,
			wantErr: "encode " + db.EngineSQLServer + " query CSV: CSV row 1 has 1 values; want 2",
		},
		{
			name:    "names the engine in a JSON serialization error",
			engine:  db.EngineOracle,
			columns: columns,
			rows:    [][]any{{1}},
			format:  db.ExportTypeJSON,
			wantErr: "encode " + db.EngineOracle + " query JSON: JSON row 1 has 1 values; want 2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := db.SerializeCLIResult(test.engine, test.columns, test.rows, test.format)

			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				assert.Empty(t, result)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, test.want, result)
		})
	}
}

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

func TestJSONValueMarshalJSON(t *testing.T) {
	encoded, err := json.Marshal(db.JSONValue(`{"id":"1","name":"Coffee"}`))

	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":"1","name":"Coffee"}`, string(encoded))

	_, err = json.Marshal(db.JSONValue("not JSON"))
	assert.Error(t, err)
}
