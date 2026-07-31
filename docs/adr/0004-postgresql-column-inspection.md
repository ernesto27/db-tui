# ADR 0004: Provide cross-engine column inspection

## Status

Accepted on 2026-07-30.

## Context

db-tui needs an IDE-style view of a selected table's columns. It must show the column name, ordinal position, data type, identity status, collation, nullability, default expression, and comment.

The shared `db.Database` interface is implemented by PostgreSQL, MySQL, and SQLite. Each engine exposes this metadata differently: PostgreSQL uses catalogs, MySQL uses `information_schema.columns`, and SQLite uses `pragma_table_xinfo`.

## Decision

`db.Database` will define `ListColumns(context.Context, Table) ([]Column, error)`. PostgreSQL, MySQL, and SQLite will implement it. `Ctrl+G` will continue to open the existing table actions menu; selecting `Inspect columns` opens the inspection screen for the selected table.

The inspection screen will be a centered, read-only overlay, following the existing DDL viewer rather than replacing the application body. `Esc` will close it and return to the unchanged underlying view.

The screen will use a tabular layout with these columns, in this order:

- `Column Name`
- `#`
- `Data type`
- `Identity`
- `Collation`
- `Not Null`
- `Default`
- `Comment`

The adapters preserve the native type text supplied by their engine and return columns in ordinal-position order. Fields unavailable in an engine, such as SQLite column comments and collation, are blank.

The overlay will render `Not Null` as `[v]`; absent identity, default, and comment values as empty cells; and a collatable column using PostgreSQL's default collation as `default`. When the terminal is too narrow for every field, left and right navigation will horizontally scroll the grid. Up/down and `j`/`k` will vertically scroll long column lists while the grid header remains visible.

## Consequences

- Each supported engine exposes a consistent column-inspection interface.
- The UI has a single rendering path despite differences in engine metadata catalogs.
- Engine-specific semantics remain visible: for example, MySQL maps `AUTO_INCREMENT` to the Identity field, while SQLite leaves unavailable fields blank.

## Alternatives considered

### Keep inspection as an optional capability

Rejected because every supported adapter now has a concrete metadata query and the UI should offer the action consistently.

### Reuse the structural DDL screen

Rejected because DDL is an executable script and is less scannable than a column-oriented grid.
