## 1. Active Saved-Connection Identity

- [x] 1.1 Add failing app tests for recording the selected, edited, and newly created connection indexes, including duplicate connection settings.
- [x] 1.2 Add failing app tests for preserving, decrementing, or clearing the active index when saved connections are deleted.
- [x] 1.3 Extend connection-selection messages and root model state to track the pending and active saved-connection indexes.
- [x] 1.4 Refactor deletion handling to use the tracked index instead of comparing `ConnectionSettings`, and make the identity tests pass.

## 2. Ctrl+G Contextual Actions

- [x] 2.1 Add failing modal tests for available actions, explicit target labels, keyboard navigation, action selection, and Esc behavior.
- [x] 2.2 Implement the contextual actions modal and integrate it into root modal routing and overlay rendering.
- [x] 2.3 Replace direct Ctrl+G DDL routing with action-modal creation, including rename-only behavior when no table is selected.
- [x] 2.4 Route the DDL selection through the existing `ddlModal` and `loadTableDDL` lifecycle, and update existing Ctrl+G tests to cover the additional selection step.
- [x] 2.5 Change key-map and footer help text from table DDL to contextual actions and update view tests.

## 3. Config-Only Connection Rename

- [x] 3.1 Add failing tests for the prefilled rename input, whitespace trimming, empty-name validation, unchanged-name no-op, cancellation, and retry after failure.
- [x] 3.2 Add failing command and lifecycle tests proving successful rename changes only `Connection.Name`, save failure preserves in-memory state, stale completions are ignored, and the database session is not closed or replaced.
- [x] 3.3 Implement cloned-config rename persistence as a `tea.Cmd` with typed, request-guarded completion messages.
- [x] 3.4 Implement rename editing, saving, success, and failure modal states and make the rename behavior tests pass.
- [x] 3.5 Verify reopening the Ctrl+L connections modal displays the updated name while engine, settings, status, table selection, and active database remain unchanged.

## 4. Verification

- [x] 4.1 Run `gofmt` on all changed Go files.
- [x] 4.2 Run focused `internal/app` tests and `go test ./...`.
- [x] 4.3 Run `scripts/validate.sh` and resolve formatting, vet, test, or race-test failures.
