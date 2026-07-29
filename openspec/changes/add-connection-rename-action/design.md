## Context

Ctrl+G currently reads the selected table and immediately starts the asynchronous DDL request. Saved connection names are generated when a connection is created and are displayed in the Ctrl+L connections modal, but neither the connection editor nor another TUI flow can change `config.Connection.Name`.

The root model retains the active connection's normalized settings but loses the saved connection's config index when a row is selected. Existing deletion logic compares settings to decide whether the deleted entry is active. That comparison is ambiguous when two saved entries use identical settings.

The feature crosses key routing, modal rendering, connection identity, and config persistence. Repository rules also require file I/O to run in a `tea.Cmd`, not directly inside `Update`.

## Goals / Non-Goals

**Goals:**

- Preserve Ctrl+G as the entry point while adding an explicit action-selection step.
- Preserve the existing asynchronous DDL behavior.
- Rename only the active saved connection's JSON `name`.
- Keep the live database session and all connection settings unchanged.
- Target the correct config entry even when saved settings are duplicated.
- Make config-write success and failure observable in the modal.

**Non-Goals:**

- Renaming a database, schema, or table.
- Changing the JSON schema or adding persistent connection IDs.
- Editing engine, host, credentials, DSN, or database name through the rename flow.
- Automatically enforcing unique saved-connection names.
- Changing how config files are encoded or written atomically.

## Decisions

### Use one contextual actions modal for Ctrl+G

Introduce a dedicated modal owned by the root model. Its initial state lists explicit, context-bearing labels such as `View DDL for Album` and `Rename connection "Local"`. Arrow keys and j/k move selection, Enter chooses an action, and Esc closes the modal.

The DDL action is available only when `navigator.selectedTable()` succeeds. Rename remains available whenever the root model has a valid active saved-connection index, including databases with no tables and raw query mode. If neither action is available, Ctrl+G remains a no-op.

This keeps the requested shortcut and avoids silently applying an action to a different entity than its label describes. Adding a second shortcut was considered, but it would not satisfy the requested Ctrl+G action chooser.

### Reuse the existing DDL modal and command

Choosing DDL closes the actions modal, creates the existing `ddlModal`, increments `ddlRequest`, and starts `loadTableDDL` plus the spinner exactly as the current direct Ctrl+G route does. DDL loading, stale-result guards, scrolling, errors, and cancellation remain unchanged.

Duplicating DDL state inside the actions modal was rejected because it would fork established and tested behavior.

### Model rename as modal states with a focused text input

The actions modal owns states for action selection, name editing, saving, success, and failure. Entering rename initializes a Bubble Tea text input with the current saved name and focuses it. Esc from editing returns to action selection; Esc from selection closes the modal.

Submitting trims surrounding whitespace. Empty input produces an inline validation error. An unchanged trimmed name performs no write and returns to the selection state. During saving, further input is ignored. Success and failure states remain visible until Enter or Esc; failure permits returning to the input for correction or retry.

A success state is preferred over closing immediately because the main header currently displays the database name rather than the saved connection label.

### Track saved connections by config index

Add explicit root-model state for the active saved-connection index, initialized to `-1`. The selection message emitted by the connections modal carries both the selected index and connection. Successful connection establishment records the appropriate index:

- selecting an existing entry uses its supplied index;
- editing uses `editingConnection`;
- creating uses the newly appended index.

When a connection is deleted, an index below the active entry decrements the active index, deletion of the active index clears it and closes the session, and deletion above it leaves the index unchanged. Index bounds are checked before every rename.

Matching by `ConnectionSettings` was rejected because duplicate saved entries are valid today. Adding a UUID to each JSON entry would be more stable across external file edits but would expand the schema and migration scope unnecessarily; the running application already treats its loaded config slice as authoritative.

### Persist a cloned configuration through a tea.Cmd

On valid submission, clone `Config.Connections`, update only the target entry's `Name`, and pass that candidate configuration to a command that calls `Config.Save`. The command returns a typed completion message containing a rename request number, target index, candidate config, and error.

`Update` applies the candidate config only when the request still matches the open saving modal and the target remains the active index. On failure it retains the original in-memory config and displays `save connection name: <error>`. This mirrors the application's stale asynchronous message defenses while keeping filesystem I/O out of `Update`.

No `db.Database` method is called and the database session counter is not changed.

### Keep the config schema and name policy unchanged

The stored name is the trimmed submitted text. Duplicate non-empty names remain allowed because the current config format has no uniqueness requirement and existing generated names can already collide. `Engine`, `Settings`, and `Status` are copied without modification.

### Update visible help and tests

Rename the key-map meaning and footer label from `table DDL` to `actions`. Add focused modal tests, root update/lifecycle tests, and config-only assertions. Existing DDL tests move behind action selection and continue proving request freshness.

## Risks / Trade-offs

- [The actions menu combines a selected-table operation with an active-connection operation] → Use fully qualified labels that name each target and omit unavailable actions.
- [Config indexes are unstable if the file is edited externally while the TUI runs] → Continue treating the initially loaded/in-app-mutated config as authoritative and validate bounds before saving.
- [A successful rename is not visible in the current header] → Show a modal success state and verify the Ctrl+L list uses the updated in-memory config.
- [A config save can complete after the modal or connection context changes] → Guard completion messages with a request number, modal state, and active index.
- [Existing `Config.Save` writes are not atomic] → Keep the current persistence behavior in scope; a separate change can introduce atomic replacement for all config mutations.

## Migration Plan

No data migration is required because `Connection.Name` already exists. Deploying the change only adds UI and state-management behavior. Rollback leaves any user-renamed names as valid values in the unchanged JSON schema.

## Open Questions

None.
