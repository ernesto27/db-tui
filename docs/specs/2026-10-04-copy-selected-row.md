# Spec: Copy the Selected Data Row

## Status

Approved design on 2026-10-04. Implemented with automated regression coverage;
`scripts/validate.sh` passed, including formatting, vet, tests, and race tests.

## Objective

Pressing lowercase `c` while a data panel has focus copies exactly its
highlighted row to the clipboard. This applies to the SQL table browser,
results from the raw SQL editor, and the Redis key grid.

The clipboard contains every value in the row, in column order, separated by
literal tab characters. It contains no column headers or added final newline.

## Decisions

- Both SQL data panels and the Redis key grid support the same shortcut,
  serialization, and toast.
- SQL query results gain a highlighted row, initially the first returned row.
- `Up`/`Down` and `j`/`k` move the query-result highlight.
- Copy includes full loaded row values and all columns, including columns
  outside the visible area. Display truncation does not affect copied values.
- Values are not quoted, escaped, or sanitized for clipboard output. Embedded
  tabs, newlines, and quotes remain unchanged, as explicitly requested.
- SQL `NULL` becomes `NULL`; byte slices become their direct string contents;
  other values use their Go text representation (`fmt.Sprint`).

## Shortcut Scope

In the table browser, copying requires data-panel focus, an active
row-browsable relation, a successfully loaded page, and a valid selected row.
The copied row belongs to the active relation, even when another relation is
highlighted in the navigator.

In the raw SQL panel, copying requires result focus, a successful completed
execution, and a valid highlighted result row. With editor focus, `c` retains
its ordinary text-input behavior.

In the Redis key grid, copying requires data-panel focus, a connected Redis
client, a successfully loaded key page, and a valid selected row. Copy all four
columns in grid order: key, value, type, and TTL. Use the cached row from the
selected logical database and page without issuing another Redis request.
Redis rows already contain the adapter's formatted, terminal-safe values;
copy preserves that loaded representation without further truncation or
escaping. It does not reconstruct original Redis bytes or collection values.

An empty, loading, failed, disconnected, or invalid-selection state does not
change the clipboard. Active overlays retain their existing key routing;
`c` must not reach the underlying data panel while an overlay owns input.

This feature is scoped to SQL relation rows, SQL query results, and Redis key
rows. It does not add copying to Redis command replies, function definitions,
or extension metadata. Redis command-editor `c` remains ordinary text input.
Existing DDL-modal copying and mouse text selection retain their current
behavior.

## Query-Result Selection

1. A successful execution with rows selects the first row.
2. `Up`/`Down` and `j`/`k` move one row, bounded by the loaded result.
3. `PgUp`/`PgDown` move the selection by the visible result-page size, clamped
   to the loaded result bounds.
4. Keyboard navigation keeps the highlighted row visible using the grid's
   actual layout, including rows whose displayed values wrap.
5. Mouse-wheel scrolling retains its scrolling behavior. If it scrolls the
   highlight outside the visible rows, selection moves to the nearest visible
   row, so the row copied by `c` remains visible.
6. Resizing clamps selection and viewport as needed and keeps the selected
   row visible.
7. Switching between editor and results preserves the selected row.
8. Starting a new execution, clearing the query, or resetting the connection
   clears the previous result selection. Existing session and request checks
   continue rejecting stale query results.

The table browser retains its existing row-selection and paging behavior.
Copying does not move the selection, execute SQL, reload data, or modify rows.

## Clipboard Representation

Use one application-owned helper for both panels. Its intended behavior is:

```go
func rowClipboardText(row []any) string {
	values := make([]string, len(row))
	for i, value := range row {
		switch value := value.(type) {
		case nil:
			values[i] = "NULL"
		case []byte:
			values[i] = string(value)
		default:
			values[i] = fmt.Sprint(value)
		}
	}
	return strings.Join(values, "\t")
}
```

For example, values `42`, `Alice`, and SQL `NULL` copy as
`42<TAB>Alice<TAB>NULL`, where `<TAB>` denotes a literal tab character.

Because embedded tabs and newlines remain literal, pasted text can appear as
extra columns or lines. SQL `NULL` and the string `NULL` have the same copied
representation. This is a simple copy action, not a lossless interchange format.

Use the existing Bubble Tea `tea.SetClipboard` command. Clipboard handling
depends on the user's terminal and its configuration; dispatching the command
does not provide confirmation that the destination clipboard accepted it.

After a valid copy action, show `Copied to clipboard` in a small bordered
toast at the bottom-right of the terminal, above the footer, for two seconds.
The toast is a Lip Gloss compositing layer; it neither replaces footer help
nor takes keyboard focus. Dialogs render above it. Clamp its text and position
on resize, and omit it when the terminal is too small to fit a bordered toast.
Repeated copies restart that duration; an earlier timer must not dismiss a
newer toast. The toast belongs to its connection session and must not appear
after the session changes. Invalid copy actions do not create a toast. This
feedback indicates command dispatch, not an acknowledgement from the terminal.

## Ownership and Implementation Boundaries

- `internal/app` owns keyboard routing, row serialization, query selection,
  rendering, and tests.
- `queryModel` owns query-result selection independently of its viewport.
- The root model groups toast visibility, connection session, and expiry
  request identity in `rowCopyState`.
- Reuse existing semantic row-highlight colors and layout calculations.
- Updates return clipboard commands rather than performing clipboard I/O.
- No database-interface, adapter, dependency, or architectural-rule changes
  are required.
- Add `c` to relevant shortcut help, and update the README when the feature
  is implemented.

## Automated Acceptance Coverage

Keep coverage concise and focused on the most important behavior:

- full, ordered clipboard values, SQL `NULL`, byte slices, and literal tabs,
  newlines, and quotes;
- selected-row copying from SQL tables, query results, and Redis keys;
- normal `c` text input in SQL and Redis editors;
- visible toast feedback without replacing the footer, and expiry that does
  not dismiss a newer copy's toast;
- initial query selection, row/page navigation, bounds, visibility on resize,
  and reset on a new execution.

Use application tests and fake databases without remote credentials. Run
`scripts/validate.sh` after implementation and regression coverage are complete.
The user performs manual clipboard testing; automated application tests cannot
verify the destination terminal's clipboard support.

## Out of Scope

- Multiple-row or cell-only copying.
- Headers, format pickers, escaping, quoting, or hexadecimal binary formatting.
- Additional mouse row-selection interactions in the SQL result panel.
- New clipboard dependencies or terminal-specific fallback utilities.
- Query execution, export, or database-adapter changes.

## Related Documents

- [ADR 0020: Copy selected data rows](../adr/0020-copy-selected-data-row.md)
- [Glossary](../glossary.md)
- [Architecture](../../ARCHITECTURE.md)
- [Lip Gloss v2 compositing](https://github.com/charmbracelet/lipgloss/blob/v2.0.5/README.md)
- [Bubble Tea v2 Tick](https://pkg.go.dev/charm.land/bubbletea/v2@v2.0.8#Tick)
