package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/jackc/pgx/v5"
)

const tableDDLSQL = `
	SELECT c.oid, c.relname, c.relkind, c.relispartition,
		EXISTS (
			SELECT 1
			FROM pg_catalog.pg_inherits inheritance
			WHERE inheritance.inhrelid = c.oid OR inheritance.inhparent = c.oid
		)
	FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace namespace ON namespace.oid = c.relnamespace
	WHERE namespace.nspname = 'public' AND c.relname = $1`

const tableDDLColumnsSQL = `
	SELECT a.attname,
		pg_catalog.format_type(a.atttypid, a.atttypmod),
		a.attnotnull,
		COALESCE(pg_catalog.pg_get_expr(default_value.adbin, default_value.adrelid), ''),
		COALESCE(NULLIF(a.attidentity, '')::text, ''),
		COALESCE(NULLIF(a.attgenerated, '')::text, ''),
		COALESCE(
			CASE WHEN a.attcollation <> typ.typcollation THEN pg_catalog.quote_ident(coll.collname) END,
			''
		)
	FROM pg_catalog.pg_attribute a
	JOIN pg_catalog.pg_type typ ON typ.oid = a.atttypid
	LEFT JOIN pg_catalog.pg_attrdef default_value
		ON default_value.adrelid = a.attrelid AND default_value.adnum = a.attnum
	LEFT JOIN pg_catalog.pg_collation coll ON coll.oid = a.attcollation
	WHERE a.attrelid = $1 AND a.attnum > 0 AND NOT a.attisdropped
	ORDER BY a.attnum`

const tableDDLConstraintsSQL = `
	SELECT conname, pg_catalog.pg_get_constraintdef(oid, true)
	FROM pg_catalog.pg_constraint
	WHERE conrelid = $1
	ORDER BY conname`

const tableDDLIndexesSQL = `
	SELECT pg_catalog.pg_get_indexdef(index_definition.indexrelid)
	FROM pg_catalog.pg_index index_definition
	WHERE index_definition.indrelid = $1
		AND NOT EXISTS (
			SELECT 1
			FROM pg_catalog.pg_constraint constraint_definition
			WHERE constraint_definition.conindid = index_definition.indexrelid
		)
	ORDER BY index_definition.indexrelid`

type tableDDLMetadata struct {
	tableName   string
	unsupported bool
	columns     []ddlColumn
	constraints []ddlConstraint
	indexes     []string
}

type ddlColumn struct {
	Name      string
	Type      string
	Collation string
	Default   string
	Identity  string
	Generated string
	NotNull   bool
}

type ddlConstraint struct {
	Name       string
	Definition string
}

// TableDDL returns a compact CREATE TABLE statement and standalone indexes.
func (p *postgresql) TableDDL(ctx context.Context, table db.Table) (string, error) {
	if table.Name == "" {
		return "", errors.New("table name is required")
	}

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", fmt.Errorf("begin PostgreSQL DDL transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	p.logger.Log("SET LOCAL search_path = pg_catalog")
	if _, err := tx.Exec(ctx, "SET LOCAL search_path = pg_catalog"); err != nil {
		return "", fmt.Errorf("set PostgreSQL DDL search path: %w", err)
	}

	metadata, err := p.loadTableDDL(ctx, tx, table.Name)
	if err != nil {
		return "", err
	}
	sql, err := buildTableDDL(metadata)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit PostgreSQL DDL transaction: %w", err)
	}
	return sql, nil
}

func (p *postgresql) loadTableDDL(ctx context.Context, tx pgx.Tx, tableName string) (tableDDLMetadata, error) {
	var metadata tableDDLMetadata
	var oid uint32
	var relationKind string
	var partitioned, inherited bool
	p.logger.Log(tableDDLSQL)
	if err := tx.QueryRow(ctx, tableDDLSQL, tableName).Scan(&oid, &metadata.tableName, &relationKind, &partitioned, &inherited); err != nil {
		return tableDDLMetadata{}, fmt.Errorf("find PostgreSQL table DDL: %w", err)
	}
	if relationKind != "r" || partitioned || inherited {
		return tableDDLMetadata{unsupported: true}, nil
	}

	columns, err := p.loadDDLColumns(ctx, tx, oid)
	if err != nil {
		return tableDDLMetadata{}, err
	}
	constraints, err := p.loadDDLConstraints(ctx, tx, oid)
	if err != nil {
		return tableDDLMetadata{}, err
	}
	indexes, err := p.loadDDLIndexes(ctx, tx, oid)
	if err != nil {
		return tableDDLMetadata{}, err
	}
	metadata.columns = columns
	metadata.constraints = constraints
	metadata.indexes = indexes
	return metadata, nil
}

