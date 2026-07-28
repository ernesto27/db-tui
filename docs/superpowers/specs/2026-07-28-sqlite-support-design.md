# SQLite support design

## Goal

Add SQLite as a third database engine with the same user-facing capabilities as
PostgreSQL and MySQL: table browsing, paged data, raw SQL, table DDL,
CSV/JSON exports, and a portable SQL dump.

## Constraints and decisions

- Use the pure-Go `modernc.org/sqlite` driver. The application must not require
  CGO or a system SQLite library to connect to a database.
- SQLite connections are local database-file paths only. SQLite URI and
  in-memory connection modes are outside this change.
- The connection modal shows a single **Database file** input when SQLite is
  selected. It reuses the existing optional DSN storage field internally, so
  no configuration-file migration is needed.
- SQLite table and query CSV/JSON exports reuse the existing in-application
  export pipeline.
- SQLite database dumps use the installed `sqlite3` command-line tool and its
  `.dump` command. A missing executable or a failed dump is reported in the
  existing dump modal.
- The permanent test database is a repository-owned fixture at
  `docker/sqlite/employee.db`. It is downloaded once from SQLite's official
  sample test-database repository and never downloaded during tests.

## Architecture

Add `internal/db/sqlite`, which implements the existing `db.Database`
interface. No changes to that interface are required.

```text
connection modal (SQLite / database file)
  -> app.ConnectFunc
  -> cmd/db-tui connection factory
  -> internal/db/sqlite.Connect
  -> db.Database
  -> existing app navigator, data, query, export, DDL, and dump flows
```

The command package remains the composition root: it selects the SQLite adapter
alongside the existing PostgreSQL and MySQL adapters. The `app` package remains
driver-neutral and continues to use only `db.Database`.

## UI and connection flow

Add `db.EngineSQLite` and include it in the engine selector. For SQLite:

- Hide host, port, database name, username, and password inputs.
- Relabel the existing DSN input as **Database file**.
- Require a non-empty path and pass it unchanged to the SQLite adapter.
- Keep existing PostgreSQL/MySQL behavior and saved connection compatibility
  unchanged.

The database file's base name is displayed in the header as the connected
database name.

## SQLite adapter behavior

The adapter owns SQLite-specific SQL and driver concerns:

- `Connect` opens and verifies a local database file with `modernc.org/sqlite`
  and initializes the shared query logger.
- `ListTables` reads user tables from `sqlite_master`, excluding `sqlite_%`
  internal tables.
- `GetRows` validates the shared paging contract and uses safely quoted SQLite
  identifiers with `LIMIT ? OFFSET ?`.
- `TableDDL` reads the table's `CREATE TABLE` statement from SQLite schema
  metadata and returns executable SQL.
- `Execute` logs arbitrary SQL and returns no more than
  `db.MaxQueryResultRows` rows.
- `Export` and `ExportQuery` retrieve rows and use existing `csvexport` and
  `jsonexport` packages.
- `Dump` runs `sqlite3 <database-file> .dump`, writes command stdout to a
  timestamped, sanitized `.sql` filename in the current directory, cleans up a
  partial output file on failure, and wraps stderr in a useful error.

## Error handling

Connection errors preserve modal input exactly as current engines do. SQLite
operations wrap their underlying errors and honor the contexts provided by app
commands. Dump errors distinguish an unavailable `sqlite3` executable from
command execution failures. Stale async results continue to be rejected by the
app's existing session and request counters.

## Testing

Add SQLite package tests using `docker/sqlite/employee.db` as a stable,
read-only integration fixture, matching the project pattern of reusable
database assets. Tests must not alter that file.

Coverage includes:

- connection success/failure and database naming;
- user-table discovery and internal-table exclusion;
- paging limits, offsets, column/row mapping, and identifier quoting;
- DDL retrieval;
- raw query result bounds and command handling;
- CSV and JSON table/query exports;
- dump invocation, SQL output handling, missing CLI errors, and cleanup;
- app connection validation, engine cycling, and SQLite-specific modal
  rendering;
- README documentation for SQLite connections and the `sqlite3` requirement
  for dumps.

Dump tests may replace the executable runner with a controlled test double so
they verify arguments, output, and failures without requiring `sqlite3` in CI.

## Fixture provenance

The fixture will come from SQLite's official
[sample database repository](https://sqlite.org/test-dbs/), which publishes
database files used for SQLite testing.
