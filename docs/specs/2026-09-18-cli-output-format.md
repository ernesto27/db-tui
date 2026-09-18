# Spec: Non-interactive CLI output format

## Objective

Let the non-interactive CLI emit CSV as well as JSON, selected by a new `-t`
flag. Today `db-tui -q <SQL> -c <DSN>` always prints a JSON array, because the
output format is fixed by the `Database.ExecuteCLI` return type. `todo.txt`
records a CSV CLI mode as complete, but only JSON exists.

The user is the operator at a shell prompt who wants query output in the shape
the next tool in the pipeline expects. Success is `-t csv` producing a CSV
document on standard output for every supported engine, with `-t` omitted
behaving exactly as the CLI behaves today.

CSV serialization reuses `internal/csvexport` rather than introducing a second
CSV formatter, so CLI output and TUI table export cannot drift apart.

## Tech Stack

- Go 1.26
- `internal/db` driver-neutral contracts and validation
- `internal/db/{postgres,mysql,oracle,sqlite,sqlserver}` adapters
- `internal/{csvexport,jsonexport}` export serialization
- Go `testing` and `testify`

No new dependency, database schema, or configuration key is required.

## Commands

```sh
go build ./...
go test ./...
scripts/validate.sh
```

Run automated verification once after implementation and formatting are
complete. The user performs manual terminal testing.

## Project Structure

```text
cmd/db-tui/
  main.go                       -t flag parsing, format validation, stdout writing
  main_test.go                  New: runCLI flag and exit-code tests
internal/db/
  db.go                         ExecuteCLI contract
  postgres/postgres.go          PostgreSQL CLI format branch
  mysql/mysql.go                MySQL CLI format branch
  oracle/oracle.go              Oracle CLI format branch
  sqlite/sqlite.go              SQLite CLI format branch
  sqlserver/sqlserver.go        SQL Server CLI format branch
  */*_test.go                   Per-adapter format coverage
internal/csvexport/
  csvexport.go                  In-memory CSV marshalling behind the file writer
  csvexport_test.go             Marshal coverage
internal/app/
  test_helpers_test.go          Fake database signature update
docs/specs/
  2026-09-18-cli-output-format.md
```

Dependencies stay directed inward. `cmd` owns flag parsing and stdout;
adapters own serialization of their own result sets and must not import
`internal/app`.

## Code Style

The format threads through the existing boundary as a documented parameter,
matching how `Export` already takes an export type:

```go
// ExecuteCLI runs a SELECT in a read-only transaction and returns all of its
// rows serialized as format, which must be db.ExportTypeJSON or
// db.ExportTypeCSV.
ExecuteCLI(ctx context.Context, statement, format string) (string, error)
```

The format reuses the existing `db.ExportTypeJSON` and `db.ExportTypeCSV`
constants rather than introducing a parallel vocabulary. Flag validation is a
CLI concern, so `validateOutputFormat` lives in `cmd/db-tui` rather than in
`internal/db`; it returns an actionable error naming the accepted values.
Adapters independently reject an unknown format, because `ExecuteCLI` is an
exported boundary that the CLI is not the only possible caller of.

`csvexport` gains an in-memory `Marshal` that `Write` delegates to, mirroring
the existing `jsonexport.Marshal` / `jsonexport.Write` pair, so the CLI can
serialize to standard output without touching the filesystem.

The format switch itself is written once, in `internal/db`, beside the other
neutral helpers that every adapter already calls:

```go
func SerializeCLIResult(engine string, columns []string, rows [][]any, format string) (string, error)
```

Each adapter's `ExecuteCLI` reduces to `executeAll` plus one delegating call,
passing its own `Engine()` identifier so no engine name is written as a
literal. Errors therefore name the engine by its `Engine*` constant.

Exported declarations carry Go doc comments, adapter methods accept
`context.Context`, and propagated errors are wrapped with `%w`.

## CLI behavior

- `-t` accepts `json` and `csv`, case-sensitive, and defaults to `json` when
  omitted. Existing invocations keep their current output byte-for-byte.
- An unrecognized `-t` value is a usage error: a message naming the accepted
  values on standard error, exit code 2, and no database connection attempted.
