# Plan: Add MySQL CLI Query Execute Support

- **Date:** 2025-07-25
- **Domain(s):** database
- **Status:** Draft

## 1. Summary
Add CLI query execution (`ExecuteCLI`) for MySQL, mirroring the existing PostgreSQL adapter. The `cmd/db-tui` CLI will detect the engine from the DSN string rather than requiring a new flag.

## 2. Scope

### In scope
- `mysqlDatabase.ExecuteCLI()` method that validates SELECT, runs in a read-only transaction, and returns JSON array
- `mysql.ExecuteCLI()` package-level function (connect + execute + disconnect)
- CLI engine detection from DSN (`mysql://` schema for MySQL, else PostgreSQL default)
- Tests in `mysql/mysql_test.go`

### Out of scope / non-goals
- Adding a new `-e` / `--engine` CLI flag
- Changing the PostgreSQL adapter
- Changing the `db.Database` interface

## 3. Resolved decisions

| # | Question | Decision |
|---|----------|----------|
| 1 | Should MySQL ExecuteCLI validate SELECT queries? | Yes — `db.ValidateSelectQuery` for parity with PostgreSQL |
| 2 | Should MySQL ExecuteCLI use a read-only transaction? | Yes — `sql.TxOptions{ReadOnly: true}` like PostgreSQL's `executeAll` |
| 3 | How to select engine in CLI? | Detect from DSN (`mysql://` = MySQL, else PostgreSQL). No new flag added. |
| 4 | Should we add standalone `mysql.ExecuteCLI`? | Yes — mirrors `postgres.ExecuteCLI` |

## 4. Design

`mysqlDatabase.ExecuteCLI` runs the statement inside a read-only `sql` transaction, uses `readQueryResult` with limit 0 (no row limit for CLI), normalizes values, and marshals through `jsonexport.Marshal`.

`mysql.ExecuteCLI` connects via `mysql.Connect`, delegates to the adapter method, and defers `Close`.

CLI detects engine via `detectEngine`: checks DSN for `mysql://` prefix; everything else defaults to PostgreSQL (backward compatible).

## 5. Interfaces & contracts

- `mysqlDatabase.ExecuteCLI(ctx, statement string) (string, error)` — returns pretty-printed JSON array or wrapped error
- `mysql.ExecuteCLI(ctx, dsn, statement string) (string, error)` — connects, executes, disconnects
- CLI: `mysql://` → MySQL adapter; anything else → PostgreSQL adapter

## 6. Behavior & states
- Non-SELECT statements return an error (`"only SELECT queries..."`)
- Connection failures return errors propagated with `%w`
- Read-only transaction rolls back regardless (defer `tx.Rollback`)

## 7. Implementation tasks

### Task 1 — Add `mysqlDatabase.ExecuteCLI` method
**File:** `internal/db/mysql/mysql.go`

Add after `ExportQuery` (before `dockerContainerIDForPort`). Use `sql.TxOptions{ReadOnly: true}`, `tx.QueryContext`, `readQueryResult(rows, 0, "")`, then `jsonexport.Marshal`.

### Task 2 — Add `mysql.ExecuteCLI` function
**File:** `internal/db/mysql/mysql.go`

Add after `mysqlDatabase.ExecuteCLI`. Call `Connect`, defer `Close`, delegate to adapter method.

### Task 3 — Update CLI engine detection and dispatch
**File:** `cmd/db-tui/main.go`

Replace the hardcoded `postgres.ExecuteCLI` call with a `switch` on the engine from DSN. Default to PostgreSQL. No new flag.

### Task 4 — Add tests
**File:** `internal/db/mysql/mysql_test.go`

Add `TestExecuteCLI` with subtests: returns JSON rows, validates SELECT only. Use `connectWorld` helper.

## 8. Testing

- **Unit tests:** `mysql/mysql_test.go` — `TestExecuteCLI` (SELECT returns JSON, non-SELECT returns error)
- **Integration tests:** Run against `127.0.0.1:3307` (docker compose `mysql` service) using `worldDSN`

## 9. Acceptance criteria
- `db-tui -c "mysql://db_tui:db_tui@tcp(127.0.0.1:3307)/world" -q "SELECT * FROM city LIMIT 5"` returns JSON array
- Non-SELECT query returns error
- Existing PostgreSQL CLI usage continues to work unchanged (backward compatible)

## 10. Risks & open items
None — all decisions recorded above.
