# Spec: Execute Selected SQL

## Objective

Allow a database user to execute only a mouse-selected portion of the raw SQL
editor with `Ctrl+P`. This makes it possible to keep multiple statements in one
script and run only the relevant statement or fragment.

When no selection exists, `Ctrl+P` retains its current behavior and executes
the complete editor contents.

## Tech Stack

- Go 1.26
- Bubble Tea v2
- Bubbles textarea v2
- Lip Gloss v2
- Testify

The existing textarea does not expose text-selection state. Selection therefore
belongs to the application query model and must not change the database
interface or individual database adapters.

## Commands

```sh
docker compose up -d
go run ./cmd/db-tui
go build ./...
scripts/validate.sh
```

No dependency change is expected. If implementation unexpectedly changes
dependencies, request approval before running:

```sh
go mod tidy
go mod verify
go build ./...
```

## Project Structure

- `internal/app/` owns SQL-editor selection state, mouse-event routing,
  selection rendering, execution choice, and application tests.
- `internal/app/textselection/` may contain reusable range behavior only if the
  query editor and data grid have genuinely compatible consumers.
- `internal/db/` and its adapters retain their current execution contracts.
- `docs/specs/` contains this feature specification.

## Behavior

1. A primary-button drag beginning within the visible SQL editor creates an
   application-owned selection.
2. The selected range is highlighted using the existing semantic selection
   colors.
3. Forward and reverse drags are supported.
4. Multiline and soft-wrapped selections resolve to the corresponding logical
   SQL substring, preserving its original whitespace and newlines.
5. A click without a drag does not create an active selection.
6. `Ctrl+P` executes the exact selected substring when an active selection
   exists.
7. If the active selection contains only whitespace, `Ctrl+P` does nothing. It
   must not fall back to executing the complete editor.
8. Without an active selection, `Ctrl+P` executes the complete non-whitespace
   editor value as it does today.
9. The selected SQL becomes `lastExecutedSQL`, so query results and subsequent
   query export correspond to the SQL that actually ran.
10. Automatic script saving continues to save the complete editor contents,
    even when only a selection is executed.
11. The selection remains active after execution, allowing repeated execution.
12. Loading or creating a script, reconnecting, or resetting the query editor
    clears the selection.
13. Input that edits the editor or moves its cursor clears the selection before
    normal textarea handling, preventing a stale range from being executed.
14. Dragging outside the selectable editor surface is clamped to visible editor
    text. Automatic scrolling during a drag is out of scope.
15. Native terminal-emulator selection is not used because its selected text is
    not available to the Bubble Tea application.

## Code Style

Selection and execution decisions remain owned by focused query-model helpers:

```go
func (m queryModel) executableSQL() string {
	if selected, ok := m.selectedSQL(); ok {
		return selected
	}
	return m.editor.Value()
}
```

Use `gofmt`, standard Go naming, semantic color constants, and explicit state
ownership. Bubble Tea updates remain free of I/O; query execution continues
through the existing command and typed-message flow.

## Testing Strategy

Add automated coverage at the lowest practical application layer before
running the final verification once.

Tests must cover:

- forward and reverse mouse drags;
- single-line, multiline, and soft-wrapped selections;
- selection at editor boundaries;
- Unicode and wide-character coordinate handling;
- click-without-drag behavior;
- selected SQL execution through the fake database;
- fallback to complete-editor execution with no selection;
- whitespace-only selection performing no execution;
- complete script contents being saved while only selected SQL executes;
- `lastExecutedSQL` containing only the executed selection;
- selection persistence after execution;
- selection clearing after editor mutation, script replacement, reset, and
  reconnection;
- mouse events outside the SQL editor not creating a selection.

Tests must not require remote databases or credentials. Run
`scripts/validate.sh` once after implementation and tests are complete.

## Boundaries

- Always:
  - Keep selection state within the application/query-editor boundary.
  - Preserve the exact selected SQL sent for execution.
  - Continue saving the complete script.
  - Use semantic selection colors.
  - Add automated behavior coverage.
- Ask first:
  - Add or upgrade dependencies.
  - Change the `db.Database` interface.
  - Add keyboard-selection bindings.
  - Add mouse-driven cursor placement, selected-text replacement, or
    drag-to-autoscroll behavior.
  - Change script persistence semantics.
- Never:
  - Infer selected statements by parsing SQL.
  - Execute the complete editor when an active selection is whitespace-only.
  - Modify editor contents merely because a range was selected or executed.
  - Depend on native terminal selection state.
  - Introduce database-engine-specific selection behavior.

## Success Criteria

- A user can mouse-drag over SQL and see the exact range highlighted.
- Pressing `Ctrl+P` with a non-whitespace selection sends only that substring
  to the active database.
- Pressing `Ctrl+P` without a selection sends the complete editor contents.
- A whitespace-only active selection executes nothing.
- Query result/export state records the SQL that actually executed.
- Saved scripts retain the complete editor contents.
- Selection behavior remains correct across multiple lines, wrapping, reverse
  drags, and Unicode text.
- Existing full-query execution behavior remains compatible.
- Relevant automated tests pass under `scripts/validate.sh`.
- No database contract, adapter, or architectural-boundary change is required.

## Out of Scope

- Keyboard-created selection.
- Copy, cut, delete, or replace-selection editing behavior.
- Mouse-based cursor placement.
- Automatic editor scrolling while dragging.
- SQL parsing or automatic statement-boundary detection.
- Database-adapter changes.
- New keyboard shortcuts.

## Open Questions

None. Mouse-only selection was explicitly chosen.
