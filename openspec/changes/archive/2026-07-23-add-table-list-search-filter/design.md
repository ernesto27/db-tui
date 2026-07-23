## Context

The navigator currently retains one loaded `[]db.Table`, indexes that slice directly for selection and scrolling, and routes all non-modal key presses through the root Bubble Tea model. Its footer reports the visible range against the complete table count. The application already uses `bubbles/v2/textinput` for connection fields, so an inline filter can follow an established dependency and message-update pattern without changing database access.

## Goals / Non-Goals

**Goals:**

- Make a loaded table discoverable from the keyboard with `Ctrl+F`.
- Filter immediately and case-insensitively on table name.
- Preserve correct table selection, paging, mouse behavior, and row loading for the visible result set.
- Make search state, active query, result count, and no-match state legible in the navigator and footer.

**Non-Goals:**

- Server-side table search, schema-qualified search, fuzzy ranking, or persistence of a query across connections.
- Filtering query results, rows, columns, connections, or database metadata other than public-table names.
- Changing the database interface or issuing additional introspection queries while a user types.

## Decisions

### Keep source tables and visible tables distinct

The navigator will retain the complete table list received from the database and derive a visible list from the current filter. Selection, range calculation, scroll bounds, keyboard movement, and mouse hit testing will operate on the visible list. Clearing the filter restores the full list without another database call.

This is preferred over destructively replacing `tables`, which would lose the source data and make clear/cancel behavior depend on a reload. It is also preferable to filtering in the PostgreSQL adapter because the interaction must update synchronously for each keystroke.

### Use an inline, focused text input for search mode

`Ctrl+F` will activate a navigator-owned text input and move focus to it. While active, printable input and editing keys update the query; `Enter` exits input editing while retaining the filter for list navigation; `Esc` cancels search and clears the query. The focused input will be rendered above the results, and the existing key map/footer will advertise the interaction.

An inline input is preferred over a modal because users need to see results update beside the query, and over an implicit type-to-search model because it avoids stealing navigator shortcuts such as `j`, `k`, and `q`.

### Filter with normalized substring matching

The visible list will include every loaded table whose name contains the trimmed query under case-insensitive comparison. An empty query shows all tables. The original database order is retained; there is no fuzzy score or reordering.

Substring matching supports partial names predictably, requires no new dependency, and keeps filter behavior easy to test. Prefix-only matching was rejected because it makes common suffix or infix searches fail unexpectedly.

### Make selection and load behavior result-set aware

Whenever filtering changes the visible result set, the navigator will normalize its selected index and scroll offset. A table selection initiated through the filtered list—by keyboard, mouse, or result-set normalization—will use the existing row-load command path. An empty result set has no selected table and does not start a row request. Row-load messages will continue to be validated against the current selected table before rendering.

This preserves the existing model contract that the highlighted table drives the data panel. Retaining a hidden selection was rejected because it would display data for a table the navigator does not show.

## Risks / Trade-offs

- [Filtering can invalidate the prior selection] → Clamp selection and offset after every query change; render a dedicated no-match state when the visible list is empty.
- [Async row requests may complete after another filtered selection] → Reuse current session/table validation and manually verify rapid result-set changes if the navigator state gains a request discriminator.
- [The search field consumes navigation keys] → Restrict text-input routing to active search mode and make `Enter` return control to navigator navigation.
- [Additional search UI reduces vertical list space] → Derive visible rows from layout with the search-row height included and manually inspect narrow/short terminal layouts.

## Migration Plan

No data or configuration migration is required. The feature is additive, uses existing UI dependencies, and can be rolled back by removing navigator search state and key routing.

## Open Questions

None.
