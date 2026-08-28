# Implementation Plan: Connection Environment Color

## Overview

Add a persistent, opt-in environment marker to saved database connections. From Ctrl+G Actions, users choose no environment, testing, or production. The active connection renders the entire header green with `TESTING`, red with `PRODUCTION`, or exactly as it does today when unclassified.

## Architecture Decisions

- Store an optional `environment` value on `config.Connection`; valid application values are `testing` and `production`, and blank means unclassified.
- Preserve the existing `Status` field for compatibility; it cannot express the required three states.
- Model the config save as a `tea.Cmd` returning a typed message, following the existing asynchronous rename flow so `Update` performs no I/O.
- Expose the action only when `activeConnectionIndex` identifies a saved connection; a table-only Ctrl+G menu cannot change connection metadata.
- Derive header state from the persisted active connection after a successful save; do not optimistically recolor on a failed save.

## Dependency Graph

```text
config.Connection.environment
          |
          v
Ctrl+G picker + save command/result
          |
          v
active connection header styling
```

## Task List

### Phase 1: Persistence Foundation

#### Task 1: Persist an Optional Connection Environment

**Description:** Add an optional environment value to saved connection configuration. Ensure old configuration files remain unclassified and the JSON representation omits the field when unset.

**Acceptance criteria:**

- [ ] Existing configuration without `environment` loads successfully with no environment.
- [ ] `testing` and `production` round-trip through config load and save.
- [ ] Saving an unclassified connection omits the empty environment field.

**Verification:** Covered by the final `scripts/validate.sh` run after Task 3.

**Dependencies:** None.

**Files likely touched:**

- `internal/config/config.go`
- `internal/config/config_test.go`

**Estimated scope:** Small (2 files).

### Phase 2: Environment Selection

#### Task 2: Add the Ctrl+G Environment Picker and Save Flow

**Description:** Add `Set connection environment…` to the Actions modal for an active saved connection. Implement the no-color/testing/production picker, asynchronous persistence, and success or failure feedback using typed Bubble Tea messages.

**Acceptance criteria:**

- [ ] The action is unavailable when there is no active saved connection.
- [ ] The picker supports all three values and Escape returns to the Actions menu.
- [ ] A successful save updates only the active saved connection; a failed save retains the previous value and shows an error.

**Verification:** Covered by the final `scripts/validate.sh` run after Task 3.

**Dependencies:** Task 1.

**Files likely touched:**

- `internal/app/actions_modal.go`
- `internal/app/commands.go`
- `internal/app/update.go`
- `internal/app/actions_modal_test.go`

**Estimated scope:** Medium (4 files).

### Phase 3: Active Header Signal

#### Task 3: Render the Active Environment in the Header

**Description:** Derive the environment of the active saved connection and render the full header in its semantic color with a textual environment label. Keep the current style for unclassified or unsaved connections.

**Acceptance criteria:**

- [ ] Testing renders the full header green and includes `TESTING`.
- [ ] Production renders the full header red and includes `PRODUCTION`.
- [ ] Unclassified and unsaved connections preserve the current header styling and content.

**Verification:** Covered by the final `scripts/validate.sh` run after Task 3.

**Dependencies:** Tasks 1-2.

**Files likely touched:**

- `internal/app/colors.go`
- `internal/app/view.go`
- `internal/app/view_test.go`

**Estimated scope:** Small (3 files).

### Checkpoint: Complete

- [ ] Run `scripts/validate.sh` once after all three tasks are implemented.
- [ ] User manually verifies Ctrl+G selection, immediate header change, clearing the environment, and persistence after reconnecting.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| A stale asynchronous save overwrites a newer selection | High | Correlate the save result with the active connection and request, as the rename flow does. |
| A config-save failure produces a misleading header | High | Update in-memory config and header only after the save result succeeds. |
| Color alone is ambiguous or inaccessible | Medium | Always render the `TESTING` or `PRODUCTION` text label in the header. |
| The action appears for an unsaved connection | Medium | Gate it on a valid `activeConnectionIndex`. |

## Open Questions

- None for the approved MVP. Showing the environment in the saved-connections list is deliberately deferred.
