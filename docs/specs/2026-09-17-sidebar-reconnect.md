# Spec: Sidebar Database Reconnect

## Objective

Add an `r` shortcut that reconnects to the active database only while the
left navigator has focus.

The feature lets db-tui users refresh a stale database session and reload its
schema without restarting the application. A successful reconnect replaces the
active session and resets the navigator to its initial state. A failed reconnect
shows a sanitized error while preserving the existing session and navigator.

## Tech Stack

- Go
- Bubble Tea v2
- Existing `internal/app` root model and asynchronous `tea.Cmd` flow
- Existing driver-neutral `db.Database` interface and injected `ConnectFunc`

## Commands

```sh
go test ./...
go build ./...
scripts/validate.sh
```

## Project Structure

```text
internal/app/             Root model, asynchronous reconnect lifecycle,
                          navigator key handling, status/error rendering,
                          shortcut help, and model tests.
docs/specs/               Feature specifications.
```

The change remains within `internal/app`. It must not add methods to
`internal/db.Database` or change database adapters.

## Code Style

Use typed Bubble Tea messages and commands for all connection I/O. `Update`
only changes model state and returns commands; commands create the connection
and return a result message.

```go
func (m Model) startReconnect() tea.Cmd {
	m.connectionAttempt++
	attempt := m.connectionAttempt
	settings := m.savedConnection

	return func() tea.Msg {
		database, err := m.connect(settings)
		return reconnectResultMsg{
			attempt:  attempt,
			database: database,
			err:      err,
		}
	}
}
```

Names use Go conventions, errors are wrapped/sanitized before terminal
rendering, and user-facing text is kept in shared constants or focused helpers.

## Testing Strategy

Add focused `internal/app` model tests using the existing fake/injected
connector.

- `r` starts a reconnect only with navigator focus; it does nothing with data
  panel focus, modal focus, or other non-navigator input paths.
- A successful reconnect adopts the new database, closes the old database,
  increments/replaces the session identity, resets navigator state, and starts
  database-object loading for the new session.
- A failed reconnect preserves the old database and navigator and exposes an
  error state suitable for the existing UI.
- A stale result from an earlier reconnect cannot replace the result of a newer
  attempt; a stale successful database is closed.
- Shortcut help advertises `r` only in the relevant navigator context.

Run the repository validation once after implementation:

```sh
scripts/validate.sh
```

## Boundaries

- Always: keep I/O in `tea.Cmd` values; use connection-attempt and session
  identities; close replaced and stale successful connections; add focused
  automated tests; sanitize displayed errors.
- Ask first: adding dependencies, changing the `db.Database` contract,
  altering adapter behavior, changing configuration persistence, or changing
  global shortcut behavior.
- Never: reconnect synchronously from `Update` or `View`; import database
  adapters from `internal/app`; expose credentials in errors/logs; preserve
  navigator selection/expansion in this first version.

## Success Criteria

1. Pressing `r` with navigator focus starts an asynchronous reconnect using the
   active saved connection settings.
2. Pressing `r` outside the navigator does not reconnect.
3. During a reconnect, result handling is identity-safe: an older attempt can
   never replace a newer connection.
4. On success, the new connection becomes active, the prior connection is
   closed, navigator state resets to its initial state, and schema object
   loading begins for the new session.
5. On failure, the old connection and its navigator state remain available and
   the user sees a sanitized reconnect error.
6. Navigator-scoped shortcut help documents `r`.
7. Focused automated tests and `scripts/validate.sh` pass.

## Open Questions

None.
