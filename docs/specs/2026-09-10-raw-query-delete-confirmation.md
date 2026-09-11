# Spec: Raw-query DELETE confirmation

## Objective

Prevent accidental row deletion from the raw-query panel.

When a user submits raw SQL that contains a `DELETE` statement, db-tui must
show a confirmation modal before starting database work. This applies to:

- a direct `DELETE`;
- `WITH … DELETE` statements; and
- multi-statement batches containing a `DELETE`.

The user can confirm to execute the original SQL unchanged, or cancel to close
the modal without executing anything. Statements other than `DELETE` remain
outside this feature's scope.

## Tech Stack

- Go
- Bubble Tea v2
- Existing `internal/app` modal, root-model, and async-command patterns
- Existing `testing` and `testify` test utilities

No adapter, database-interface, dependency, or schema changes are required.

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
  query_panel.go                 Raw-query state and query-submission helpers
  update.go                      Root message routing and query execution start
  *_modal.go                     Modal state, update behavior, and rendering
  query_panel_test.go            Raw-query unit tests
  update*_test.go                Root-model transition tests
docs/specs/
  2026-09-10-raw-query-delete-confirmation.md
```

The feature remains within `internal/app`. It must not change `internal/db` or
database-adapter packages.

## Code Style

Keep detection and state transitions local to the app package. `Update` does
not execute SQL directly; confirmation returns the existing asynchronous
`tea.Cmd`.

```go
if queryContainsDelete(sql) {
    modal := newRawQueryDeleteModal(sql)
    m.rawQueryDeleteModal = &modal
    return m, nil
}

return m, m.startQuery(sql)
```

Use `gofmt`, unexported feature-local identifiers, typed messages where a
message is needed, existing semantic colors, and shared UI text constants.

## Testing Strategy

Use table-driven Go tests with `testify/assert` or `testify/require`.

- Classify direct, mixed-case, whitespace-prefixed, and comment-prefixed
  `DELETE` statements.
- Classify `WITH … DELETE` statements.
- Classify batches containing a `DELETE` after another statement.
- Do not show the modal for `SELECT`, `INSERT`, `UPDATE`, `TRUNCATE`, DDL, or
  query text where `DELETE` appears only as a string or comment.
- Verify raw `DELETE` submission does not set query loading state or invoke
  database execution before confirmation.
- Verify confirm executes the original query through the existing query command.
- Verify cancel/escape closes the modal without execution and preserves the
  editor contents.
- Verify modal-first routing prevents query-panel input while confirmation is
  open.

## Boundaries

- Always: require confirmation before every detected raw-query `DELETE`; retain
  async query execution and stale-result protections; add automated coverage.
- Ask first: adding a SQL parser dependency; changing database interfaces,
  adapter behavior, or modal conventions.
- Never: execute a detected `DELETE` before confirmation; modify submitted SQL;
  extend this confirmation behavior to `UPDATE`, `TRUNCATE`, DDL, or other
  statements in this change; commit credentials or generated artifacts.

## Success Criteria

1. Submitting a raw SQL batch containing a `DELETE` opens a confirmation modal
   and performs no database work yet.
2. Confirming starts the current raw-query execution flow with the exact,
   original SQL text.
3. Cancelling or pressing escape closes the modal and executes no SQL.
4. Direct `DELETE`, `WITH … DELETE`, and multi-statement batches containing
   `DELETE` are all detected, regardless of case or leading whitespace/comments.
5. `DELETE` text inside comments or string literals alone does not trigger the
   modal.
6. Non-`DELETE` raw SQL keeps its existing submission behavior.
7. The focused automated tests and `scripts/validate.sh` pass.

## Open Questions

None.
