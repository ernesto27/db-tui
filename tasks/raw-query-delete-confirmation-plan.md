# Implementation Plan: Raw-query DELETE confirmation

## Overview

Add a pre-execution confirmation modal for every raw SQL submission containing
a real `DELETE` statement, while preserving the existing asynchronous query
command, cancellation, and result handling.

## Architecture Decisions

- Keep SQL classification and modal state in `internal/app`; no database
  adapter or `db.Database` changes.
- Use a local, dependency-free SQL scanner that ignores comments and string
  literals, respects statement boundaries, and recognizes direct, CTE-prefixed,
  and later-in-batch `DELETE` statements.
- Store the original SQL with the modal. Confirmation sends it unchanged to the
  existing query-start path; cancellation discards only the pending request.
- Route this modal before normal key/paste handling and render it through the
  existing modal overlay.

## Dependency Graph

```text
SQL statement detection
        ↓
Pending raw-query DELETE modal
        ↓
Root-model confirmation routing
        ↓
Existing async query execution
        ↓
Regression tests
```

## Task List

### Task 1: Classify raw SQL containing DELETE

**Description:** Add a focused, dependency-free classifier for raw SQL batches.

**Acceptance criteria:**

- [x] Direct, CTE-prefixed, and later-in-batch `DELETE` statements are found.
- [x] Keyword text in comments, string literals, and quoted identifiers is ignored.
- [x] `SELECT`, `INSERT`, `UPDATE`, `TRUNCATE`, and DDL do not trigger detection.

**Verification:** Focused app-package tests after the final implementation pass.

**Dependencies:** None

**Files likely touched:**

- `internal/app/raw_query_delete_detection.go`
- `internal/app/raw_query_delete_detection_test.go`

**Estimated scope:** Small

### Task 2: Add confirmation state, modal, and execution handoff

**Description:** Intercept raw-query submission, display a modal for detected
deletes, and execute the saved SQL only after confirmation.

**Acceptance criteria:**

- [x] Submission of detected SQL opens a modal without starting a query command.
- [x] Confirm starts the existing query command with unchanged SQL.
- [x] Cancel and Escape close the modal without execution.
- [x] Modal-first routing blocks background editor input and renders correctly.

**Verification:** Covered by final automated verification after Task 3.

**Dependencies:** Task 1

**Files likely touched:**

- `internal/app/raw_query_delete_modal.go`
- `internal/app/model.go`
- `internal/app/update.go`
- `internal/app/view.go`

**Estimated scope:** Medium

### Checkpoint: Core behavior

- [x] Detection and confirmation tests pass.
- [x] Non-DELETE query submission remains unchanged.
- [ ] Review the TUI flow before final regression coverage.

### Task 3: Cover end-to-end model transitions

**Description:** Add regression tests for submission, confirm, cancel, input
isolation, and unchanged query execution.

**Acceptance criteria:**

- [x] Tests prove no query starts before confirmation.
- [x] Tests prove confirmation uses the exact original SQL.
- [x] Tests prove cancel/escape preserve editor contents and execute nothing.
- [x] Tests cover modal rendering and routing precedence.

**Verification:** Run `scripts/validate.sh` once after all implementation work.

**Dependencies:** Tasks 1–2

**Files likely touched:**

- `internal/app/query_panel_test.go`
- `internal/app/update_key_test.go`
- `internal/app/view_test.go`

**Estimated scope:** Medium

### Checkpoint: Complete

- [x] `scripts/validate.sh` passes.
- [ ] User manually verifies the confirm and cancel flows in the TUI.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| SQL dialect syntax causes a false negative | High | Test lexical states and statement boundaries; no execution occurs until detection completes. |
| A keyword inside SQL text triggers the modal | Medium | Tokenize comments, quoted identifiers, and strings rather than searching text. |
| Modal bypasses existing query safeguards | High | Reuse the existing query-start command after confirmation. |
| Feature overwrites unrelated planning work | Medium | Use feature-specific task files. |

## Open Questions

None.
