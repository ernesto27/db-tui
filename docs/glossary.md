# Glossary

## Object selector

The read-only category picker opened with `Ctrl+O`. It changes the object type
displayed in the left navigator between tables, views, materialized views when
supported, and stored functions when supported. Selecting a category does not
activate an item or replace the right panel.

## Function list

The filtered left-navigator listing of stored functions for the selected
object type. It shows only function names, while retaining the selected
function's arguments, return type, language, and definition for inspection.

## Active function

The stored function most recently activated with `Enter` while the Functions
object type is displayed. It owns the right panel's read-only, scrollable
function-detail view until a row-browsable relation is activated.

## Connection script library

The local collection of saved SQL scripts for one configured connection. Its
directory uses the connection's current display name. When that connection is
renamed successfully, db-tui moves the directory to the new name so the same
scripts remain available. The raw-query panel can access only the library of
its active database connection. Executing non-empty SQL creates a script even
if the database reports an execution failure; empty editor content creates no
script. `Ctrl+H` opens its script library.

## SQL script preview

The truncated first non-empty SQL line shown for a saved script in the active
connection's script-library modal. It lets users identify scripts whose
generated filenames are not meaningful.

## Saved SQL filename

The generated local filename for a saved SQL script. It retains the `.txt`
extension.

## SQL script ordering

The script-library modal lists saved SQL scripts with the most recently
created or updated script first.

## SQL-scripts shortcut scope

`Ctrl+H` is available only while the raw-query panel is active. It has no
effect in other panels.

## Empty SQL script library

The normal script-modal state for an active connection with no saved scripts.
It displays “No saved scripts”; absence of the library directory is not an
error.

## Load-only SQL-scripts modal

The initial script-library modal supports selection with `Up`/`Down`, loading
with `Enter`, and closing with `Esc`. It intentionally excludes script
deletion, renaming, and manual saving.

## Loaded SQL script

A saved script selected from the active connection's script-library modal.
Loading it replaces all raw-query editor content; the user can then edit the
SQL before executing it. Its generated filename is retained, so executing
non-empty SQL with `Ctrl+P` replaces the same saved file.

## SQL script save failure

A local filesystem error while creating or updating a saved SQL script. It is
shown to the user but does not prevent the corresponding SQL from executing
against the active database. The raw-query panel displays it as a non-modal
warning alongside the query outcome.

## Script-library rename collision

The condition where the target directory for a renamed connection already
exists. db-tui rejects the rename and leaves the connection configuration and
all script directories unchanged.

## Configured page size

The application-wide, persisted number of rows requested for each row page.
It is any positive whole number and has no application-defined upper bound.
Saving a new configured page size does not reload the currently displayed page;
the next row-page request uses the new size. When moving forward after a size
change, the next page starts immediately after the last row in the currently
displayed page so that the transition neither skips nor repeats rows.

## Data-panel refresh

An explicit `r` action available only when the table-data panel has focus. It
reissues the active relation's row query from offset zero with no selected row,
so a successful refresh displays its first configured page. While a row load
is in progress, with no active relation, or outside data-panel focus, the
action has no effect. Refresh uses normal row-load feedback and displays the
normal row-load error if it fails.

## Data-grid text selection

An application-owned, character-range selection within one rendered data-grid
cell. Its selectable surface comprises column-header cells and visible
result-data cells; it excludes panel chrome, the navigator, query panel, and
overlays. A drag is clamped to the row and column where it starts. Releasing a
primary-button drag copies the selected text as plain text, retaining selected
spacing but omitting ANSI styling sequences.
A click without a drag remains the ordinary data-row selection interaction and
does not copy.

## Row deletion confirmation

A modal, destructive-action checkpoint opened with `d` for a selected row in
an active base table while the data panel has focus. It names the table without
displaying primary-key values. `Enter` confirms, `Esc` cancels or returns, and
the modal shows deletion progress, success, or failure before dismissal.

## Row-browsable relation

A base table, ordinary database view, or materialized view whose rows db-tui can display in the pageable data panel. All row-browsable relation types use the same explicit activation behavior in the navigator.