- `-t` supplied without both `-q` and `-c` is a usage error with exit code 2.
  Silently ignoring the flag and starting the TUI would hide the mistake.
- The usage line becomes `Usage: db-tui [-q <SQL> -c <DSN> [-t json|csv]]`.
- Output ends with exactly one trailing newline. `jsonexport.Marshal` returns
  no trailing separator and `csvexport.Marshal` ends with one, so the CLI must
  not unconditionally use `fmt.Fprintln` for both.
- Exit codes are unchanged otherwise: 2 for usage errors, 1 for query failure,
  0 for success.

## Engine behavior

Every adapter keeps its existing `executeAll` path: `ValidateSelectQuery`, a
read-only transaction, unbounded row reading, and rollback on completion. Only
serialization branches on format.

PostgreSQL keeps `normalizeJSONValues` on the JSON branch only. That step
exists because `encoding/json` ignores `fmt.Stringer`, so a
`pgtype.InfinityModifier` would otherwise serialize as its underlying integer.
`csvexport.formatValue` falls through to `fmt.Sprint`, which does honor
`fmt.Stringer`, so the CSV branch renders the same value correctly without it.

CSV rendering is inherited from `csvexport` and is deliberately unchanged: SQL
`NULL` becomes an empty field, a header row is always written, `time.Time`
uses RFC3339Nano, `[]byte` is written as a raw string, and records are
separated by `\n`.

CSV preserves the column order of the query. JSON does not, because
`jsonexport.Marshal` builds a `map[string]any` per row and `encoding/json`
sorts map keys. This difference is pre-existing and out of scope.

## Testing Strategy

Use table-driven subtests with `testify/assert` or `testify/require`.

- `csvexport.Marshal`: header row, SQL `NULL` as an empty field, embedded
  comma and double-quote quoting, `time.Time` formatting, `[]byte` handling,
  a row whose length does not match the columns, and the empty result set.
- `csvexport.Write`: unchanged file output after delegating to `Marshal`.
- `validateOutputFormat`: both accepted values, the empty string, and an
  unknown value including a wrong-case one.
- Each adapter's existing `TestExecuteCLI` table gains CSV cases alongside its
  JSON cases, plus an unknown-format rejection. PostgreSQL has no
  `TestExecuteCLI` today; add one at the same layer as its siblings.
- `runCLI`: default format is JSON, `-t csv` selects CSV, an unknown `-t`
  exits 2 without connecting, `-t` without `-q`/`-c` exits 2, and output ends
  with exactly one newline. Cover this with an injected execution seam or
  SQLite fixture rather than a remote engine.
- Update `fakeDatabase.ExecuteCLI` in `internal/app/test_helpers_test.go` and
  any compile-time interface assertions for the new signature.

No test may require remote credentials or a remote database.

## Boundaries

- Always: default to JSON when `-t` is absent; reuse `db.ExportType*`
  constants; reuse `internal/csvexport` for CSV bytes; validate the format
  before opening a connection; wrap propagated errors with `%w`; add focused
  automated coverage.
- Ask first: add a third output format; change CSV `NULL`, quoting, or header
  behavior; change the JSON document shape; rename or repurpose the `-t` flag;
  replace the threaded format parameter with a rows-returning interface
  method.
- Never: write a second CSV formatter; change TUI export behavior; change
  `Database.Execute`; emit partial output after an error; commit credentials,
  logs, or generated binaries.

## Success Criteria

1. `db-tui -q <SELECT> -c <DSN> -t csv` prints a CSV document with a header
   row on standard output for PostgreSQL, MySQL, Oracle, SQLite, and SQL
   Server.
2. `-t json` and an omitted `-t` produce today's JSON output unchanged.
3. An unrecognized `-t` value, or `-t` without `-q` and `-c`, exits 2 with an
   actionable message and no database connection.
4. `Database.ExecuteCLI` carries the format parameter and every adapter and
   fake implements the new signature.
5. CSV bytes come from `internal/csvexport`; no second CSV formatter exists.
6. Output ends with exactly one trailing newline in both formats.
7. README documents `-t`, and the `todo.txt` CSV CLI entry is accurate.
8. Focused tests plus `scripts/validate.sh` pass.

## Open Questions

None.
