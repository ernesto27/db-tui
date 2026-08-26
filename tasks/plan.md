# Implementation Plan: Searchable Connection Picker

## Overview

Add a visible, auto-focused search input to the saved-connections modal. Typing filters connection names immediately. Down transfers focus to results; Up/Down selects a filtered result; Enter connects using its original config index.

## Architecture Decisions

- Keep filtering local to `connectionsModal`; no config or database changes.
- Use Bubble Tea's existing `textinput.Model`, consistent with other app inputs.
- Represent visible rows with original indexes so selection, edit, and deletion continue to target the correct saved connection.
- Match connection display names case-insensitively.

## Task List

### Phase 1: Modal Search Behavior

#### Task 1: Add Search State and Filtered-Result Mapping

**Description:** Add the focused search input, result-list focus state, and filtered connection rows to `connectionsModal`.

**Acceptance criteria:**

- [ ] A visible `Search connections...` field appears at the top of the modal.
- [ ] The field is focused when the modal opens.
- [ ] Typing and Backspace filter names case-insensitively.
- [ ] Each displayed result retains its source index.

**Verification:** Covered by the final automated verification run after Task 3.

**Dependencies:** None

**Files likely touched:**

- `internal/app/connections_modal.go`

**Estimated scope:** Small

#### Task 2: Route Picker Keyboard Actions Through Filtered Results

**Description:** Make Down transfer focus from search to results; navigate filtered rows with Up/Down; preserve Enter, edit, delete, and Esc behavior.

**Acceptance criteria:**

- [ ] Down enters the results list without changing the query.
- [ ] Up/Down cannot select outside filtered bounds.
- [ ] Enter, Ctrl+E, and d act on the correct original connection.
- [ ] Empty-result states cannot connect, edit, or delete.

**Verification:** Covered by the final automated verification run after Task 3.

**Dependencies:** Task 1

**Files likely touched:**

- `internal/app/connections_modal.go`

**Estimated scope:** Small

### Phase 2: Regression Coverage

#### Task 3: Test Search, Focus, and Source-Index Behavior

**Description:** Cover the new modal behavior at its lowest practical layer.

**Acceptance criteria:**

- [ ] New modal starts with a focused empty search input.
- [ ] Filtering narrows results and handles no matches.
- [ ] Keyboard focus moves from input to results as specified.
- [ ] Selecting a filtered connection emits its original index.

**Verification:** Run `scripts/validate.sh` once after implementation.

**Dependencies:** Tasks 1-2

**Files likely touched:**

- `internal/app/connections_modal_test.go`

**Estimated scope:** Small

### Checkpoint: Complete

- [ ] Run `scripts/validate.sh` once after implementation.
- [ ] User manually confirms the picker flow in the TUI.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Filtered position differs from configuration index | High | Carry the original index in every visible result and test action messages. |
| Input consumes list-navigation keys | Medium | Handle focus-transition and list-navigation keys before forwarding messages to the input. |
| No matches leave a stale selection | Medium | Reset or clamp selection whenever the filter changes and reject actions with no visible row. |

## Open Questions

- None. The confirmed scope is limited to name-based filtering and keyboard navigation in the existing picker.
