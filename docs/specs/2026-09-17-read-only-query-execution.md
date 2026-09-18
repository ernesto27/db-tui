# Spec: Read-only query execution

## Objective

Give the raw-query feature a database-layer path for executing read-only SQL
without changing ordinary raw-query execution. The future TUI read-only mode
will use this capability to prevent persistent schema and data changes while
still supporting common inspection queries.

For PostgreSQL, MySQL, Oracle, and SQLite, the adapter must enforce the
restriction at the database connection or transaction. SQL Server has no
portable per-query transaction equivalent in the current adapter, so its
future TUI path will use the same conservative client-side validation before
calling ordinary execution. The validation is a safety guard, not a substitute
for least-privilege database credentials.

## Tech Stack

- Go 1.26
- `internal/db` driver-neutral contracts and validation
- `internal/db/{postgres,mysql,oracle,sqlite,sqlserver}` adapters
- pgx v5, go-sql-driver/mysql, go-ora, modernc SQLite, and go-mssqldb
- Go `testing` and `testify`

No new dependency or database schema is required.

## Commands

```sh
go test ./...
go build ./...
scripts/validate.sh
```

Run automated verification once after implementation and formatting are
complete. The user performs manual terminal testing.

## Project Structure

```text
internal/db/
  db.go                         Query execution mode and validation
  db_test.go                    Neutral validation tests
  postgres/postgres.go          PostgreSQL read-only transaction execution
  mysql/mysql.go                MySQL read-only transaction execution
  oracle/oracle.go              Oracle read-only transaction execution
  sqlite/sqlite.go              Dedicated SQLite query-only execution
  sqlserver/sqlserver.go        Read-only-mode validation and ordinary execution
  */*_test.go                   Adapter-level execution and rejection tests
internal/app/                   Deferred consumer; no change in this module
docs/specs/
  2026-09-17-read-only-query-execution.md
```

The neutral contract belongs in `internal/db`; adapters implement it and must
not import `internal/app`. The future application feature passes the execution
mode through the existing `Database.Execute` boundary.

## Code Style

Extend the existing database-execution contract with a named mode:

```go
type QueryExecutionMode uint8

const (
    QueryExecutionDefault QueryExecutionMode = iota
    QueryExecutionReadOnly
)

Execute(context.Context, string, QueryExecutionMode) (QueryResult, error)
```

`ValidateReadOnlyQuery` must be driver-neutral and accept the current engine
identifier plus the original SQL text. It must return an actionable error
without rewriting SQL. Every adapter method accepts `context.Context`, wraps
propagated errors with `%w`, logs the submitted query under the existing safe
logging rules, and bounds interactive results to `db.MaxPageSize`.

## Read-only SQL policy

The validator is lexical and deliberately fail-closed. It ignores comments,
string literals, and quoted identifiers while inspecting SQL tokens. It
accepts exactly one statement with at most one trailing semicolon and rejects
ambiguous or unsupported syntax.

Accepted forms are:

- every engine: `SELECT`, a `WITH` expression whose final operation is
  `SELECT`, and `VALUES`;
- PostgreSQL: additionally `TABLE`, `SHOW`, and `EXPLAIN` excluding
  `EXPLAIN ANALYZE`;
- MySQL: additionally `SHOW`, `DESCRIBE`, and `EXPLAIN`;
- SQLite: additionally `EXPLAIN` and `EXPLAIN QUERY PLAN`; generic `PRAGMA`
  remains rejected because pragmas can modify state;
- Oracle: only `SELECT` and read-only `WITH … SELECT`; `EXPLAIN PLAN` remains
  rejected because it can write a plan table; and
- SQL Server: `SELECT`, read-only `WITH … SELECT`, and `VALUES`.

The validator rejects mutating, transactional, locking, and session-control
tokens—including `INSERT`, `UPDATE`, `DELETE`, `MERGE`, `CREATE`, `ALTER`,
`DROP`, `TRUNCATE`, `GRANT`, `REVOKE`, `CALL`, `EXEC`, `COMMIT`, `ROLLBACK`,
`BEGIN`, `SET`, `SELECT INTO`, and locking reads—at any nesting level. This
includes data-modifying CTEs and multi-statement batches. A keyword-looking
string, comment, or quoted identifier alone is not a rejection.

## Engine behavior

- PostgreSQL: read-only mode validates then opens a pgx transaction with
  `AccessMode: pgx.ReadOnly`; it runs the original SQL there and rolls back on
  completion.
- MySQL: it validates then starts a transaction with
  `sql.TxOptions{ReadOnly: true}`; it executes the original SQL and rolls back
  on completion.
- Oracle: it validates then begins a transaction and issues
  `SET TRANSACTION READ ONLY` before running the original SQL; it rolls back
  on completion.
- SQLite: it validates then acquires one dedicated connection, enables
  `PRAGMA query_only = ON`, executes the original SQL, restores the setting,
  and releases the connection. Cleanup must occur on success, error, and
  context cancellation so the setting cannot leak through the pool.
- SQL Server: read-only mode validates before ordinary execution. It must not
  claim database-enforced protection.

All paths preserve the current result shape, row bound, cancellation behavior,
and stale-result handling at the application boundary. A rejected statement
does not reach an adapter and must not trigger SQL-script saving.

## Testing Strategy

Use table-driven tests with `testify/assert` or `testify/require`.

- Validate case-insensitive accepted forms, leading whitespace, comments,
  quoted identifiers, string literals, and one trailing semicolon.
- Reject empty SQL, multiple statements, unsupported first tokens, write and
  transaction statements, locking reads, `SELECT INTO`, data-modifying CTEs,
  and write keywords outside comments/literals/quoted identifiers.
- Test every engine-specific accepted and rejected form in the policy above.
- At the lowest practical adapter layer, verify PostgreSQL, MySQL, Oracle, and
  SQLite start their documented read-only mechanism before executing the
  original SQL and always clean it up.
- Verify failed validation and adapter rejection return errors with no partial
  result and no mutable execution command.
- Update fakes, callers, and compile-time assertions for the expanded method
  signature.

No test may require remote credentials or a remote database. Integration tests
may use the repository's local reproducible fixtures where a driver behavior
cannot be verified by a focused fake.

## Boundaries

- Always: validate without rewriting SQL; preserve the exact submitted SQL;
  use engine enforcement where available; clean up transactions and dedicated
  connections; add focused automated coverage.
- Ask first: add a SQL parser dependency; broaden the accepted SQL allowlist;
  add another execution mode or change its semantics;
  change connection credentials, DSNs, or SQL Server connection routing.
- Never: claim the SQL Server lexical gate is authorization; execute a
  rejected statement; change the ordinary `Database.Execute` behavior; persist
  a connection-wide SQLite `query_only` setting; commit credentials, logs, or
  generated artifacts.

## Success Criteria

1. `internal/db` exposes a documented execution mode and a driver-neutral
   validator for the specified engine-aware allowlist.
2. PostgreSQL, MySQL, Oracle, and SQLite execute accepted SQL through a scoped
   engine-enforced read-only path and clean up on every outcome.
3. SQL Server has no falsely advertised engine-enforced mode; it validates
   before ordinary execution when read-only mode is selected.
4. Rejected statements never reach an adapter, preserving the original SQL for
   display or later editing.
5. Existing arbitrary raw-query execution remains unchanged until the deferred
   TUI specification consumes this module.
6. Focused tests plus `scripts/validate.sh` pass.

## Open Questions

None.