func (p *postgresql) loadDDLColumns(ctx context.Context, tx pgx.Tx, oid uint32) ([]ddlColumn, error) {
	p.logger.Log(tableDDLColumnsSQL)
	rows, err := tx.Query(ctx, tableDDLColumnsSQL, oid)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL DDL columns: %w", err)
	}
	defer rows.Close()

	columns := make([]ddlColumn, 0)
	for rows.Next() {
		var column ddlColumn
		if err := rows.Scan(&column.Name, &column.Type, &column.NotNull, &column.Default, &column.Identity, &column.Generated, &column.Collation); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL DDL column: %w", err)
		}
		column.Type = compactPostgresType(column.Type)
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL DDL columns: %w", err)
	}
	return columns, nil
}

func (p *postgresql) loadDDLConstraints(ctx context.Context, tx pgx.Tx, oid uint32) ([]ddlConstraint, error) {
	p.logger.Log(tableDDLConstraintsSQL)
	rows, err := tx.Query(ctx, tableDDLConstraintsSQL, oid)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL DDL constraints: %w", err)
	}
	defer rows.Close()

	constraints := make([]ddlConstraint, 0)
	for rows.Next() {
		var constraint ddlConstraint
		if err := rows.Scan(&constraint.Name, &constraint.Definition); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL DDL constraint: %w", err)
		}
		constraints = append(constraints, constraint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL DDL constraints: %w", err)
	}
	return constraints, nil
}

func (p *postgresql) loadDDLIndexes(ctx context.Context, tx pgx.Tx, oid uint32) ([]string, error) {
	p.logger.Log(tableDDLIndexesSQL)
	rows, err := tx.Query(ctx, tableDDLIndexesSQL, oid)
	if err != nil {
		return nil, fmt.Errorf("query PostgreSQL DDL indexes: %w", err)
	}
	defer rows.Close()

	indexes := make([]string, 0)
	for rows.Next() {
		var statement string
		if err := rows.Scan(&statement); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL DDL index: %w", err)
		}
		indexes = append(indexes, statement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL DDL indexes: %w", err)
	}
	return indexes, nil
}

func buildTableDDL(metadata tableDDLMetadata) (string, error) {
	if metadata.unsupported || metadata.tableName == "" {
		return "", errors.New("unsupported PostgreSQL table structure")
	}
	if len(metadata.columns) == 0 {
		return "", errors.New("PostgreSQL table has no columns")
	}

	lines := make([]string, 0, len(metadata.columns)+len(metadata.constraints))
	for _, column := range metadata.columns {
		line := pgx.Identifier{column.Name}.Sanitize() + " " + column.Type
		if column.Collation != "" {
			line += " COLLATE " + column.Collation
		}
		switch {
		case column.Generated != "":
			storage := "STORED"
			if column.Generated == "v" {
				storage = "VIRTUAL"
			}
			line += " GENERATED ALWAYS AS (" + column.Default + ") " + storage
		case column.Identity != "":
			mode := "BY DEFAULT"
			if column.Identity == "a" {
				mode = "ALWAYS"
			}
			line += " GENERATED " + mode + " AS IDENTITY"
		case column.Default != "":
			line += " DEFAULT " + column.Default
		}
		if column.NotNull {
			line += " NOT NULL"
		}
		lines = append(lines, "    "+line)
	}
	for _, constraint := range metadata.constraints {
		lines = append(lines, "    CONSTRAINT "+pgx.Identifier{constraint.Name}.Sanitize()+" "+constraint.Definition)
	}

	statements := []string{"CREATE TABLE " + publicTableIdentifier(metadata.tableName) + " (\n" + strings.Join(lines, ",\n") + "\n);"}
	for _, index := range metadata.indexes {
		statements = append(statements, ensurePostgresSemicolon(index))
	}
	return strings.Join(statements, "\n\n"), nil
}

func compactPostgresType(dataType string) string {
	switch {
	case dataType == "smallint":
		return "int2"
	case dataType == "integer":
		return "int4"
	case dataType == "bigint":
		return "int8"
	case strings.HasPrefix(dataType, "character varying"):
		return "varchar" + strings.TrimPrefix(dataType, "character varying")
	default:
		return dataType
	}
}

func ensurePostgresSemicolon(statement string) string {
	statement = strings.TrimRight(statement, " \t\r\n")
	if strings.HasSuffix(statement, ";") {
		return statement
	}
	return statement + ";"
}

func publicTableIdentifier(name string) string {
	return "public." + pgx.Identifier{name}.Sanitize()
}
