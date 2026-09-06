# Spec: SQL Server TUI Support

## Objective

Make the existing SQL Server adapter reachable from the TUI. `internal/db/sqlserver`
already implements `db.Database` in full, but nothing outside its own package
imports it: it is absent from the connection factory, from the engine whitelist,
and from the engine picker. Selecting SQL Server in the application is currently
impossible, and a hand-edited `config.json` with `"engine": "sqlserver"` fails in
`normalizedEngine` with `unsupported database engine "sqlserver"`.

This is a wiring change, not an adapter change. SQL Server becomes the fifth
selectable engine, with the same feature surface the PostgreSQL adapter offers.

## Tech Stack

- Go 1.26
- Bubble Tea v2
- `github.com/microsoft/go-mssqldb` v1.10.0 (already a direct dependency)
- Testify

No dependency change is expected.

## Commands

```sh
docker compose up -d --wait sqlserver
docker compose up sqlserver-init
go run ./cmd/db-tui
scripts/validate.sh
```

`sqlserver-init` is a separate one-shot service that creates the `db_tui`
database. Starting `sqlserver` alone is not sufficient for the adapter tests.

## Project Structure

- `internal/db/db.go` owns the engine constants.
- `internal/db/sqlserver/` owns the adapter; only defect fixes are in scope.
- `cmd/db-tui/main.go` is the only place an adapter package is imported.
- `internal/app/` owns engine capability decisions, the connection modal, and
  DSN assembly. It must not import an adapter.
- `docs/specs/` contains this specification.

## Decisions

Each decision below was settled explicitly. The rationale is recorded because
several of them reverse a plausible default.

### Scope

1. **Parity target is PostgreSQL**, the richest existing adapter, not the Oracle
   or SQLite subset.
2. **Objects shown are tables, views, and functions.** Stored procedures are not
   introduced as a new object kind, so no `db.SchemaObjectType` member is added
   and the other four adapters are untouched.
3. **Materialized views are not supported.** SQL Server has indexed views, not
   materialized views, so `supportsMaterializedViews` continues to exclude it.

### Connection

4. **The connection modal keeps its existing five fields.** No encryption or
   named-instance field is added. The `todo.txt` item "Add configurable TLS/SSL
   settings" remains open and is the place that belongs.
5. **SQL authentication only.** Integrated/Windows authentication needs Kerberos
   plumbing and Azure AD needs a token provider; neither is reachable from the
   local fixture, so neither is half-supported.
6. **No `encrypt` parameter is injected into the DSN.** A probe against the
   2022 container confirmed that `go-mssqldb` v1.10.0's own defaults connect
   successfully with no encryption parameters present. Adding one would be
   unfounded.
7. **The SQL Server DSN carries the database as a query parameter, not a path
   segment.** This is the one non-obvious correctness requirement in the change.
   `ConnectionSettings.connectionDSN` builds `scheme://user:pass@host:port/name`
   for every server engine, but `go-mssqldb` parses the path segment as an
   *instance name*. A probe showed the path form connects successfully to the
   wrong database:

   ```text
   sqlserver://sa:***@localhost:1434/db_tui        -> Name()="" , master's tables
   sqlserver://sa:***@localhost:1434?database=db_tui -> Name()="db_tui", correct
   ```

   The failure is silent — no error, an empty database name in the header, and
   the wrong table list — so the query-parameter form is mandatory.
8. **Default port is 1433**, and the DSN placeholder shows the query-parameter
   form so a user writing a DSN by hand does not hit decision 7's trap.

### Navigation

9. **SQL Server uses the PostgreSQL-style schema hierarchy.** Its object path is
   server -> database -> schema -> table, identical in shape to PostgreSQL's and
   unlike MySQL's.
10. **The default schema is `dbo`**, hardcoded the way PostgreSQL hardcodes
    `public`. This is a hard blocker rather than a preference: SQL Server's
    `ListTables` filters on `TABLE_SCHEMA = @p1`, so the current default of `""`
    would render an empty navigator.
11. **The multi-schema database explorer is enabled for SQL Server.** The
    adapter already implements `ListSchemaObjectGroups`; without this the method
    is dead code and a SQL Server user is confined to `dbo`. The three existing
    `== db.EnginePostgreSQL` comparisons are replaced by a named
    `supportsSchemaObjectGroups` helper so the next engine is a one-line change.
12. **Cross-database browsing from one connection is out of scope**, matching
    every current engine.

### Adapter changes

13. **`Close()` is fixed.** It is currently an empty body, leaking the `*sql.DB`
    pool on every reconnect and on quit, while all four other adapters close
    their handle. This is the only defect fixed on its own merits.
14. **`ListMaterializedViews` returns `nil, nil`** instead of an error. It is
    unreachable given decision 3; a safe empty result is preferred over an error
    that a future caller could trip. `errNotImplemented` loses its last use and
    is removed.
15. **`dockerContainerIDForPort` gets an actionable error message.** Its current
    bare `not found` is the one place decision 16 becomes user-visible.
16. **`Dump` is kept as-is**, including its `BACKUP DATABASE` plus `docker ps` /
    `docker cp` / `docker exec` implementation. It works only when the server
    runs as a locally published container; that limitation is surfaced through
    decision 15's message rather than removed.
