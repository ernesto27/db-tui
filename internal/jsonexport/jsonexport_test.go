package jsonexport

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCreatesJSONFromMockData(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "albums.json")
	createdAt := time.Date(2026, time.July, 24, 12, 30, 45, 123000000, time.UTC)

	err := Write(path, "Album",
		[]string{"AlbumId", "Title", "Available", "CreatedAt", "Notes", "Raw"},
		[][]any{
			{1, "ACME, Inc.", true, createdAt, nil, []byte("binary as text")},
			{2, "A \"quoted\" title\nwith a new line", false, createdAt, "", []byte("second row")},
		},
	)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var document map[string][]map[string]any
	require.NoError(t, json.Unmarshal(data, &document))

	assert.Equal(t, []map[string]any{
		{
			"AlbumId":   float64(1),
			"Title":     "ACME, Inc.",
			"Available": true,
			"CreatedAt": "2026-07-24T12:30:45.123Z",
			"Notes":     nil,
			"Raw":       "YmluYXJ5IGFzIHRleHQ=",
		},
		{
			"AlbumId":   float64(2),
			"Title":     "A \"quoted\" title\nwith a new line",
			"Available": false,
			"CreatedAt": "2026-07-24T12:30:45.123Z",
			"Notes":     "",
			"Raw":       "c2Vjb25kIHJvdw==",
		},
	}, document["Album"])
}

func TestWriteNormalizesValuesUnsupportedByJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "values.json")
	farFuture := time.Date(10000, time.January, 2, 3, 4, 5, 0, time.UTC)

	err := Write(path, "Measurement",
		[]string{"NaN", "PositiveInfinity", "NegativeInfinity", "FarFuture"},
		[][]any{{math.NaN(), math.Inf(1), math.Inf(-1), farFuture}},
	)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var document map[string][]map[string]any
	require.NoError(t, json.Unmarshal(data, &document))
	assert.Equal(t, map[string]any{
		"NaN":              "NaN",
		"PositiveInfinity": "+Inf",
		"NegativeInfinity": "-Inf",
		"FarFuture":        farFuture.Format(time.RFC3339Nano),
	}, document["Measurement"][0])
}

func TestWriteNormalizesUUIDByteArray(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "uuid.json")
	uuid := [16]byte{0x8f, 0xdc, 0x4a, 0x11, 0xb6, 0x41, 0x4a, 0xc3, 0x82, 0x4b, 0x91, 0x2e, 0x34, 0xae, 0xd1, 0x7b}

	err := Write(path, "Event", []string{"ID"}, [][]any{{uuid}})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var document map[string][]map[string]any
	require.NoError(t, json.Unmarshal(data, &document))
	assert.Equal(t, "8fdc4a11-b641-4ac3-824b-912e34aed17b", document["Event"][0]["ID"])
}

func TestWriteRejectsEmptyPath(t *testing.T) {
	t.Parallel()

	err := Write("", "Album", []string{"id"}, nil)

	assert.EqualError(t, err, "JSON export path is required")
}

func TestWriteRejectsEmptyTableName(t *testing.T) {
	t.Parallel()

	err := Write(filepath.Join(t.TempDir(), "albums.json"), "", []string{"id"}, nil)

	assert.EqualError(t, err, "JSON export table name is required")
}

func TestWriteRejectsRowsWithWrongNumberOfValues(t *testing.T) {
	t.Parallel()

	err := Write(
		filepath.Join(t.TempDir(), "invalid.json"),
		"Album",
		[]string{"id", "title"},
		[][]any{{1}},
	)

	assert.EqualError(t, err, "JSON row 1 has 1 values; want 2")
}

func TestWriteReturnsCreateError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing", "albums.json")
	err := Write(path, "Album", []string{"id"}, nil)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "create JSON file"))
}
