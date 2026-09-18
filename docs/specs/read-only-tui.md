# Spec: Session Read-Only Mode

## Objective

Add a session-only read-only switch for the connected database session.
`Alt+R` toggles it. When active, the header visibly includes `READ ONLY`.
The setting is never written to connection configuration and resets when the
application starts, reconnects, or adopts another saved connection.

Raw-query execution passes `db.QueryExecutionReadOnly` to `Database.Execute`
when enabled; the adapters that already implement this mode enforce it.

SQL Server has no backend read-only enforcement. When read-only mode is active,
the TUI must reject SQL Server raw queries containing a write-side statement
before calling `Execute`.

## Tech Stack

- Go 1.26
- Bubble Tea v2
- Existing `internal/app` root model and `internal/db` execution-mode contract
- Go `testing` and `testify`

No dependency or database-schema change is required.

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
internal/app/
  model.go                  Session-only read-only state
  keymap.go, update.go      Alt+R binding and state transition
  view.go                   Header indicator and footer help
  shortcuts_modal.go        Shortcut reference
  commands.go               Execution mode passed to Database.Execute
  read_only.go              SQL Server statement validation and shared text
  *_test.go                 Focused app behavior tests
docs/specs/read-only-tui.md Feature specification
```

The change belongs in `internal/app`. It must not import adapters, change the
`db.Database` interface, or modify adapter behavior.

## Behavior

1. `Alt+R` toggles read-only only while a database is connected.
2. When active, the header renders `READ ONLY` using an existing or new
   semantic application color.
3. The state is in-memory only. It resets when the application starts, a
   database is reconnected, or another saved connection is adopted.
4. All non-SQL-Server raw queries run through `Database.Execute` with
   `db.QueryExecutionReadOnly` when active, otherwise
   `db.QueryExecutionDefault`. Backend errors remain the query-result error.
5. In SQL Server read-only mode, the TUI lexically validates the executable
   raw query before calling `Execute`. Rejected queries display an actionable
   read-only error and do not start execution or save a script.
6. The SQL Server validator is case-insensitive, ignores comments, string
   literals, and quoted identifiers, accepts read-only query forms, and
   rejects write-side, transactional, locking, and session-control forms.
   This includes `INSERT`, `UPDATE`, `DELETE`, `MERGE`, `CREATE`, `ALTER`,
   `DROP`, `TRUNCATE`, `GRANT`, `REVOKE`, `DENY`, `EXEC`/`EXECUTE`,
   `BULK INSERT`, `SELECT INTO`, `BEGIN`, `COMMIT`, `ROLLBACK`, and `SET`.
7. Existing write controls remain visible. This feature does not hide or
   disable row actions, and it does not add a generic TUI SQL validator for
   non-SQL-Server engines.

## Code Style

Keep state, key handling, validation, and rendering inside `internal/app`.
`Update` and `View` perform no I/O; query execution remains a `tea.Cmd` that
returns a typed result message. Keep user-visible text in named constants or a
focused helper, and use semantic colors from `colors.go`.

```go
func (m *Model) queryExecutionMode() db.QueryExecutionMode {
	if m.readOnly {
		return db.QueryExecutionReadOnly
	}
	return db.QueryExecutionDefault
}
```

## Testing Strategy

Use focused table-driven `internal/app` tests and the existing fake database.

- `Alt+R` toggles only with an active connection; it does not persist in
  configuration.
- The flag resets when a connection is adopted or the user reconnects.
- The header and shortcut reference expose the active state and key binding.
- The fake records the mode supplied to `Execute`; default and read-only modes
  are asserted separately.
- SQL Server write-side statements are rejected without calling `Execute`.
- SQL Server read statements and keywords within comments, literals, or quoted
  identifiers are allowed.
- Non-SQL-Server queries reach the backend regardless of lexical content, so
  the backend remains their enforcement point.

## Boundaries

- Always: keep the state in memory; preserve Bubble Tea update/view purity;
  add focused tests; run `scripts/validate.sh` once after implementation.
- Ask first: add dependencies, change the database contract, persist the
  setting, or expand raw-query validation beyond SQL Server.
- Never: modify adapter enforcement, hide write controls, claim the SQL Server
  guard is authorization, or store the state in a saved connection.

## Success Criteria

1. `Alt+R` visibly and reversibly toggles `READ ONLY` for the active live
   connection.
2. The state resets when the TUI adopts a new or reconnected database.
3. The `Database.Execute` mode matches the state for every raw query.
4. SQL Server write-side raw queries cannot reach `Execute` while active.
5. Existing non-SQL-Server backend enforcement and visible write controls are
   unchanged.
6. Focused automated tests and `scripts/validate.sh` pass.

## Open Questions

None.