17. **Other adapter blemishes are left alone**: the three copy-pasted
    "PostgreSQL" strings in `ListTables` error messages, and `Export`'s missing
    `default` case for an unrecognized format. They are pre-existing and not
    what this change is about.

### Naming

18. **`db.EngineSqlServer` is renamed `db.EngineSQLServer`**, matching
    `EngineSQLite`/`EnginePostgreSQL` and the repository's consistent-initialism
    rule. The string value stays `"sqlserver"`, so no saved config is affected.
19. **The display name is `SQL Server`**, spelled out like `PostgreSQL` rather
    than abbreviated.
20. **The saved-connections list keeps rendering `connection.Engine` raw.** It
    will show `sqlserver` while the header shows `SQL Server`, but that mismatch
    already exists for all four current engines (`postgres` vs `PostgreSQL`) and
    fixing it is unrelated churn.

### Deferred

21. **CI is not changed in this pass.** `.github/workflows/go-test.yml` starts
    only `postgres mysql oracle` while running `go test ./...`, and the SQL
    Server tests hard-fail via `require.NoError` rather than skipping, so they
    have likely not passed since `a348d88`. Adding `sqlserver` and
    `sqlserver-init` to CI is agreed but deliberately follows the TUI work.
22. **No engine gains a test skip guard.** No adapter has one today; introducing
    one for SQL Server alone would make it the only engine that skips and would
    let real regressions pass silently.
23. **The hardcoded fixture password in `sqlserver_test.go` stays.** It is a
    local throwaway credential already present in `compose.yaml`, identical in
    kind to the committed `db_tui:db_tui` used by MySQL.
24. **Documentation is not updated.** `README.md:3`, `ARCHITECTURE.md:24`, and
    `AGENTS.md:18` still enumerate four engines and will be factually stale once
    this ships. This is a known, accepted gap.

## Behavior

1. The connection modal's engine selector cycles PostgreSQL, MySQL, Oracle,
   SQLite, and SQL Server.
2. Choosing SQL Server shows the full field set (host, database name, port,
   username, password, DSN) and defaults the port to `1433`.
3. Connecting with the discrete fields produces
   `sqlserver://user:password@host:port?database=name`.
4. A non-empty DSN field continues to win over the discrete fields, unchanged.
5. The header renders `SQL Server` and the connected database name.
6. On connect, tables, views, functions, and schema object groups load for the
   `dbo` schema.
7. The navigator shows tables, views, and functions; the materialized-views
   section stays hidden.
8. The objects key opens the multi-schema database explorer, as it does for
   PostgreSQL.
9. Data paging, row edit, row delete, DDL, columns, indexes, CSV/JSON export,
   and raw query execution all work through the already-implemented adapter.
10. A SQL Server connection saves to and reloads from `config.json` with
    `"engine": "sqlserver"`.
11. Reconnecting or quitting closes the SQL Server connection pool.
12. Dump against a non-container server reports that a local Docker container is
    required.

## Testing Strategy

Application-layer tests use the existing fake database and require no container.
Adapter behavior is already covered by the 15 integration tests in
`internal/db/sqlserver/sqlserver_test.go`.

Tests must cover:

- `normalizedEngine` accepting `sqlserver` and preserving existing rejections;
- `connectionDSN` producing the `?database=` form for SQL Server and leaving
  every other engine's path form unchanged;
- `connectionDSN` still honoring an explicit DSN;
- engine cycling reaching SQL Server and defaulting its port to `1433`;
- `engineDisplayName` returning `SQL Server`;
- `supportsFunctions`, `supportsMaterializedViews`, and
  `supportsSchemaObjectGroups` for all five engines;
- `functionSchema` returning `dbo` for SQL Server;
- the objects key opening the database explorer for a SQL Server session;
- a SQL Server connection round-tripping through `config.json`.

Run `scripts/validate.sh` once after implementation and tests are complete.

## Boundaries

- Always:
  - Keep `internal/app` free of adapter imports.
  - Build the SQL Server DSN with `?database=`, never a path segment.
  - Express engine capability through named helpers, not inline comparisons.
  - Preserve the behavior of the four existing engines.
- Ask first:
  - Add or upgrade dependencies.
  - Change the `db.Database` interface.
  - Add a connection-modal field.
  - Rewrite `Dump`.
- Never:
  - Add an `encrypt` parameter without evidence it is required.
  - Introduce a test skip guard for one engine only.
  - Commit credentials beyond the existing local fixture values.

## Success Criteria

- SQL Server is selectable in the connection modal and connects to the Compose
  fixture using the discrete fields.
- The header shows `SQL Server` and the correct database name.
- The navigator lists `dbo` tables, views, and functions.
- The database explorer opens for a SQL Server session.
- Browsing, editing, exporting, and querying work end to end.
- `Close()` releases the pool.
- `scripts/validate.sh` passes with the Compose services running.

## Out of Scope

- Stored procedures as a navigable object kind.
- Windows/Integrated and Azure AD authentication.
- Named-instance connections.
- TLS/certificate configuration UI.
- Rewriting or removing the Docker-coupled dump.
- CI workflow changes.
- Documentation updates.

## Open Questions

None. All decisions above were settled explicitly before implementation.
