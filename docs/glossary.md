# Glossary

## Row deletion confirmation

A modal, destructive-action checkpoint opened with `d` for a selected row in
an active base table while the data panel has focus. It names the table without
displaying primary-key values. `Enter` confirms, `Esc` cancels or returns, and
the modal shows deletion progress, success, or failure before dismissal.

## Row-browsable relation

A base table, ordinary database view, or materialized view whose rows db-tui can display in the pageable data panel. All row-browsable relation types use the same explicit activation behavior in the navigator.

## Highlighted relation

The row-browsable relation under the navigator cursor. Keyboard navigation, mouse navigation, filtering, and section changes may change the highlighted relation without querying its rows.

## Active relation

The row-browsable relation most recently activated with `Enter` or a double-click. Activating a relation makes it the target of row-page requests and the owner of the data panel. The active relation remains unchanged while the user highlights other relations.

Activation transfers data-panel ownership immediately: the prior relation's page is cleared before the new relation's first page finishes loading. A loading state or load error therefore belongs to the newly active relation.

## Stale row result

An asynchronous row-page result whose connection session, request ID, relation identity, or page offset no longer matches the active row request. Changing only the navigator highlight does not make a result stale; activating a different relation does.

## Relation identity

The combination of a relation's type and name. The active relation retains this identity independently of the navigator highlight so row paging, panel labeling, and asynchronous-result validation continue to refer to the same row source.

## Client-observed elapsed time

The duration measured by the raw-query command from immediately before `db.Database.Execute` until it returns. It includes driver work, network transfer where applicable, and bounded result decoding; it is not a database-server-only execution metric. The raw-query UI labels this value `Execution time`.

## JSON table export

A complete-data export of the selected table as one JSON document. The selected table name is the sole top-level key and its value is an array of objects keyed by database column name. SQL `NULL` is encoded as JSON `null`.

## Export format picker

The `Ctrl+E` table-data overlay used to select CSV or JSON before confirmation. CSV is selected initially; `Up`/`Down` or `j`/`k` change the selection, `Enter` advances, and `Esc` cancels.

## DDL modal

A centered, read-only db-tui overlay that displays the selected table's fresh structural SQL script. It is opened with `Ctrl+G` and does not change the underlying panel state.

## Column inspection

A cross-engine, read-only table screen for the currently selected table. It displays each column's name, ordinal position, data type, identity status, collation, nullability, default expression, and comment. Fields that an engine does not expose, such as SQLite collation and comments, render blank.

## Index inspection

A cross-engine, read-only table screen for the currently selected table. It displays the index name, indexed column, table name, and access method. It includes standalone indexes plus indexes created for primary-key and unique constraints. A multi-column index is represented by one row for each indexed column. PostgreSQL reports its access method, MySQL reports its index type, and SQLite reports `BTREE` for ordinary indexes.

## Structural DDL

The executable SQL needed to define a supported table's columns, defaults, identity/generated clauses, collations, inline constraints, and non-constraint indexes. PostgreSQL structural DDL intentionally excludes comments, ownership, grants, triggers, and separate sequence declarations.

## Table DDL

The driver-neutral string returned by `db.Database.TableDDL` for one selected table. MySQL supplies its server-generated `SHOW CREATE TABLE` result; PostgreSQL reconstructs the supported structural script from catalogs.

## Constraint-owned index

An index PostgreSQL creates to enforce a primary-key, unique, or exclusion constraint. It appears through that constraint's DDL and must not be emitted again as a standalone index.

## Stale DDL result

An asynchronous DDL command result whose connection session, request ID, selected table, or modal state no longer matches the active request. db-tui ignores stale results.

## Supported PostgreSQL table

For the first DDL-modal release, an ordinary table in the `public` schema that is neither partitioned nor inherited. Other relation structures return an explicit unsupported-structure error.

## Database view

A named, query-backed relation. The view browser lists ordinary PostgreSQL views in the `public` schema, views in the active MySQL database, and non-internal SQLite views. Views appear separately from tables and support row browsing only.

## Materialized view

A stored query result in PostgreSQL or Oracle that persists until it is refreshed. db-tui lists materialized views in their own navigator section. They support pageable row browsing only; refresh and all table-specific actions are intentionally excluded.

## Oracle dump placeholder

The initial Oracle engine's implementation of the `db.Database` dump operation.
It returns an explicit unsupported-operation error because Oracle Data Pump
requires external tooling and database-specific privileges that db-tui does not
provision.
