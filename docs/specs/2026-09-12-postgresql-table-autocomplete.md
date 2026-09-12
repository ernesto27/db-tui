# Spec: PostgreSQL raw-query table autocomplete

## Objective

Help users write PostgreSQL raw SQL by automatically suggesting table names in
the raw-query editor. Suggestions use the table metadata already loaded for the
navigator's current schema and insert an exact double-quoted identifier.

For example, typing `SELECT * FROM al` suggests `Album`; accepting it replaces
`al` with `"Album"`.

## Tech Stack

- Go 1.26
- Bubble Tea v2 and Bubbles textarea
- Existing `internal/app` root model, query panel, navigator state, and tests
- Existing `testing` and `testify` utilities

No dependency, database-interface, schema, or PostgreSQL-adapter change is
required.

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
  query_autocomplete.go       Completion state, SQL-position detection,
                              matching, insertion, and popup rendering
  query_panel.go              Raw-query state and editor rendering seam
  update.go                   Completion-first key routing
  query_autocomplete_test.go  Completion behavior tests
  query_panel_test.go         Query-panel rendering tests
  update_key_test.go          Root-model key-routing tests
docs/specs/
  2026-09-12-postgresql-table-autocomplete.md
```

The feature remains in `internal/app`. It reads `navigator.tables`, which
already contains the current schema's tables. It must not issue database I/O or
change `internal/db` or `internal/db/postgres`.

## Code Style

Completion state remains owned by `queryModel`; root `Model` supplies the
current navigator tables and active engine during keyboard handling. Update
performs no I/O. Keep helpers unexported and feature-local, use existing
semantic colors, sanitize displayed table names, and run `gofmt`.

```go
if m.database != nil && m.database.Engine() == db.EnginePostgreSQL {
	if command, handled := m.query.updateTableCompletion(msg, m.navigator.tables); handled {
		return command
	}
}
```

## Behavior

- Enable only in the focused raw-query editor with an active PostgreSQL
  connection.
- Source suggestions only from `navigator.tables` for the current schema.
- Open automatically after at least one identifier character in a table
  position: after `FROM`, `JOIN`, `UPDATE`, `INTO`, `DELETE FROM`, or
  `TRUNCATE`.
- Never open inside quoted strings, quoted identifiers, line comments, or
  block comments.
- Match table names case-insensitively and display at most five matches.
- Render the popup over the editor without shifting query results.
- Highlight the first match by default.
- `Up`/`Down` moves the selection; `Tab` or `Enter` replaces the typed
  identifier with the selected table's double-quoted PostgreSQL name; `Esc`
  closes the popup.
- Continue typing and deletion refilters candidates. Hide the popup for no
  matches, lost editor focus, query execution, or connection replacement.
- Preserve existing `Tab` focus switching when completion is not open.

## Testing Strategy

Use table-driven Go tests with `testify/assert` or `testify/require`.

- Unit-test relation-position detection, including mixed case and the supported
  keywords.
- Prove strings, quoted identifiers, line comments, and block comments do not
  activate completion.
- Test case-insensitive matching, a five-result limit, no-match dismissal, and
  exact PostgreSQL double-quote escaping.
- Test selection navigation, insertion, escape dismissal, and continued
  filtering.
- Test that only current-schema navigator tables are considered.
- Test PostgreSQL-only activation, normal `Tab` behavior when no popup is
  open, and query execution/focus changes dismissing an open popup.
- Run `scripts/validate.sh` once after all feature and test changes are done.

## Boundaries

- Always: reuse navigator table metadata; keep work in `internal/app`; retain
  existing asynchronous and focus behavior; add automated coverage.
- Ask first: adding dependencies; changing `db.Database`, PostgreSQL adapter
  behavior, configuration, or shared layout conventions.
- Never: query PostgreSQL on every keystroke; modify database metadata; support
  views, columns, aliases, keywords, cross-schema suggestions, or non-
  PostgreSQL engines in this change.

## Success Criteria

1. In a PostgreSQL raw-query editor, `FROM al` automatically shows up to five
   current-schema table matches, including `"Album"` when present.
2. The popup is absent outside supported table positions and in comments or
   quoted text.
3. `Up`/`Down`, `Tab`/`Enter`, and `Esc` behave as specified.
4. Accepted suggestions replace only the active identifier token and insert a
   correctly escaped double-quoted table name.
5. Non-PostgreSQL sessions and disconnected editors retain current behavior.
6. Focus changes, execution, and connection replacement cannot leave a stale
   completion popup visible.
7. Focused tests and `scripts/validate.sh` pass.

## Open Questions

None.
