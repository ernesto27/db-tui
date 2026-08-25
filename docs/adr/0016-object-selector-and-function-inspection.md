# ADR 0016: Select navigator object types through a modal and inspect functions

## Status

Accepted on 2026-08-24.

## Context

The navigator currently uses horizontal tabs to switch among tables, views,
and materialized views. Its narrow left pane has no room to add a fourth
function category without crowding or truncating those controls.

Functions are not row-browsable relations. The existing `ListFunctions`
database contract already supplies the information needed to inspect a
function: name, arguments, return type, language, and definition. Functions
therefore need a navigator category and a dedicated right-panel presentation,
not another data-grid row source.

## Decision

`Ctrl+O` opens an **Objects** modal. The modal selects which object type the
left navigator displays:

| Object type | Availability | Activation result |
| --- | --- | --- |
| Tables | All engines | Loads pageable relation rows |
| Views | All engines | Loads pageable relation rows |
| Materialized views | PostgreSQL and Oracle | Loads pageable relation rows |
| Functions | PostgreSQL, MySQL, and Oracle | Displays function details |

The modal supports keyboard selection and closes after a choice. `Esc` closes
it without changing the selected object type. The navigator no longer renders
horizontal object-type tabs. Instead it shows the selected type as a short,
non-interactive heading above its filtered list.

Selecting an object type changes only the navigator list and its highlighted
item. It does not activate an item or replace the right panel. This retains
the explicit activation behavior defined by ADR 0010.

Function discovery is asynchronous and uses the existing `ListFunctions`
method. db-tui requests PostgreSQL's `public` schema, the connected MySQL
database name, and the connected Oracle user's schema. SQLite does not show a
Functions choice because it has no persistent function catalog.

The function list displays only function names. Each list item retains its
complete `FunctionColumns` value, so selecting duplicate names or overloads
still opens the metadata and definition returned for that item.

With the navigator focused, `Enter` on a function activates it and transfers
focus to the right panel. The right panel displays a scrollable horizontal
table with columns for Arguments, Return type, and Definition. The function
name remains the title above the table.

The detail panel scrolls with `Up`/`Down`, `j`/`k`, and the mouse wheel while
it has focus. It is read-only. Activating a table, view, or materialized view
later transfers right-panel ownership back to the normal pageable data view.

Function loads and activation state use the existing connection-session and
request-validation rules. Loading, empty-list, and error states identify the
Functions category without replacing a previously active right-panel item.

## Consequences

- The left navigator can grow to four object categories without horizontal
  layout pressure.
- Object-type switching is discoverable through one dedicated shortcut and
  does not depend on left/right tab navigation.
- Function metadata and definitions are available without treating functions
  as tables or issuing a separate lookup after activation.
- The right panel has two read-only content modes: pageable relation data and
  scrollable function details.
- PostgreSQL, MySQL, and Oracle require consistent function-list loading and
  error handling; SQLite remains explicit about its lack of stored functions.
- The implementation requires focused tests for modal navigation, category
  availability, function loading states, explicit function activation,
  duplicate-name selection, scrolling, and stale asynchronous results.

## Alternatives considered

### Add a fourth horizontal navigator tab

Rejected because the existing left pane cannot accommodate a `FUNCTIONS` tab
alongside the three relation categories.

### Give functions their own browser modal

Rejected because the object type must control the main left navigator, and
function inspection belongs in the application's normal right-hand content
area.

### Display a function's details when it is merely highlighted

Rejected because db-tui uses explicit `Enter` activation for navigator
content. Highlighting remains a non-destructive browsing operation.

### Render function names with their argument signatures

Rejected for this release. The navigator lists only names as requested; the
selected entry's arguments appear in the right-panel detail view.

### Treat SQLite functions as catalog objects

Rejected because SQLite's registered functions are process-local and do not
have a persistent database catalog to browse.
