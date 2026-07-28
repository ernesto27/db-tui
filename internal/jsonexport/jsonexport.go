package jsonexport

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"time"
)

// Write writes table rows as a JSON document keyed by tableName.
func Write(path, tableName string, columns []string, rows [][]any) error {
	if path == "" {
		return errors.New("JSON export path is required")
	}
	if tableName == "" {
		return errors.New("JSON export table name is required")
	}

	records := make([]map[string]any, 0, len(rows))

	for rowIndex, row := range rows {
		if len(row) != len(columns) {
			return fmt.Errorf("JSON row %d has %d values; want %d", rowIndex+1, len(row), len(columns))
		}

		record := make(map[string]any, len(columns))
		for columnIndex, column := range columns {
			record[column] = normalizeValue(row[columnIndex])
		}
		records = append(records, record)
	}

	data, err := json.MarshalIndent(map[string][]map[string]any{tableName: records}, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("create JSON file: %w", err)
	}

	return nil
}

func normalizeValue(value any) any {
	switch value := value.(type) {
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Sprint(value)
		}
	case float32:
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return fmt.Sprint(value)
		}
	case time.Time:
		return value.Format(time.RFC3339Nano)
	case [16]byte:
		return formatUUID(value)
	}

	return value
}

func formatUUID(value [16]byte) string {
	hexValue := fmt.Sprintf("%x", value)
	return hexValue[:8] + "-" + hexValue[8:12] + "-" + hexValue[12:16] + "-" + hexValue[16:20] + "-" + hexValue[20:]
}
