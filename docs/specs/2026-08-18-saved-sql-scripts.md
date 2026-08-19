# Saved SQL scripts

## Problem Statement

Raw queries are currently ephemeral. A user cannot return to a query that they
executed, including one that failed, nor can they revise and re-run a previous
query as part of the same connection-specific workflow. The user also needs
saved scripts to remain associated with a renamed connection rather than being
orphaned in local storage.

## Solution

The raw-query panel will keep a local, connection script library. Executing
non-empty SQL with `Ctrl+P` automatically creates a generated `.txt` script
for a new editor query, regardless of the database execution outcome. If the
editor contains a loaded SQL script, the same key updates that script with the
current editor content.

`Ctrl+H`, while the raw-query panel is active, opens a load-only modal for the
active connection's script library. The modal lists scripts by a truncated
first non-empty SQL line, newest created or updated first. `Enter` replaces the
raw-query editor with the selected script and leaves the user free to edit it;
`Esc` closes the modal.

The library follows a connection rename. A target-directory collision rejects
the rename without changing connection configuration or either library. These
behaviors comply with ADR 0015 and the glossary definitions for connection
script libraries, loaded SQL scripts, previews, and script-save failures.

## User Stories

1. As a database user, I want each non-empty raw query I execute to be saved, so that I can revisit useful and failed SQL alike.
2. As a database user, I want a failed database query to be saved too, so that I can correct and retry it later.
3. As a database user, I want whitespace-only editor content to do nothing on `Ctrl+P`, so that my history contains actual SQL only.
4. As a database user, I want new executions to receive generated `.txt` filenames, so that the client can save them without asking me to name each query.
5. As a database user, I want `Ctrl+H` to open my active connection's script library from the raw-query panel, so that I can quickly recover prior work.
6. As a database user, I want the modal to contain only scripts for the active database connection, so that I cannot accidentally load history from another connection.
7. As a database user, I want saved scripts identified by a readable truncated SQL preview, so that generated filenames do not prevent me from finding the right query.
8. As a database user, I want most recently created or updated scripts first, so that my current work is immediately accessible.
9. As a database user, I want an empty script library to say “No saved scripts”, so that a new connection does not look broken.
10. As a database user, I want `Enter` to load a selected script into the editor, so that I can edit it before executing it again.
11. As a database user, I want loading to replace all editor content, so that the selected script becomes the complete query I am revising.
12. As a database user, I want re-executing an edited loaded script to update that saved script, so that revisions do not create duplicate history entries.
13. As a database user, I want a script-save failure to leave database execution intact, so that a local filesystem issue does not stop my work.
14. As a database user, I want a visible non-modal warning when a script cannot be saved, so that I know my history was not updated even if the query ran.
15. As a database user, I want scripts to remain available after renaming a connection, so that the connection script library continues to represent the same database profile.
16. As a database user, I want a rename that would collide with another script library to be rejected, so that two connections' histories are never silently mixed.
17. As a database user, I want the initial script modal to be load-only, so that its keyboard behavior stays simple and does not add unrequested script-management actions.
18. As a database user, I want `Ctrl+H` outside the raw-query panel to have no effect, so that the shortcut stays scoped to its relevant workflow.

## Implementation Decisions

- Extend the application key map with `Ctrl+H` for SQL-script history. It is recognized only when the raw-query panel is active and a connection is active.
- Add application-owned script-modal state that contains the active connection identity, loaded script entries, a selection index, and any load error. It is a standard modal that blocks underlying key handling while open.
- The modal supports only `Up`/`Down`, `Enter`, and `Esc`. Selection is clamped; `Enter` is inert for an empty library.
- A script entry has a generated `.txt` filename and its SQL content. The UI derives the label from the first non-empty SQL line and truncates it to the available modal width.
- The local script-store boundary owns path derivation, generated-file creation, replacement of a known file, listing, preview-ready content, newest-first ordering, and connection-directory rename behavior. It must not permit path traversal through connection names or generated filenames.
- Missing script directories represent an empty connection script library, not an error. Other filesystem failures are returned with context.
- On `Ctrl+P`, trim only for the purpose of deciding whether the editor is empty; preserve the editor's original non-empty content when saving and executing.
- A new, non-empty query creates a generated script. A loaded script retains its filename in query-panel state; a non-empty re-execution replaces that exact saved file. Saving is initiated independently of the database execute command so a local write failure cannot suppress execution.
- Query state records the most recent script-save warning separately from the database query result and error. A save failure renders in the raw-query panel without replacing database success, result rows, command status, elapsed time, or database error feedback.
- The active connection's display name determines its script-library directory. Connection rename coordinates filesystem and configuration changes so the directory moves to the new name on success. If the destination exists, reject the rename before changing either side. If the source directory is absent, the rename may proceed with no directory move.
- Rename coordination must preserve the observable all-or-nothing outcome as far as the local filesystem and configuration persistence allow: report failures clearly and avoid committing a configuration name that points away from a successfully retained library.

## Testing Decisions

- Test observable behavior rather than private fields: use the application model's update/message seam for shortcut scope, opening and closing the modal, selecting and loading scripts, editor replacement, execution initiation, and save-warning rendering.
- Use the existing raw-query execution test seam and fake database to prove that executing non-empty SQL still occurs when the script-store command fails, and that whitespace-only SQL neither executes nor saves.
- Test the script-store boundary with a temporary configuration directory for generated `.txt` creation, updating a loaded file, empty-library behavior, first-non-empty-line previews, descending modification-time ordering, invalid path components, and contextual filesystem failures.
- Test connection rename through the existing rename command/message seam: an existing source library moves to the new name, an absent source library is harmless, and a pre-existing destination causes a visible rejection with configuration and libraries unchanged.
- Follow existing table-driven Go tests and the project's application modal/key-handling tests as prior art. Tests must avoid a real database and remote filesystem dependencies.
- Keep the main feature behavior at the application model seam; introduce only the script-store boundary needed to make local filesystem behavior deterministic and independently testable.

## Out of Scope

- Cross-connection script browsing, loading, or automatic connection switching.
- Script deletion, renaming, manual save commands, arbitrary user-chosen script filenames, or changing the `.txt` extension.
- Confirmation before replacing editor content with a loaded script.
- Retrospective migration, merging, or deduplication of script libraries.
- Changing database execution semantics, query result limits, elapsed-time reporting, or query-log behavior.

## Further Notes

- The raw-query panel owns one active database connection at a time; the connection script library follows that ownership boundary.
- A script-save warning is independent from the database query outcome. A query can succeed while saving fails, or fail while saving succeeds.
- Re-executing a loaded script updates its modification time, which places it at the top of the next script-modal listing.
- The specification deliberately keeps script management small for its first release; future actions should be separately specified.
