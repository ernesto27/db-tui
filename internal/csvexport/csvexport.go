// Package csvexport writes tabular data to CSV files.
package csvexport

import (
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

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(columns); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	for rowIndex, row := range rows {
		if len(row) != len(columns) {
			return fmt.Errorf("CSV row %d has %d values; want %d", rowIndex+1, len(row), len(columns))
		}
		if err := writer.Write(formatRow(row)); err != nil {
			return fmt.Errorf("write CSV row %d: %w", rowIndex+1, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close CSV file: %w", err)
	}

	return nil
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
