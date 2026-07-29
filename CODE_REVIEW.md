## Review: checked-out `add-query-log` snapshot (working diff unavailable)

**Verdict:** Changes requested — query logging can block valid database connections and does not faithfully preserve the SQL that was executed.
**Checks:** gofmt ✓ · vet ✓ · build ✓ · tests compile ✓

### Findings

> **[Major] `internal/logger/logger.go:60` — whitespace normalization changes the SQL recorded in the log**
> `strings.Fields` followed by `strings.Join` does not merely make a statement single-line: it collapses whitespace inside quoted literals and comments, so the log can describe a different query from the one sent to the database.
> **Fails when:** executing `SELECT 'Ada  Lovelace'` records `SELECT 'Ada Lovelace'`; a multiline statement containing `--` can also be flattened so later SQL appears to remain inside the comment.
> **Fix:** preserve ordinary whitespace and escape only record-breaking control characters (for example, encode `\r`, `\n`, and `\t` visibly), then add regression tests for quoted whitespace, comments, and multiline SQL.

> **[Major] `internal/db/postgres/postgres.go:43` — opening the query log is a hard prerequisite for connecting to a database**
> PostgreSQL returns before creating its pool when the user config directory cannot be written; MySQL and SQLite repeat the same coupling at `internal/db/mysql/mysql.go:47` and `internal/db/sqlite/sqlite.go:52`. Query logging is ancillary, but its filesystem availability currently determines whether any database can be used.
> **Fails when:** a valid database is reachable but `$XDG_CONFIG_HOME`/the home config directory is read-only or unavailable → `Connect` returns `open query log: ...` and the TUI cannot connect. This reproduced with `HOME=/`; the SQLite suite passed once `XDG_CONFIG_HOME` was redirected to a writable temporary directory.
> **Fix:** create and own one logger at application startup, inject it into adapters, and define an explicit non-fatal fallback (such as an `io.Discard` logger plus a visible warning) when file logging cannot be initialized.

> **[Minor] `internal/logger/logger.go:47` — the new logging behavior has no automated coverage**
> The package has no `_test.go` file, despite repository policy requiring query-log coverage. The test-compilation output confirms `internal/logger [no test files]`, leaving formatting, append behavior, permissions, concurrency, and close behavior unprotected.
> **Fix:** add focused table-driven tests using `bytes.Buffer` and `t.TempDir`, including the SQL-preservation cases above and concurrent logging.

> **[Minor] `internal/db/sqlite/sqlite_test.go:215` — database tests use the real user config location**
> Connection helpers call production `Connect`, which opens `logs.txt`, without isolating `XDG_CONFIG_HOME`; the MySQL and PostgreSQL helpers do the same. Tests therefore append to a developer's real application log and fail before exercising database behavior in environments whose home directory is not writable.
> **Fix:** set `XDG_CONFIG_HOME` to a suite-owned temporary directory in each adapter package's `TestMain`, or inject a buffer/discard logger through test-only constructors.

The full `go test ./...` run was additionally limited by unavailable local PostgreSQL/MySQL services. With a writable temporary config directory, the self-contained SQLite tests passed. `go test -race` could not run because this container has CGO disabled.

Scope limitation: `.git` points to a missing linked-worktree directory (`/home/eponce/code/ai-enginering/db-tui/.git/worktrees/add-query-log`), so `git status` and `git diff HEAD` could not resolve the exact working change. The review therefore used the worktree name and checked-out query-logging implementation as the concrete scope.
