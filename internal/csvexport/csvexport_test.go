package csvexport

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCreatesCSVFromMockData(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "albums.csv")
	createdAt := time.Date(2026, time.July, 24, 12, 30, 45, 123000000, time.UTC)

	err := Write(path,
		[]string{"AlbumId", "Title", "Available", "CreatedAt", "Notes", "Raw"},
		[][]any{
			{1, "ACME, Inc.", true, createdAt, nil, []byte("binary as text")},
			{2, "A \"quoted\" title\nwith a new line", false, createdAt, "", []byte("second row")},
		},
	)
	require.NoError(t, err)

	file, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	records, err := csv.NewReader(file).ReadAll()
	require.NoError(t, err)
	assert.Equal(t, [][]string{
		{"AlbumId", "Title", "Available", "CreatedAt", "Notes", "Raw"},
		{"1", "ACME, Inc.", "true", "2026-07-24T12:30:45.123Z", "", "binary as text"},
		{"2", "A \"quoted\" title\nwith a new line", "false", "2026-07-24T12:30:45.123Z", "", "second row"},
	}, records)
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.July, 24, 12, 30, 45, 123000000, time.UTC)

	tests := []struct {
		name    string
		columns []string
		rows    [][]any
		want    string
		wantErr string
	}{
		{
			name:    "header row and rows",
			columns: []string{"AlbumId", "Title", "CreatedAt", "Notes", "Raw"},
			rows: [][]any{
				{1, "ACME, Inc.", createdAt, nil, []byte("binary as text")},
				{2, `A "quoted" title`, createdAt, "", []byte("second row")},
			},
			want: strings.Join([]string{
				`AlbumId,Title,CreatedAt,Notes,Raw`,
				`1,"ACME, Inc.",2026-07-24T12:30:45.123Z,,binary as text`,
				`2,"A ""quoted"" title",2026-07-24T12:30:45.123Z,,second row`,
				``,
			}, "\n"),
		},
		{
			name:    "header row without rows",
			columns: []string{"id", "title"},
			want:    "id,title\n",
		},
		{
			name:    "empty field for a NULL value",
			columns: []string{"id", "title"},
			rows:    [][]any{{1, nil}},
			want:    "id,title\n1,\n",
		},
		{
			name:    "quoted field containing a separator",
			columns: []string{"name"},
			rows:    [][]any{{"Doe, Jane"}},
			want:    "name\n\"Doe, Jane\"\n",
		},
		{
			name:    "quoted field containing a new line",
			columns: []string{"note"},
			rows:    [][]any{{"first\nsecond"}},
			want:    "note\n\"first\nsecond\"\n",
		},
		{
			name:    "row has fewer values than columns",
			columns: []string{"id", "title"},
			rows:    [][]any{{1}},
			wantErr: "CSV row 1 has 1 values; want 2",
		},
		{
			name:    "row has more values than columns",
			columns: []string{"id"},
			rows:    [][]any{{1, "extra"}},
			wantErr: "CSV row 1 has 2 values; want 1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			data, err := Marshal(test.columns, test.rows)

			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				assert.Nil(t, data)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, string(data))
		})
	}
}

func TestWriteRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		columns []string
		rows    [][]any
		wantErr string
	}{
		{
			name:    "empty path",
			columns: []string{"id"},
			wantErr: "CSV export path is required",
		},
		{
			name:    "row has fewer values than columns",
			path:    filepath.Join(t.TempDir(), "invalid.csv"),
			columns: []string{"id", "title"},
			rows:    [][]any{{1}},
			wantErr: "CSV row 1 has 1 values; want 2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Write(test.path, test.columns, test.rows)
			assert.EqualError(t, err, test.wantErr)
		})
	}
}

func TestWriteReturnsCreateError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing", "albums.csv")
	err := Write(path, []string{"id"}, nil)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "create CSV file"))
}
