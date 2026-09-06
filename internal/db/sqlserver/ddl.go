package sqlserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/db"
)

const tableDDLTableSQL = `
	SELECT
		table_info.object_id,
		schema_info.name,
		table_info.name
	FROM sys.tables AS table_info
	JOIN sys.schemas AS schema_info
		ON schema_info.schema_id = table_info.schema_id
	WHERE schema_info.name = @p1
		AND table_info.name = @p2
		AND table_info.is_ms_shipped = 0`

const tableDDLColumnsSQL = `
	SELECT
		column_info.name,
		type_schema.name,
		type_info.name,
		column_info.max_length,
		column_info.precision,
		column_info.scale,
		column_info.is_nullable,
		default_info.definition,
		CASE WHEN identity_info.object_id IS NULL THEN 0 ELSE 1 END,
		CONVERT(nvarchar(40), identity_info.seed_value),
		CONVERT(nvarchar(40), identity_info.increment_value)
	FROM sys.columns AS column_info
	JOIN sys.types AS type_info
		ON type_info.user_type_id = column_info.user_type_id
	JOIN sys.schemas AS type_schema
		ON type_schema.schema_id = type_info.schema_id
	LEFT JOIN sys.default_constraints AS default_info
		ON default_info.object_id = column_info.default_object_id
	LEFT JOIN sys.identity_columns AS identity_info
		ON identity_info.object_id = column_info.object_id
			AND identity_info.column_id = column_info.column_id
	WHERE column_info.object_id = @p1
	ORDER BY column_info.column_id`

const tableDDLPrimaryKeySQL = `
	SELECT
		constraint_info.name,
		column_info.name
	FROM sys.key_constraints AS constraint_info
	JOIN sys.index_columns AS index_column_info
		ON index_column_info.object_id = constraint_info.parent_object_id
			AND index_column_info.index_id = constraint_info.unique_index_id
	JOIN sys.columns AS column_info
		ON column_info.object_id = index_column_info.object_id
			AND column_info.column_id = index_column_info.column_id
	WHERE constraint_info.parent_object_id = @p1
		AND constraint_info.type = 'PK'
		AND index_column_info.key_ordinal > 0
	ORDER BY index_column_info.key_ordinal`

type tableDDLMetadata struct {
	schema     string
	tableName  string
	columns    []ddlColumn
	primaryKey *ddlPrimaryKey
}

type ddlColumn struct {
	name         string
	typeSchema   string
	typeName     string
	maxLength    int16
	precision    uint8
	scale        int8
	isNullable   bool
	defaultValue sql.NullString
	isIdentity   bool
	identitySeed sql.NullString
	identityStep sql.NullString
}

type ddlPrimaryKey struct {
	name    string
	columns []string
}

