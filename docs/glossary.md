# Glossary

## JSON table export

A complete-data export of the selected table as one JSON document. The selected table name is the sole top-level key and its value is an array of objects keyed by database column name. SQL `NULL` is encoded as JSON `null`.

## Export format picker

The `Ctrl+E` table-data overlay used to select CSV or JSON before confirmation. CSV is selected initially; `Up`/`Down` or `j`/`k` change the selection, `Enter` advances, and `Esc` cancels.

## DDL modal

A centered, read-only db-tui overlay that displays the selected table's fresh structural SQL script. It is opened with `Ctrl+G` and does not change the underlying panel state.

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
