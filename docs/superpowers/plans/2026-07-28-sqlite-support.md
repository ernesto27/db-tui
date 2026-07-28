# SQLite Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full SQLite support—file-path connections, table browsing, SQL, DDL, CSV/JSON exports, and `sqlite3` SQL dumps—with a permanent official SQLite test fixture used by SQLite tests.

**Architecture:** Add an `internal/db/sqlite` adapter implementing the existing `db.Database` interface. Extend the composition root and connection modal for a SQLite file path while keeping the app layer driver-neutral. Store and test against a checked-in `docker/sqlite/employee.db` fixture; use the external `sqlite3` executable only for dump generation.

**Tech Stack:** Go 1.26, Bubble Tea v2, `database/sql`, `modernc.org/sqlite`, existing CSV/JSON exporters, and the `sqlite3` CLI for `.dump`.

## Global Constraints

- SQLite connections accept local database-file paths only; do not add SQLite URI or in-memory modes.
- Use the pure-Go `modernc.org/sqlite` driver; do not require CGO or a system SQLite library to connect.
- Reuse the existing optional DSN config slot internally, labeling it **Database file** for SQLite; do not add a config migration.
- Keep table/query CSV and JSON exports in the existing in-app export pipeline.
- Generate SQLite database dumps with `sqlite3 <database-file> .dump`; report missing executable and command failures through the existing dump modal.
- Store the permanent fixture at `docker/sqlite/employee.db`; tests must not modify it or download files at test runtime.
- Preserve the existing `db.MaxPageSize` and `db.MaxQueryResultRows` bounds.
- Do not commit credentials, generated dumps, or unrelated working-tree changes.

## Files and responsibilities

- Create `internal/db/sqlite/sqlite.go`: SQLite connection, table discovery, paging, DDL, query execution, dump, export, close, and value conversion.
- Create `internal/db/sqlite/sqlite_test.go`: adapter unit/integration tests against the checked-in SQLite fixture and isolated test databases where a write is required.
- Create `docker/sqlite/employee.db`: downloaded official SQLite sample test database fixture.
- Modify `internal/db/db.go`: add `EngineSQLite`.
- Modify `cmd/db-tui/main.go`: select `sqlite.Connect` in the connection factory.
- Modify `internal/app/connection.go`: normalize SQLite and treat its saved DSN value as a required file path without host/port credentials.
- Modify `internal/app/connection_modal.go`: add SQLite to engine choices, show the file-path field conditionally, and preserve existing engine fields/behavior.
- Modify `internal/app/connection_test.go` and `internal/app/connections_modal_test.go`: cover SQLite path validation, engine cycling, and saved settings.
- Modify `internal/config/config_test.go`: verify existing JSON shape remains compatible and SQLite paths persist in the existing `dsn` field.
- Modify `README.md`: document SQLite file connections, pure-Go driver behavior, and the `sqlite3` prerequisite for dumps.
- Modify `go.mod`/`go.sum`: add `modernc.org/sqlite` and its transitive dependencies with `go mod tidy`.

### Task 1: Add the permanent SQLite fixture

**Files:**
- Create: `docker/sqlite/employee.db`

**Interfaces:**
- Produces: a stable database path that SQLite tests open directly.

- [ ] Download the official SQLite `employee.db` sample from the SQLite test-database repository.
- [ ] Verify the downloaded file opens as SQLite and contains the expected employee sample tables.
- [ ] Confirm the fixture is not ignored by repository rules and is not accidentally staged with unrelated files.

### Task 2: Add the SQLite adapter contract and connection