func (s *sqlserverDatabase) loadTableDDL(ctx context.Context, table db.Table) (tableDDLMetadata, error) {
	schema := table.Schema
	if schema == "" {
		schema = "dbo"
	}

	metadata := tableDDLMetadata{schema: schema}
	var objectID int
	if err := s.database.QueryRowContext(ctx, tableDDLTableSQL, schema, table.Name).Scan(&objectID, &metadata.schema, &metadata.tableName); err != nil {
		return tableDDLMetadata{}, fmt.Errorf("find SQL Server table DDL: %w", err)
	}

	rows, err := s.database.QueryContext(ctx, tableDDLColumnsSQL, objectID)
	if err != nil {
		return tableDDLMetadata{}, fmt.Errorf("query SQL Server DDL columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var column ddlColumn
		if err := rows.Scan(
			&column.name,
			&column.typeSchema,
			&column.typeName,
			&column.maxLength,
			&column.precision,
			&column.scale,
			&column.isNullable,
			&column.defaultValue,
			&column.isIdentity,
			&column.identitySeed,
			&column.identityStep,
		); err != nil {
			return tableDDLMetadata{}, fmt.Errorf("scan SQL Server DDL column: %w", err)
		}
		metadata.columns = append(metadata.columns, column)
	}
	if err := rows.Err(); err != nil {
		return tableDDLMetadata{}, fmt.Errorf("iterate SQL Server DDL columns: %w", err)
	}

	primaryKey, err := s.loadDDLPrimaryKey(ctx, objectID)
	if err != nil {
		return tableDDLMetadata{}, err
	}
	metadata.primaryKey = primaryKey

	return metadata, nil
}

func (s *sqlserverDatabase) loadDDLPrimaryKey(ctx context.Context, objectID int) (*ddlPrimaryKey, error) {
	rows, err := s.database.QueryContext(ctx, tableDDLPrimaryKeySQL, objectID)
	if err != nil {
		return nil, fmt.Errorf("query SQL Server table primary key: %w", err)
	}
	defer rows.Close()

	var primaryKey *ddlPrimaryKey
	for rows.Next() {
		var name, column string
		if err := rows.Scan(&name, &column); err != nil {
			return nil, fmt.Errorf("scan SQL Server table primary key: %w", err)
		}
		if primaryKey == nil {
			primaryKey = &ddlPrimaryKey{name: name}
		}
		primaryKey.columns = append(primaryKey.columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate SQL Server table primary key: %w", err)
	}

	return primaryKey, nil
}

func buildSQLServerTableDDL(metadata tableDDLMetadata) (string, error) {
	if metadata.tableName == "" {
		return "", errors.New("SQL Server table was not found")
	}
	if len(metadata.columns) == 0 {
		return "", errors.New("SQL Server table has no columns")
	}

	lines := make([]string, 0, len(metadata.columns)+1)
	for _, column := range metadata.columns {
		line := quoteIdentifier(column.name) + " " + sqlServerColumnType(column)
		if column.isIdentity {
			line += " IDENTITY(" + column.identitySeed.String + "," + column.identityStep.String + ")"
		}
		if column.defaultValue.Valid {
			line += " DEFAULT " + column.defaultValue.String
		}
		if !column.isNullable {
			line += " NOT NULL"
		}
		lines = append(lines, "    "+line)
	}
	if metadata.primaryKey != nil {
		columns := make([]string, 0, len(metadata.primaryKey.columns))
		for _, column := range metadata.primaryKey.columns {
			columns = append(columns, quoteIdentifier(column))
		}
		lines = append(lines, "    CONSTRAINT "+quoteIdentifier(metadata.primaryKey.name)+" PRIMARY KEY ("+strings.Join(columns, ", ")+")")
	}

	return "CREATE TABLE " + quoteIdentifier(metadata.schema) + "." + quoteIdentifier(metadata.tableName) + " (\n" + strings.Join(lines, ",\n") + "\n);", nil
}

func sqlServerColumnType(column ddlColumn) string {
	dataType := column.typeName
	if column.typeSchema != "sys" {
		dataType = quoteIdentifier(column.typeSchema) + "." + quoteIdentifier(column.typeName)
	}

	switch column.typeName {
	case "binary", "char", "varbinary", "varchar":
		return dataType + "(" + sqlServerLength(column.maxLength) + ")"
	case "nchar", "nvarchar":
		return dataType + "(" + sqlServerUnicodeLength(column.maxLength) + ")"
	case "decimal", "numeric":
		return dataType + "(" + strconv.Itoa(int(column.precision)) + "," + strconv.Itoa(int(column.scale)) + ")"
	case "datetime2", "datetimeoffset", "time":
		return dataType + "(" + strconv.Itoa(int(column.scale)) + ")"
	default:
		return dataType
	}
}

func sqlServerLength(length int16) string {
	if length == -1 {
		return "max"
	}
	return strconv.Itoa(int(length))
}

func sqlServerUnicodeLength(length int16) string {
	if length == -1 {
		return "max"
	}
	return strconv.Itoa(int(length / 2))
}
