# ADR 0011: Confirm single-row deletion from the data panel

## Status

Accepted on 2026-08-10.

## Context

db-tui can display rows from base tables, views, and materialized views. Deleting a displayed row is destructive, so it needs an explicit confirmation and a stable way to identify exactly one database row.

The existing row editor already establishes the application's modal conventions: `Enter` advances or confirms, `Esc` returns or cancels, asynchronous work shows progress, and completion or failure remains visible until the user dismisses it.

## Decision

When the data panel has focus, the active relation is a base table, and a row is selected, `d` opens a delete confirmation for that row. The shortcut is unavailable for ordinary views and materialized views.

The confirmation names the table but does not display primary-key names or values. Its prompt asks whether to delete the row and uses the edit modal's controls: `Enter` confirms and `Esc` cancels/backtracks.

Confirmation is available for every selected base-table row. The UI does not hide or pre-reject the action when the table has no primary key. On confirmation it invokes `db.Database.DeleteRow` asynchronously. That operation must safely reject a missing or incomplete primary key; it must not issue an unqualified `DELETE`. The failure is shown in the delete modal using the edit modal's failure and dismissal behavior.

The row identity sent to the database consists of the selected row's original primary-key values. This supports composite keys and avoids deriving identity from user-visible formatting.

While deletion is in progress, the modal shows a spinner. On success it reports completion and reloads the active table at the current page offset and selected row index. Normal selection clamping then selects the row now at that position, or the preceding final row when the deleted row was last on the page.

## Consequences

- A destructive action always requires a second deliberate `Enter`.
- Users receive a consistent modal interaction across editing and deletion.
- Keyless tables expose an explicit, safe operation failure rather than silently making deletion unavailable.
- Views and materialized views remain read-only in the row browser.
- The refresh keeps users in their current data-page context after a successful deletion.

## Alternatives considered

### Hide deletion for keyless tables

Rejected. The interaction remains available for every selected base-table row; identity validation belongs to the database operation, which reports why it cannot safely delete the row.

### Show primary-key values in the confirmation

Rejected. The confirmation identifies the table only, avoiding unnecessary exposure of key values in the destructive-action prompt.

### Support deletion from views and materialized views

Rejected for now. Their updatability differs across engines and relations. The initial behavior is limited to base tables.

### Issue deletion without a primary-key predicate

Rejected. A failed safe operation is preferable to any possibility of deleting multiple rows.
