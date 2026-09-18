// Package csvexport writes tabular data to CSV files.
package csvexport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"time"
)

// Write creates or replaces path with a CSV document containing columns and rows.
func Write(path string, columns []string, rows [][]any) error {
	if path == "" {
		return errors.New("CSV export path is required")
	}

	data, err := Marshal(columns, rows)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("create CSV file: %w", err)
	}

	return nil
}

// Marshal converts columns and rows to a CSV document with a leading header row.
//
// The returned document ends with a record separator.
func Marshal(columns []string, rows [][]any) ([]byte, error) {
	var buffer bytes.Buffer

	writer := csv.NewWriter(&buffer)
	if err := writer.Write(columns); err != nil {
		return nil, fmt.Errorf("write CSV header: %w", err)
	}

	for rowIndex, row := range rows {
		if len(row) != len(columns) {
			return nil, fmt.Errorf("CSV row %d has %d values; want %d", rowIndex+1, len(row), len(columns))
		}
		if err := writer.Write(formatRow(row)); err != nil {
			return nil, fmt.Errorf("write CSV row %d: %w", rowIndex+1, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush CSV document: %w", err)
	}

	return buffer.Bytes(), nil
}

func formatRow(row []any) []string {
	values := make([]string, len(row))
	for index, value := range row {
		values[index] = formatValue(value)
	}
	return values
}

func formatValue(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(value)
	case time.Time:
		return value.Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(value)
	}
}
