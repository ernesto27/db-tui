# Raw Query Panel Design

## Goal

Add a raw SQL panel that lets a connected user execute any PostgreSQL statement and inspect its result without leaving db-tui. The panel replaces the right-side table-data panel while open; the table navigator stays visible.

## User interaction

- `Ctrl+R` opens the raw query panel.
- `Ctrl+T` returns to the table-data panel.
- The table-data panel is the default view and keeps its loaded page and selection while the raw query panel is open.
- The raw-query panel starts with the SQL editor focused. Plain Enter inserts a newline; `Ctrl+P` submits the editor contents.
- The footer identifies the active panel and shows its applicable shortcuts.
- A disconnected user can open the panel but sees that a connection is required and cannot execute SQL.

## Query panel behavior

The right pane contains a multiline SQL editor above a result area, separated by one blank line. On every terminal resize, the upper section receives one quarter of the query panel's usable height and contains the `RAW QUERY` heading, editor, and separator; the result section receives the remaining three quarters. The editor has no background or separate active-line color and uses explicit light text for reliable contrast. The query panel has one column of horizontal padding and the editor has no line-number/prompt inset; the panel keeps its border and heading. Query state is separate from table-page state and includes the editor text, a loading flag, an execution error, and the latest execution result. Switching between panels does not clear either panel's state.

Submitting SQL creates a Bubble Tea command that executes asynchronously. The result area shows the existing spinner while it is running. A query error is rendered in the result area. Successful row-producing statements render with the existing data-grid styling, showing at most 100 rows. Results receive focus after execution and their viewport is navigated with Up/Down, j/k, PgUp/PgDown, or the mouse wheel; Tab switches focus between results and the editor. Successful statements with no returned rows render the PostgreSQL command status, such as `INSERT 0 3` or `CREATE TABLE`.

Only the newest query result for the active database session may update the panel. Results from an older submission or a replaced connection are ignored.

## Database boundary

`db.Database` gains a driver-neutral raw-execution operation and result type. The result type represents either rows (ordered column names and values) or a command status. The app depends only on this abstraction; it does not import PostgreSQL or pgx packages.

The PostgreSQL adapter executes the submitted SQL, writes it to the existing query log, wraps propagated errors, and returns the structured result. It supports arbitrary SQL statements, including reads, writes, and DDL. For row-producing statements, it collects no more than 100 rows for display. Command-only statements expose PostgreSQL's command status.

## Lifecycle and layout

Changing database connections resets raw-query state alongside the navigator and table-data state. Window resizing recomputes the query panel's editor and result dimensions. The query result grid reuses the existing bounded rendering and horizontal-column behavior where applicable.

## Tests

- Key handling opens query mode with `Ctrl+R` and returns to table mode with `Ctrl+T`.
- `Ctrl+P` dispatches asynchronous raw-query execution and leaves plain Enter available to the editor.
- Query results, command statuses, loading text, disconnected state, and execution errors render correctly.
- Connection changes and later submissions reject stale query-result messages.
- The PostgreSQL adapter returns columns and rows for row-producing SQL, command status for non-row SQL, logs submitted SQL, applies the 100-row display limit, and wraps execution errors.
