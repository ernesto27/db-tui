# ADR 0015: Saved SQL scripts follow connection renames

## Status

Accepted on 2026-08-18.

## Context

Saved SQL scripts are organized in a directory associated with a configured
connection. Connection names can be changed in the TUI. Leaving the existing
directory under the old name would make a renamed connection appear to have no
scripts and would orphan the user's saved work.

## Decision

A connection's SQL-script directory is named from its current display name.
When a connection rename succeeds, db-tui must move that directory from the
old name to the new name as part of the rename operation. The scripts therefore
remain available under the renamed connection.

The raw-query panel belongs to the currently active database connection. Its
scripts action reads only that connection's directory; it does not provide a
cross-connection picker, switch connections, or execute a script against a
connection other than the active one.

Executing a non-empty raw query with `Ctrl+P` creates a new saved script before
or regardless of the database execution outcome. An empty or whitespace-only
editor is a no-op and creates no script.

`Ctrl+H` opens the active connection's SQL-scripts modal.

The modal identifies each script by a truncated first non-empty line of its SQL
content. Its generated filename remains the internal identity used to load and
replace the saved file.

Pressing `Enter` loads the selected script by replacing the entire raw-query
editor. The user can edit the loaded SQL before executing it. A subsequent
non-empty `Ctrl+P` replaces that script's existing file instead of creating a
new one.

A local script-save failure does not prevent the query from executing. The UI
must surface that save failure independently from the database execution result
as a non-modal warning in the raw-query panel.

Generated saved-script filenames retain the existing `.txt` extension.

The modal sorts scripts by most recently created or updated first. Updating a
loaded script therefore returns it to the top of the list.

`Ctrl+H` opens the modal only while the raw-query panel is active; it is a no-op
in other panels.

An active connection with no script directory or saved scripts opens the modal
in an empty state that says “No saved scripts”; a missing directory is not an
error.

The initial modal is load-only: `Up`/`Down` select, `Enter` loads, and `Esc`
closes. It does not provide script deletion, renaming, or manual saving.

## Consequences

- Script ownership remains visible and understandable in the local directory
  structure.
- Renaming a connection now includes local SQL-script storage and must report
any storage failure rather than silently losing access to scripts.

If the target script directory already exists, the connection rename is
rejected. Neither configuration nor either script directory changes, preventing
two connections' histories from being merged.
- The rename workflow needs defined handling for absent source directories and
  a destination directory that already exists.
