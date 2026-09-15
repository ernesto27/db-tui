# ADR 0019: Browse PostgreSQL extensions as read-only database metadata

## Status

Accepted on 2026-09-12.

## Context

PostgreSQL records installed extensions in its catalog. db-tui already exposes
this data through the optional `db.Extension` capability, whose values are an
extension name, installed schema, and current installed version. The
database-explorer modal has no route to present that capability.

Extensions are database-level catalog metadata, not relations. Treating them
as a synthetic table would incorrectly expose relation-only behavior such as
DDL, export, row editing, deletion, and pagination.

## Decision

For a PostgreSQL connection implementing `db.Extension`, Database Explorer
will add a top-level **Extensions** choice. Selecting it will asynchronously
load the current extension metadata and display a non-pageable data grid with
`Name`, `Schema`, and `Version` columns.

The extension display will own dedicated application state rather than an
active relation. It will be read-only: relation-oriented DDL, export, edit,
and delete actions will be unavailable. `r` will reload the list. An empty
result will show “No extensions found”; a failed load will show the sanitized
catalog error.

The feature will use the existing optional capability instead of adding
extension methods to `db.Database`, so non-PostgreSQL adapters do not acquire
a PostgreSQL catalog contract.

## Consequences

- PostgreSQL users will be able to inspect installed extensions directly from
  Database Explorer.
- Other engines will not show an irrelevant option.
- Extension results will participate in session and request identity checks,
  so stale asynchronous results cannot replace current panel content.
- The reusable grid will be used without misrepresenting extension rows as
  editable relations.

## Alternatives considered

### Add extensions to the navigator

Rejected because extensions are database-level metadata, not a schema-scoped
object category or a relation list.

### Model extensions as a synthetic table

Rejected because it would enable invalid relation operations and couple
PostgreSQL catalog data to table semantics.

### Add `ListExtensions` to `db.Database`

Rejected because the existing optional `db.Extension` capability already
expresses the PostgreSQL-specific behavior without requiring every adapter to
implement an irrelevant method.
