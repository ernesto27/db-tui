# Implementation Plan: PostgreSQL Schema Object Picker

## Overview

Extend the PostgreSQL object picker so it lists each non-empty, schema-qualified object category visible to the connected role. Selecting an entry such as `reporting — Views` replaces the left navigator with only that schema's views. The existing object-picker behavior remains unchanged for engines without schema browsing.

## Architecture Decisions

- Represent a modal row as a database-neutral schema/object-type pair. PostgreSQL returns real pairs; the other database adapters implement the required interface method as an empty placeholder.
- Keep the modal one level deep: list non-empty pairs directly instead of schemas with nested submenus.
- Load the selected category asynchronously on Enter and bind results to the active session and selection, so stale results cannot replace a newer sidebar selection.
- Preserve schema identity in navigable relations, so existing table actions resolve an object in its selected schema rather than implicitly in `public`.
- Include the navigator's supported object kinds: tables, views, materialized views, and functions. No new actions for views or functions are added.

## Dependency Graph

```text
Schema/object contracts
        |
        v
PostgreSQL discovery + schema-qualified loading
        |
        +-------------------+
        v                   v
Object modal rows     Sidebar selection lifecycle
        |                   |
        +---------+---------+
                  v
          Unit and integration coverage
```

## Task List

### Phase 1: Schema-aware database foundation

#### Task 1: Define the schema-object browsing contract

**Description:** Add driver-neutral values for schema/type pairs and a required database-interface method implemented by every adapter.

**Acceptance criteria:**

- [ ] The database package represents schema/type pairs for tables, views, materialized views, and functions.
- [ ] The database interface exposes discovery of non-empty pairs, and its existing list methods accept a schema for scoped PostgreSQL loading.
- [ ] Navigable table/view/materialized-view values retain schema identity.
- [ ] MySQL, Oracle, and SQLite satisfy the expanded `db.Database` interface through empty placeholder results.

**Verification:** Covered by the final automated verification run after Task 5.

**Dependencies:** None

**Files likely touched:**

- `internal/db/db.go`
- `internal/db/db_test.go`

**Estimated scope:** Small

#### Task 2: Implement PostgreSQL discovery and scoped access

**Description:** Query PostgreSQL catalogs for every non-empty schema/type pair visible to the connected role and make object/table operations schema-qualified.

**Acceptance criteria:**

- [ ] Discovery returns only non-empty pairs in deterministic schema/type order.
- [ ] Scoped loaders return objects only from the requested schema.
- [ ] Selecting a non-`public` table loads its data through a schema-qualified identifier.
- [ ] Existing table metadata/actions remain schema-correct.

**Verification:** Covered by the final automated verification run after Task 5.

**Dependencies:** Task 1

**Files likely touched:**

- `internal/db/postgres/postgres.go`
- `internal/db/postgres/ddl.go`
- `internal/db/postgres/postgres_test.go`
- `internal/db/postgres/ddl_test.go`

**Estimated scope:** Medium

### Checkpoint: Database foundation

- [ ] Schema/type discovery and scoped PostgreSQL loading are covered.
- [ ] Existing public-schema behavior remains intact.

### Phase 2: One-level picker and sidebar replacement

#### Task 3: Render schema/type entries in the object modal

**Description:** Replace the PostgreSQL modal's global object-type choices with rows for available schema/type pairs, while retaining the present fallback for other engines.

**Acceptance criteria:**

- [ ] PostgreSQL rows render as `schema — Object type`.
- [ ] The modal handles loading, errors, and zero available pairs safely.
- [ ] Up/Down, Enter, and Esc work within the modal bounds.
- [ ] Other engines preserve the current object-type modal.

**Verification:** Covered by the final automated verification run after Task 5.

**Dependencies:** Tasks 1-2

**Files likely touched:**

- `internal/app/objects_modal.go`
- `internal/app/objects_modal_test.go`
- `internal/app/model.go`

**Estimated scope:** Small

#### Task 4: Apply a selected pair to the navigator

**Description:** Wire modal selection to asynchronous scoped loading and replace the left sidebar with the selected schema/type's objects.

**Acceptance criteria:**

- [ ] Selecting a pair replaces sidebar items with exactly that schema/type.
- [ ] The sidebar title identifies its active schema and object type.
- [ ] Loading/failure belongs to the selected pair and leaves no stale item selected.
- [ ] Session/request checks discard stale results after reconnecting or changing the selection.

**Verification:** Covered by the final automated verification run after Task 5.

**Dependencies:** Task 3

**Files likely touched:**

- `internal/app/commands.go`
- `internal/app/update.go`
- `internal/app/navigator.go`
- `internal/app/model.go`
- `internal/app/test_helpers_test.go`

**Estimated scope:** Medium

### Checkpoint: End-to-end behavior

- [ ] Opening the PostgreSQL object modal shows available non-empty schema/type pairs.
- [ ] Selecting a pair updates the sidebar and preserves schema-qualified table loading.

### Phase 3: Regression coverage and verification

#### Task 5: Complete regression coverage and validate

**Description:** Close test gaps across the modal, lifecycle, and PostgreSQL adapter, then run the repository's end-of-change verification command.

**Acceptance criteria:**

- [ ] Modal behavior covers empty, one-result, multiple-schema, and unavailable-capability cases.
- [ ] Application tests cover selected schema/type state and stale responses.
- [ ] PostgreSQL integration tests use an isolated non-public schema, never a remote database.
- [ ] The full repository validation passes.

**Verification:**

- [ ] Run `scripts/validate.sh` once after all implementation work.
- [ ] User manually verifies choosing a schema/type pair in the TUI.

**Dependencies:** Tasks 1-4

**Files likely touched:**

- `internal/app/objects_modal_test.go`
- `internal/app/functions_lifecycle_test.go`
- `internal/app/navigator_test.go`
- `internal/db/postgres/postgres_test.go`

**Estimated scope:** Medium

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| A non-public table is resolved from `public` | High | Preserve schema in navigator items and use quoted schema-qualified PostgreSQL identifiers throughout table paths. |
| A prior asynchronous selection finishes after a newer one | High | Attach session, schema/type selection, and request identity to typed results. |
| System or inaccessible schemas flood the modal | Medium | Discover schemas visible to the connected role and omit empty schema/type pairs. |
| PostgreSQL changes alter other engines | Medium | Require empty placeholder implementations and invoke discovery only for PostgreSQL. |

## Open Questions

- None. "All available schemas" means schemas visible to the connected PostgreSQL role; empty schema/type pairs are omitted.
