# ADR 0001: Reconstruct PostgreSQL table DDL from catalogs

## Status

Accepted on 2026-07-24.

## Context

db-tui needs to display fresh, executable structural DDL for the table selected in its navigator. MySQL provides a canonical `SHOW CREATE TABLE` result. PostgreSQL does not provide an equivalent single SQL command for ordinary table definitions.

Using `pg_dump --schema-only --table` would provide high-fidelity output, but it introduces a local executable dependency, can be incompatible with the server version, and produces surrounding dump metadata that the feature explicitly does not need. The product requirement rejects that dependency.

## Decision

PostgreSQL will reconstruct supported table DDL from server catalogs inside a read-only transaction. It will use PostgreSQL decompilation helpers where available:

- `format_type` for data types;
- `pg_get_expr` for defaults and generated expressions;
- `pg_get_constraintdef` for constraints; and
- `pg_get_indexdef` for indexes.

The emitted script contains one `CREATE TABLE` statement with columns, defaults, identity/generated clauses, collations, and inline constraints, followed by non-constraint `CREATE INDEX` statements. It omits comments, ownership, grants, triggers, and separate sequence declarations.

The first release supports ordinary, non-inherited, non-partitioned tables in the navigator's `public` schema. Unsupported structures cause a clear error rather than a partial script. MySQL returns its verbatim `SHOW CREATE TABLE` result.

## Consequences

### Positive

- The feature has no `pg_dump` or `pg_restore` runtime dependency.
- DDL is retrieved from the active connection and is fresh on every open.
- PostgreSQL output is limited to the requested structural scope.
- The database interface stays driver-neutral.

### Negative

- PostgreSQL reconstruction is more complex than running a dump command.
- The initial support boundary excludes partitioned and inherited tables.
- The result is executable structural DDL for supported tables, not a byte-for-byte dump artifact.

## Alternatives considered

### `pg_dump` and `pg_restore`

Rejected because the user does not want a dump-tool dependency, and filtering a dump to the requested structural scope adds tool/version and archive-handling complexity.

### Full catalog reconstruction for both engines

Rejected for MySQL because `SHOW CREATE TABLE` is already the engine's canonical, executable output. Rebuilding it risks lower fidelity and unnecessary complexity.

### Best-effort PostgreSQL output for every table class

Rejected because a partial script would appear usable while silently missing partitioning or inheritance semantics. The modal must fail closed outside its support boundary.
