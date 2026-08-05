# ADR 0007: Browse PostgreSQL materialized views as a separate navigator relation

## Status

Accepted on 2026-08-05.

## Context

db-tui already separates base tables and ordinary views in the navigator. PostgreSQL additionally exposes materialized views: named, stored query results that can be selected like relations but have a distinct refresh lifecycle. The Chinook seed includes materialized-view examples, and PostgreSQL discovery is available through `pg_catalog.pg_matviews`.

## Decision

For PostgreSQL connections only, show a `MATERIALIZED VIEWS` navigator section alongside `TABLES` and `VIEWS`. Load materialized-view names asynchronously with the existing relation discovery commands. The section remains visible when PostgreSQL has no materialized views, supports the same filter and independent selection/scroll position as the other sections, and loads the selected relation through the existing pageable data panel.

Materialized views are data-only in this release. They do not expose export, DDL, column inspection, index inspection, connection/table actions, or refresh controls. MySQL and SQLite do not display the section and do not invoke their placeholder `ListMaterializedViews` methods.

## Consequences

- PostgreSQL users can distinguish materialized snapshots from ordinary views and base tables while browsing their stored rows.
- Non-PostgreSQL navigation and object-discovery behavior remains unchanged.
- Refresh and metadata actions remain explicit future work, avoiding engine-specific action semantics in the first release.

## Alternatives considered

### Combine materialized and ordinary views

Rejected because their stored-data and refresh semantics differ, and a separate section gives the user the needed context before browsing.

### Treat materialized views as tables

Rejected because that would incorrectly expose table-only actions such as structural DDL and export.

### Add refresh controls now

Rejected because refresh locking, permissions, and concurrent-refresh prerequisites require a separate PostgreSQL-specific action design.
