# ADR 0006: Group database views in the navigator

## Status

Accepted on 2026-08-04.

## Context

The database navigator lists tables and views as separate relation types. PostgreSQL, MySQL, and SQLite all expose ordinary query-backed views that users should be able to browse without mistaking them for base tables.

## Decision

For every connection, show a `VIEWS` navigator section alongside `TABLES`, including when the active database has no views or none match the filter. The navigator's filter searches both sections; `Left` and `Right` switch sections while the navigator is focused, and each section retains its own selection and scroll position.

View discovery is engine-specific: PostgreSQL lists ordinary views in the `public` schema, MySQL lists views in the active database, and SQLite lists non-internal entries of type `view` from `sqlite_master`. Table and view discovery run in parallel. When an engine has no tables but has views, the navigator automatically selects the first view once both discovery operations finish, regardless of their completion order.

Selecting a view loads and displays its rows through the existing data panel. The initial release includes no materialized-view listing and no view-specific actions such as DDL, column inspection, index inspection, export, or refresh.

## Consequences

- Users can discover and browse PostgreSQL, MySQL, and SQLite views while clearly seeing that they are not base tables.
- The first implementation adds only relation discovery and row browsing, keeping the existing table action surface unchanged.
- Materialized views and view-specific actions remain future work.

## Alternatives considered

### One mixed relation list with type markers

Rejected because distinct sections make the relationship type immediately visible and match the requested navigation model.

### Include materialized views now

Rejected because the current scope is ordinary views only.

### Give views the full table action menu

Rejected because the current scope is data browsing only.
