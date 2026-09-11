# Raw-query DELETE confirmation

- [x] Task 1: Classify raw SQL containing DELETE
  - Acceptance: recognize direct, CTE-prefixed, and later-in-batch deletes while ignoring comments and quoted text.
  - Verify: covered by final automated verification.

- [x] Task 2: Add confirmation state, modal, and execution handoff
  - Acceptance: confirmation executes unchanged SQL; cancel and escape execute nothing.
  - Verify: covered by final automated verification.

## Checkpoint: Core behavior

- [x] Detection and confirmation behavior is covered.
- [x] Non-DELETE query submission remains unchanged.

- [x] Task 3: Cover end-to-end model transitions
  - Acceptance: test submission, confirm, cancel, editor preservation, and modal-first routing.
  - Verify: run `scripts/validate.sh` after all implementation work.

## Checkpoint: Complete

- [x] `scripts/validate.sh` passes.
- [ ] User manually verifies the TUI flow.
