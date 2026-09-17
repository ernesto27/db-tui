# Implementation Plan: Sidebar Database Reconnect

## Overview

Add a navigator-scoped `r` shortcut that asynchronously reconnects with the
active saved settings. Successful attempts atomically replace the session and
reset application data; failures retain the current session and display a
sanitized error.

## Architecture Decisions

- Reuse the existing `ConnectFunc`, `connectionAttempt`, `connectionFinishedMsg`,
  session counter, and object-loading commands; do not alter `internal/db`.
- Extract the existing successful-connection adoption/reset path so modal
  connections and sidebar reconnects use one lifecycle path.
- Preserve the old database until a matching reconnect succeeds. Close stale
  successful results and the replaced database exactly once.
- Reuse the existing `r` binding contextually: it continues refreshing table
  data when the data panel owns focus, and reconnects only when the navigator
  owns focus.
- Keep reconnect errors in app state and render them through the existing
  navigator/status view using sanitized text.

## Dependency Graph

```text
connection attempt/result handling
        ↓
session adoption and error state
        ↓
navigator-scoped r routing + shortcut help
        ↓
model and key-routing regression tests
```

## Task List

### Task 1: Reconnect lifecycle

Implement app-owned reconnect state and route connection results outside the
connection modal. Share the established session-adoption/reset behavior,
preserving the old session on error and rejecting stale results.

### Task 2: Navigator input and feedback

Add the navigator-only `r` route, contextual help text, reconnect progress,
and error rendering without changing data-panel table refresh behavior.

### Task 3: Regression coverage

Add focused model tests for focus scoping, successful swap/reset, failed
reconnect preservation, and stale-attempt cleanup.

## Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| Stale success leaks a database handle | Close any result whose attempt no longer matches. |
| Reconnect changes existing `r` refresh behavior | Route by focus and test the data-panel path remains unchanged. |
| Duplicate session-reset logic diverges | Extract one shared success-adoption helper. |
| Failed reconnect destroys usable UI | Do not mutate active session/navigator until success. |

## Verification Checkpoint

Run `scripts/validate.sh` once after all tasks are implemented. The user
performs the manual TUI check.

## Detailed Task Checklist

### Task 1: Add Identity-Safe Reconnect Lifecycle

**Description:** Add app state and result handling for an asynchronous sidebar
reconnect, reusing the existing connection command and sharing the established
successful-session adoption/reset path.

**Acceptance criteria:**

- [ ] A reconnect attempt uses saved settings and cannot replace a newer attempt.
- [ ] A matching success adopts the new session, closes the previous database,
      resets the navigator/data/query state, and starts object loading.
- [ ] A failure preserves the existing session; stale successful results close
      their database handle.

**Verification:**

- [ ] Covered by final focused app-model tests.
- [ ] Included in the final `scripts/validate.sh` run.

**Dependencies:** None

**Files likely touched:**

- `internal/app/model.go`
- `internal/app/update.go`
- `internal/app/update_lifecycle_test.go`

**Estimated scope:** Medium

### Task 2: Route Navigator `r` and Render Reconnect Feedback

**Description:** Make `r` start reconnect only while navigator focus owns
input, retain the existing data-panel table-refresh behavior, and expose
reconnect progress/errors in navigator status and shortcut help.

**Acceptance criteria:**

- [ ] Navigator-focused `r` starts reconnect only when a database is active.
- [ ] Data-focused `r` retains its current table-refresh behavior.
- [ ] Navigator status/help communicates reconnecting and sanitized failure
      feedback.

**Verification:**

- [ ] Covered by final keyboard-routing and view tests.
- [ ] Included in the final `scripts/validate.sh` run.

**Dependencies:** Task 1

**Files likely touched:**

- `internal/app/keymap.go`
- `internal/app/update.go`
- `internal/app/view.go`
- `internal/app/navigator.go`
- `internal/app/update_key_test.go`

**Estimated scope:** Medium

### Task 3: Add Reconnect Regression Coverage

**Description:** Add outcome-based tests for focus-scoped input, connection
replacement, failure preservation, stale results, reset behavior, and visible
feedback.

**Acceptance criteria:**

- [ ] Tests cover success, failure, stale-success cleanup, and navigator reset.
- [ ] Tests prove `r` differs correctly between navigator and data focus.
- [ ] Tests cover reconnect status/error rendering or its status-model input.

**Verification:**

- [ ] Run `scripts/validate.sh` once after implementation.
- [ ] User manually verifies sidebar `r` in the TUI.

**Dependencies:** Tasks 1–2

**Files likely touched:**

- `internal/app/update_lifecycle_test.go`
- `internal/app/update_key_test.go`
- `internal/app/view_test.go`

**Estimated scope:** Small

### Checkpoint: Complete

- [ ] All task acceptance criteria are met.
- [ ] `scripts/validate.sh` passes once after the final code change.
- [ ] User manually verifies the sidebar reconnect flow.
