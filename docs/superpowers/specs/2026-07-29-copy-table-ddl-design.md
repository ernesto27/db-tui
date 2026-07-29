# Copy table DDL design

## Goal

Let a user copy the complete DDL script shown in the table DDL modal without
leaving the terminal application.

## Interaction

When the DDL modal has loaded a non-empty script without an error, pressing
`c` copies the complete, original SQL string to the system clipboard. The modal
remains open and preserves its scroll position. Its footer includes `c copy`
and shows a short `Copied DDL` confirmation after the command is issued.

The copy key has no effect while the DDL is loading or when loading failed.
The copied value is the original DDL string, not the sanitized, wrapped, or
visible subset used for rendering.

## Architecture

The DDL-modal keyboard handler owns the behavior. On an eligible `c` keypress,
it records the confirmation state on `ddlModal` and returns Bubble Tea's
`tea.SetClipboard(modal.sql)` command. Bubble Tea emits the OSC 52 terminal
clipboard sequence, so no platform-specific subprocess or new dependency is
required.

The existing modal view renders the copy hint and confirmation in its footer.
No database, query, or modal lifecycle behavior changes.

## Error handling

Clipboard delivery is delegated to the terminal protocol. The app confirms
that it issued the copy command; it cannot reliably detect terminal support
for OSC 52. Ineligible modal states are intentional no-ops.

## Tests

Unit tests will verify that:

- `c` in a loaded DDL modal returns a command that carries the full original
  SQL and records the copied confirmation.
- The footer shows the copy affordance and confirmation.
- `c` returns no command while loading or after a DDL error.
- Existing scrolling and close behavior continue unchanged.

## Scope

This change applies only to the table DDL modal. It does not add clipboard
commands to query results, exports, or other UI surfaces.
