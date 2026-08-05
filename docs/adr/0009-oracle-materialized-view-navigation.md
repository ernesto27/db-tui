# ADR 0009: Browse Oracle materialized views in the navigator

## Status

Accepted on 2026-08-05.

## Context

ADR 0007 introduced a separate `MATERIALIZED VIEWS` navigator section for PostgreSQL. Oracle exposes the equivalent relation type through `USER_MVIEWS`, and the Oracle adapter already discovers it. The navigator was limiting this section to PostgreSQL, leaving discovered Oracle materialized views inaccessible in the TUI.

## Decision

Enable the existing `MATERIALIZED VIEWS` navigator section for Oracle connections as well as PostgreSQL connections. Oracle materialized views use the existing asynchronous discovery, filtering, selection, scrolling, and bounded row-browsing behavior.

Like PostgreSQL materialized views, Oracle materialized views are data-only in this release. They do not expose DDL, column or index inspection, export, refresh, or table actions. MySQL and SQLite continue not to display the section.

## Consequences

- Oracle users can distinguish materialized views from tables and ordinary views, then browse their rows.
- The navigator capability is shared by PostgreSQL and Oracle while preserving existing behavior for other engines.
- Oracle-specific refresh and metadata actions remain future work.

## Alternatives considered

### List materialized views as tables

Rejected because that would expose table-only actions that do not apply to the data-only materialized-view workflow.

### Keep the section PostgreSQL-only

Rejected because the Oracle adapter can already discover and page through materialized views.
