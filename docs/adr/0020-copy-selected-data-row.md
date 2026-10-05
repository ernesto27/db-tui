# ADR 0020: Copy selected data rows with a focused keyboard action

## Status

Accepted on 2026-10-04. Implemented with automated regression coverage;
`scripts/validate.sh` passed, including formatting, vet, tests, and race tests.

## Context

The table browser already selects rows. Raw SQL results currently have only a
viewport and no highlighted row. Existing data-grid mouse copying operates on
rendered text, which can be truncated and does not provide a shortcut for
copying a complete row.

The requested action is lowercase `c` with data focus, available in both SQL
data panels. The user chose tab-separated values and explicitly deferred
escaping.

## Decision

Copy exactly the selected row, using every underlying value in column order,
with literal tabs between values and no headers or added final newline.
Preserve embedded tabs, newlines, and quotes without escaping. Represent SQL
`NULL` as `NULL`, byte slices as their string contents, and other values with
`fmt.Sprint`.

Give raw SQL results a selected-row index owned by `queryModel`. Successful
results select their first row; keyboard navigation moves the highlight and
keeps it visible. Preserve the table browser's existing selection behavior.

Route copying only when the appropriate SQL data surface owns focus and has a
valid, successfully loaded row. Editor input and active overlays keep their
existing key handling. Use the existing `tea.SetClipboard` command and share
one application-owned row-serialization helper.

Display a two-second `Copied to clipboard` toast for valid copy actions.
Use a small bordered Lip Gloss layer at the bottom-right above the footer,
preserving footer help and keyboard focus. Dialogs render above the toast.
Repeated copies restart the toast, with request identity preventing older
expiry messages from dismissing newer feedback. Session identity keeps the
toast from appearing for a different connection. This replaces the initial
footer notice at the user's request.

Extend the same action to the Redis key grid at the user's request. Copy its
cached key, value, type, and TTL columns using the shared helper and toast.
Redis's adapter already formats and sanitizes these values; this action copies
the complete loaded strings, not original Redis bytes. It performs no extra
Redis reads and does not add row selection to plain-text command replies.

The detailed behavior and acceptance coverage are defined in the
[feature specification](../specs/2026-10-04-copy-selected-row.md). No database
contract or adapter change is needed.

## Consequences

- Users can copy a complete row without relying on rendered widths or mouse
  text selection.
- SQL query results acquire explicit selection state as well as scroll state.
- Embedded delimiters can produce additional pasted columns or lines; the
  payload deliberately does not guarantee round-trip serialization.
- Clipboard delivery uses the terminal's existing capability and cannot be
  acknowledged by the application merely from command dispatch.
- This adds a complementary interaction to
  [ADR 0012](0012-data-grid-drag-selection.md); it does not replace mouse
  character-range copying.

## Alternatives considered

### Copy the rendered row

Rejected because hidden columns and truncated cell values would be lost.

### Copy the first visible query row without selection

Rejected because an explicit highlight makes the copy target clear and allows
keyboard selection without forcing that row to the top of the viewport.

### Quote tab-separated fields, or copy CSV or JSON

Deferred because the user explicitly selected literal tab separators and does
not want escaping in this version.