**Files:**
- Modify: `internal/db/db.go`
- Create: `internal/db/sqlite/sqlite.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `db.Database`, `db.Table`, `db.PageRequest`, `db.RowPage`, `db.QueryResult`, shared export helpers, and `logger.Open`.
- Produces: `sqlite.Connect(ctx context.Context, path string) (db.Database, error)` and a concrete adapter satisfying `db.Database`.

- [ ] Add `EngineSQLite = "sqlite"` beside the existing engine constants.
- [ ] Add the pure-Go driver dependency and register it with `database/sql` through its blank import.
- [ ] Write failing connection tests for an existing fixture path, a missing/invalid path, and the adapter name.
- [ ] Implement `Connect` with `sql.Open("sqlite", path)`, `PingContext`, logger initialization, and cleanup on every failure path.
- [ ] Return the database file base name from `Name`; close both SQL resources and logger in `Close`.
- [ ] Run `go test ./internal/db/sqlite` and confirm the connection tests pass.

### Task 3: Implement SQLite schema, rows, DDL, and query behavior

**Files:**
- Modify: `internal/db/sqlite/sqlite.go`
- Modify: `internal/db/sqlite/sqlite_test.go`

**Interfaces:**
- Consumes: the adapter returned by Task 2 and the permanent fixture at `docker/sqlite/employee.db`.
- Produces: `ListTables`, `GetRows`, `TableDDL`, and `Execute` implementations matching `db.Database` limits and result types.

- [ ] Add tests that assert user tables are returned alphabetically and `sqlite_%` internal tables are excluded.
- [ ] Add tests for `GetRows` first page, offset page, `HasMore`, negative offset, invalid limits, empty names, quoted identifiers, and SQL `NULL` values.
- [ ] Add tests for DDL retrieval of a fixture table and a missing table.
- [ ] Add tests proving raw query results expose columns, values, and no more than `db.MaxQueryResultRows` rows.
- [ ] Implement table discovery with `sqlite_master`, using a parameterized query for metadata values.
- [ ] Implement identifier quoting by doubling embedded double quotes and surrounding names with double quotes; never interpolate an unquoted table name.
- [ ] Implement bounded paging with `LIMIT ? OFFSET ?` and fetch `limit + 1` rows to calculate `HasMore`.
- [ ] Implement row scanning with `database/sql`, preserving `nil` for SQL `NULL` and converting driver byte values consistently with the other adapters.
- [ ] Implement DDL lookup from `sqlite_master.sql` and ensure returned statements end with a semicolon.
- [ ] Implement query execution through `QueryContext`, reading at most `db.MaxQueryResultRows` rows and deriving a useful command tag for non-row statements.
- [ ] Run `go test ./internal/db/sqlite -run 'Test(ListTables|GetRows|TableDDL|Execute)'` and confirm all behavior passes against the fixture.

### Task 4: Implement SQLite exports and CLI dump

**Files:**
- Modify: `internal/db/sqlite/sqlite.go`
- Modify: `internal/db/sqlite/sqlite_test.go`

**Interfaces:**
- Consumes: adapter row/query methods, `csvexport.Write`, `jsonexport.Write`, `db.ValidateSelectQuery`, and dump filename helpers.
- Produces: full `Export`, `ExportQuery`, and `Dump` behavior for SQLite.

- [ ] Add table export tests for CSV and JSON using fixture data, including `NULL`, timestamps, byte values, and sanitized filenames.
- [ ] Add query export tests that accept a SELECT and reject a non-SELECT using the shared validation rule.
- [ ] Add dump tests using a controlled executable runner or test-local `sqlite3` shim to verify exact arguments, captured SQL output, missing executable errors, command stderr, and partial-file cleanup.
- [ ] Implement table exports by loading all rows, applying existing SQLite value normalization, and calling the shared writers.
- [ ] Implement query exports in a read-only transaction where supported, load all rows without the display limit, and write timestamped CSV output.
- [ ] Implement `Dump` with `exec.CommandContext(ctx, "sqlite3", path, ".dump")`, stream stdout to a timestamped sanitized `.sql` file, include stderr in wrapped errors, and remove incomplete output.
- [ ] Run `go test ./internal/db/sqlite -run 'Test(Export|Dump)'` and verify no generated files remain in the repository after tests.

### Task 5: Wire SQLite into the connection UI and application

**Files:**
- Modify: `cmd/db-tui/main.go`
- Modify: `internal/app/connection.go`
- Modify: `internal/app/connection_modal.go`
- Modify: `internal/app/connection_test.go`
- Modify: `internal/app/connections_modal_test.go`
- Modify: `internal/config/config_test.go`

**Interfaces:**
- Consumes: `db.EngineSQLite` and `sqlite.Connect` from Tasks 2–4.
- Produces: selectable SQLite engine with file-path-only connection settings and unchanged saved-connection flow.

- [ ] Add failing tests for SQLite settings with a file path, empty path rejection, engine normalization, and preservation of PostgreSQL/MySQL validation.
- [ ] Add tests for engine selector cycling through PostgreSQL, MySQL, and SQLite and for SQLite modal labels/hidden fields.
- [ ] Add tests proving a SQLite saved connection round-trips through the existing `dsn` JSON property.
- [ ] Extend the command factory switch with `db.EngineSQLite` → `sqlite.Connect`.
- [ ] Extend `normalizedEngine` and `connectionDSN` so SQLite requires only a non-empty saved DSN/file path and returns it unchanged.
- [ ] Add SQLite to `connectionEngines`; make the modal render only the database-file input for SQLite while retaining the existing input array and behavior for the server engines.
- [ ] Ensure switching engines updates defaults and does not erase values entered for another engine unless the existing modal behavior already does so.
- [ ] Run `go test ./internal/app ./internal/config ./cmd/db-tui` and verify all connection lifecycle tests pass.

### Task 6: Update documentation and validate the complete feature

**Files:**
- Modify: `README.md`
- Modify: `.github/workflows/go-test.yml` only if the fixture or SQLite tests require explicit CI setup

**Interfaces:**
- Consumes: completed SQLite adapter, UI wiring, and fixture.
- Produces: documented, reproducibly testable SQLite support.

- [ ] Document SQLite file-path examples and saved connection behavior.
- [ ] Document that SQLite connections are pure Go and do not require CGO.
- [ ] Document that `sqlite3` must be installed only for the SQLite database dump command.
- [ ] Document the checked-in SQLite test fixture location without exposing credentials.
- [ ] Run `gofmt` on changed Go files.
- [ ] Run `go mod tidy` and `go mod verify`.
- [ ] Run `go test ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `go build ./...`.
- [ ] Run `scripts/validate.sh` if its environment supports the existing PostgreSQL/MySQL integration services.
