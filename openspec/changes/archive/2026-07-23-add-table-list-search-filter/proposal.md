## Why

Databases with many tables make the navigator slow to use because finding a table requires repeated scrolling. A keyboard-first filter lets users locate a known table immediately while preserving the current table-data workflow.

## What Changes

- Add a `Ctrl+F` table-search interaction to the table navigator.
- Filter the loaded table list as the user types, using a case-insensitive table-name match.
- Keep navigator selection, scrolling, mouse selection, table-data loading, and footer guidance consistent with the filtered results.
- Provide clear empty-result feedback and a way to leave search or clear its query without reconnecting.

## Capabilities

### New Capabilities

- `table-list-search`: Keyboard-driven, case-insensitive filtering of loaded database tables in the navigator.

### Modified Capabilities

<!-- None. -->

## Impact

- Affects the Bubble Tea navigator model, keyboard routing, layout/rendering, and footer help in `internal/app`.
- Does not add automated test coverage for this change, by request.
- Does not change the database interface, PostgreSQL adapter, configuration format, or external APIs.
