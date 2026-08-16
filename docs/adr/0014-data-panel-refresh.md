# ADR 0014: Refresh active-relation rows from the data panel

## Status

Accepted on 2026-08-15.

## Context

ADR 0010 deliberately makes navigator activation idempotent: pressing `Enter`
for the active relation does not repeat a query or reset the displayed page.
Users nevertheless need an explicit way to retrieve changed rows for the
active relation without moving the navigator cursor or switching panels.

The data panel already owns pageable row loading, loading feedback, failure
feedback, and stale-result protection. A refresh must preserve this ownership
model and must not turn a navigator interaction into an implicit query.

## Decision

When the table-data panel has focus, pressing `r` starts a new row load for the
active relation at offset zero with no selected row. It therefore displays the
first configured page after the request succeeds.

Refresh is available only with data-panel focus. `r` has no refresh effect in
the navigator, raw-query panel, search input, or an open modal. It is ignored
when no relation is active or a row load is already in progress.

The refresh uses the existing row-load lifecycle: it clears the currently
displayed page while loading, increments the row request identity, shows the
standard loading feedback, and accepts only the matching asynchronous result.
If the request fails, the data panel shows the standard row-load error.

The table-data footer advertises the shortcut as `r refresh` while the data
panel has focus.

## Consequences

- Users can explicitly retrieve current data without changing the active or
  highlighted relation.
- Refresh always establishes a predictable first-page position rather than
  retaining a potentially invalid offset or selected row.
- Existing request IDs prevent an older page request from overwriting the
  refreshed first page.
- A failed refresh replaces the prior rows with the normal error feedback; it
  does not retain stale data while claiming it is current.
- `Enter` remains exclusively an activation gesture, preserving ADR 0010's
  idempotence rule.

## Alternatives considered

### Reuse `Enter` to refresh the active relation

Rejected. `Enter` is the navigator's relation-activation gesture and remains a
no-op for the active relation and in the data panel.

### Keep the current page on refresh

Rejected. Refresh must return to a predictable state and avoid retaining a
page offset or selection that no longer corresponds to the refreshed data.

### Make refresh global

Rejected. A global `r` could conflict with text inputs and could refresh a
relation while the user is working in another interaction context. Refresh is
specifically a data-panel action.

### Retain previous rows when refresh fails

Rejected. The existing row-load lifecycle clears stale rows and renders a
specific error for the requested load. Reusing it keeps failure behavior
consistent and unambiguous.
