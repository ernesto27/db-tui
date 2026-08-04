# ADR 0005: Provide table index inspection

## Status

Accepted on 2026-08-04.

## Context

db-tui needs a concise way to inspect the indexes of the selected table. The screen must show only the index name, indexed column, table, and access method.

Index metadata belongs to the shared `db.Database` contract. PostgreSQL catalog indexes may be standalone indexes or indexes owned by primary-key and unique constraints; both kinds need to appear. MySQL exposes comparable metadata through `information_schema.statistics`, while SQLite exposes it through `pragma_index_list` and `pragma_index_info`.

## Decision

`db.Database` defines `ListIndexes(context.Context, Table) ([]IndexColumns, error)`. PostgreSQL, MySQL, and SQLite implement it using their native metadata catalogs.

`Ctrl+G` continues to open the table actions menu. When a table is selected, the menu offers an index-inspection action for every currently supported engine. The action opens a new centered, read-only modal implemented in a dedicated file, following the existing column-inspection modal pattern.

The modal renders a flat, scrollable grid with these columns, in this order:

- `Index Name`
- `Column`
- `Table`
- `Access Method`

It includes all indexes reported by the selected engine, including indexes created for primary-key and unique constraints. A multi-column index produces one row per indexed column. Rows are ordered by index name and indexed-column position. The `Access Method` value is native engine metadata: PostgreSQL reports its access method, MySQL reports `INDEX_TYPE`, and SQLite reports `BTREE` for its ordinary indexes.

## Consequences

- PostgreSQL, MySQL, and SQLite users can inspect both explicit and constraint-owned indexes without leaving the TUI.
- The shared database interface provides one consistent metadata contract while each adapter retains its engine-specific query.
- The modal remains a simple, read-only grid despite the different metadata sources.

## Alternatives considered

### PostgreSQL-only release

Rejected because MySQL and SQLite now provide concrete index metadata queries through the shared interface.

### Exclude constraint-owned indexes

Rejected because primary-key and unique indexes are part of the table's useful physical index metadata.

### Display full index DDL

Rejected for the first release because the requested simple grid is easier to scan and matches the existing column-inspection experience.