## Highlighted relation

The row-browsable relation under the navigator cursor. Keyboard navigation, mouse navigation, filtering, and section changes may change the highlighted relation without querying its rows.

## Active relation

The row-browsable relation most recently activated with `Enter` or a double-click. Activating a relation makes it the target of row-page requests and the owner of the data panel. The active relation remains unchanged while the user highlights other relations.

Activation transfers data-panel ownership immediately: the prior relation's page is cleared before the new relation's first page finishes loading. A loading state or load error therefore belongs to the newly active relation.

## Stale row result

An asynchronous row-page result whose connection session, request ID, relation identity, or page offset no longer matches the active row request. Changing only the navigator highlight does not make a result stale; activating a different relation does.

## Relation identity

The combination of a relation's type and name. The active relation retains this identity independently of the navigator highlight so row paging, panel labeling, and asynchronous-result validation continue to refer to the same row source.

## Client-observed elapsed time

The duration measured by the raw-query command from immediately before `db.Database.Execute` until it returns. It includes driver work, network transfer where applicable, and bounded result decoding; it is not a database-server-only execution metric. The raw-query UI labels this value `Execution time`.

## JSON table export

A complete-data export of the selected table as one JSON document. The selected table name is the sole top-level key and its value is an array of objects keyed by database column name. SQL `NULL` is encoded as JSON `null`.

## Export format picker

The `Ctrl+E` table-data overlay used to select CSV or JSON before confirmation. CSV is selected initially; `Up`/`Down` or `j`/`k` change the selection, `Enter` advances, and `Esc` cancels.

## DDL modal

A centered, read-only db-tui overlay that displays the selected table's fresh structural SQL script. It is opened with `Ctrl+G` and does not change the underlying panel state.

## Column inspection

A cross-engine, read-only table screen for the currently selected table. It displays each column's name, ordinal position, data type, identity status, collation, nullability, default expression, and comment. Fields that an engine does not expose, such as SQLite collation and comments, render blank.

## Index inspection

A cross-engine, read-only table screen for the currently selected table. It displays the index name, indexed column, table name, and access method. It includes standalone indexes plus indexes created for primary-key and unique constraints. A multi-column index is represented by one row for each indexed column. PostgreSQL reports its access method, MySQL reports its index type, and SQLite reports `BTREE` for ordinary indexes.

## Structural DDL

The executable SQL needed to define a supported table's columns, defaults, identity/generated clauses, collations, inline constraints, and non-constraint indexes. PostgreSQL structural DDL intentionally excludes comments, ownership, grants, triggers, and separate sequence declarations.

## Table DDL

The driver-neutral string returned by `db.Database.TableDDL` for one selected table. MySQL supplies its server-generated `SHOW CREATE TABLE` result; PostgreSQL reconstructs the supported structural script from catalogs.

## Constraint-owned index

An index PostgreSQL creates to enforce a primary-key, unique, or exclusion constraint. It appears through that constraint's DDL and must not be emitted again as a standalone index.

## Stale DDL result

An asynchronous DDL command result whose connection session, request ID, selected table, or modal state no longer matches the active request. db-tui ignores stale results.

## Supported PostgreSQL table

For the first DDL-modal release, an ordinary table in the `public` schema that is neither partitioned nor inherited. Other relation structures return an explicit unsupported-structure error.

## Database view

A named, query-backed relation. The view browser lists ordinary PostgreSQL views in the `public` schema, views in the active MySQL database, and non-internal SQLite views. Views appear separately from tables and support row browsing only.

## Materialized view

A stored query result in PostgreSQL or Oracle that persists until it is refreshed. db-tui lists materialized views in their own navigator section. They support pageable row browsing only; refresh and all table-specific actions are intentionally excluded.

## Oracle dump placeholder

The initial Oracle engine's implementation of the `db.Database` dump operation.
It returns an explicit unsupported-operation error because Oracle Data Pump
requires external tooling and database-specific privileges that db-tui does not
provision.
