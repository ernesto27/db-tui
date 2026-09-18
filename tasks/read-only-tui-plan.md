# Implementation Plan: Session Read-Only Mode

## Overview

Add an in-memory read-only mode to the connected TUI session. The mode is
toggled with `Alt+R`, visible in the header, reset on connection adoption, and
passed to the existing query-execution contract. SQL Server receives a focused
client-side raw-query guard because its adapter does not enforce that contract.

## Architecture Decisions

- `internal/app.Model` owns the session flag; no configuration or adapter state
  is changed.
- SQL Server validation reuses the raw-query tokenizer, which already ignores
  comments, literals, and quoted identifiers. No parser dependency is added.
- The SQL Server guard runs before the existing raw-DELETE confirmation so an
  active read-only mode never opens a destructive confirmation for a rejected
  statement.
- `adoptConnection` resets the flag so reconnects and connection switches are
  isolated sessions.

## Dependency Graph

```text
session state + Alt+R ─┬─> execution-mode forwarding ─> backend enforcement
                       └─> header/help rendering
raw-query tokenizer ─────> SQL Server guard ──────────> query start path
```

## Task List

### Task 1: Add session state and toggle lifecycle

**Description:** Add the read-only flag, global `Alt+R` binding, and reset on
connection adoption.

**Acceptance criteria:**

- [ ] `Alt+R` toggles only while connected.
- [ ] Adoption of any new database resets the flag.
- [ ] The state is not part of saved connection configuration.

**Verification:** Covered by the final `scripts/validate.sh` run.

**Dependencies:** None

**Files likely touched:** `internal/app/model.go`, `internal/app/keymap.go`,
`internal/app/update.go`, `internal/app/update_test.go`

**Estimated scope:** Medium

### Task 2: Add SQL Server read-only classification

**Description:** Reuse the existing lexical tokenizer to reject write-side,
transactional, locking, and session-control SQL Server statements.

**Acceptance criteria:**

- [ ] Comments, literals, and quoted identifiers do not cause rejection.
- [ ] Supported write-side keywords are rejected case-insensitively.
- [ ] Read-only SQL remains eligible for execution.

**Verification:** Covered by the final `scripts/validate.sh` run.

**Dependencies:** None

**Files likely touched:** `internal/app/raw_query_delete_detection.go`,
`internal/app/raw_query_delete_detection_test.go`

**Estimated scope:** Small

### Task 3: Integrate mode and SQL Server guard with query execution

**Description:** Pass the active mode to `Database.Execute`, capture it in the
fake, and reject guarded SQL before either confirmation or execution.

**Acceptance criteria:**

- [ ] Default and read-only executions use their matching mode.
- [ ] A rejected SQL Server query never calls `Execute` or saves a script.
- [ ] Other engines retain backend enforcement.

**Verification:** Covered by the final `scripts/validate.sh` run.

**Dependencies:** Tasks 1-2

**Files likely touched:** `internal/app/commands.go`, `internal/app/update.go`,
`internal/app/test_helpers_test.go`, `internal/app/commands_test.go`,
`internal/app/query_panel_test.go`

**Estimated scope:** Medium

### Task 4: Render and document the active mode

**Description:** Add the header indicator and shortcut help while retaining
the existing UI text and semantic-color conventions.

**Acceptance criteria:**

- [ ] Connected read-only sessions visibly render `READ ONLY`.
- [ ] Shortcut help documents `Alt+R`.
- [ ] Existing header environment and engine details remain intact.

**Verification:** Covered by the final `scripts/validate.sh` run.

**Dependencies:** Task 1

**Files likely touched:** `internal/app/view.go`,
`internal/app/shortcuts_modal.go`, `internal/app/view_test.go`

**Estimated scope:** Small

### Checkpoint: Complete

- [ ] Run `scripts/validate.sh` once after all code and tests are complete.
- [ ] User manually toggles `Alt+R`, verifies the header, and attempts a raw
  write query on SQL Server and a supported adapter.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| SQL keywords inside text create false rejections | Medium | Reuse the existing lexical tokenizer and cover comments/literals/quoted identifiers. |
| A reconnection carries state to a different database | High | Reset read-only in the single `adoptConnection` lifecycle owner. |
| SQL Server reaches its non-enforcing adapter | High | Validate before DELETE confirmation and before `Execute`; assert zero fake calls. |

## Open Questions

None.
