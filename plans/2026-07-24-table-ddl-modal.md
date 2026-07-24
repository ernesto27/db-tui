# Plan: Table DDL Modal

- **Date:** 2026-07-24
- **Domain(s):** Go application UI, database adapters, SQL introspection
- **Status:** Approved design; ready for implementation

## 1. Summary

Add `Ctrl+G` to show fresh, read-only table DDL in a scrollable modal for the currently selected table. The command works from the data and raw-query panels. MySQL returns its native `SHOW CREATE TABLE` result; PostgreSQL reconstructs a compact `CREATE TABLE` with inline constraints followed by standalone indexes, without invoking `pg_dump` or `pg_restore`.

## 2. Scope

### In scope

- A keyboard-driven DDL modal with loading, error, wrapped SQL, vertical scrolling, resize handling, and `Esc` close.
- A driver-neutral `TableDDL` operation and asynchronous Bubble Tea command/message flow.
- MySQL `SHOW CREATE TABLE` support.
- PostgreSQL catalog reconstruction of columns, inline constraints, and non-constraint indexes.
- Stale-result protection and unit/integration coverage.
- README shortcut documentation.

### Out of scope / non-goals

- DDL editing, execution, copying, or export.
- Full database dumps and dependencies beyond a selected table's structural script.
- PostgreSQL comments, ownership, grants, triggers, partitioned tables, or inherited tables.
- A `pg_dump`, `pg_restore`, or external SQL-parser dependency.

## 3. Resolved decisions

| # | Question | Decision |
|---|----------|----------|
| 1 | What is displayed? | A compact table definition: columns and inline constraints, followed by indexes. |
| 2 | May the output contain multiple statements? | Yes; PostgreSQL emits statements in dependency order. |
| 3 | How is the feature invoked? | `Ctrl+G` opens a modal for the selected table. |
| 4 | When does the command work? | Whenever a table is selected, including from raw-query mode. |
| 5 | Is the result editable or copyable? | No; it is view-only. |
| 6 | How fresh is the result? | Fetch it every time the modal opens. |
| 7 | Which engines are supported? | PostgreSQL and MySQL. |
| 8 | May PostgreSQL depend on `pg_dump`? | No. |
| 9 | How is PostgreSQL DDL obtained? | A read-only, catalog-driven reconstruction using PostgreSQL decompilation helpers. |
| 10 | What happens outside PostgreSQL's supported structure? | Return a clear error rather than incomplete DDL. |
| 11 | Does MySQL normalize its DDL? | No; preserve `SHOW CREATE TABLE` output verbatim. |
| 12 | How are long SQL lines displayed? | Wrap them to the modal content width; vertical scrolling only. |
| 13 | Are docs or plan artifacts committed now? | No; leave all work uncommitted. |

## 4. Design

### Database contract

Add this method to `internal/db/db.go`:

```go
// TableDDL returns a fresh executable structural DDL script for table.
TableDDL(ctx context.Context, table Table) (string, error)
```

All implementations validate the table, honor `ctx`, log their database statements, and wrap propagated errors. A plain `string` is sufficient because callers only render text and do not need engine-specific metadata.

### Modal and command lifecycle

`Model` gains `ddlModal *ddlModal` and `ddlRequest uint64`. A new `ddlModal` owns the selected table name, `loading`, `sql`, `err`, and the first wrapped line visible in its viewport.

`Ctrl+G` increments `ddlRequest`, creates the modal, starts the spinner, and returns `loadTableDDL`. The command calls `database.TableDDL` with `tableLoadTimeout` and returns:

```go
type tableDDLLoadedMsg struct {
	tableName string
	sql       string
	session   uint64
	request   uint64
	err       error
}
```

The lifecycle handler only applies the message when the session, request, active modal, and selected table all match. A close, later request, or connection change makes the old message inert.

The DDL modal is the same priority as other overlays. Its key handling is independent of the background panel:

| Key | Behavior |
|---|---|
| `Esc` | Close the modal. |
| `Up`, `k` / `Down`, `j` | Scroll one rendered line. |
| `PgUp` / `PgDown` | Scroll one viewport. |
| `Home` / `End` | Jump to the first / last viewport. |
| Any other key | No-op. |

Use a large responsive modal: width `min(100, max(40, layout.width-8))` and height `max(8, layout.height-6)`, including its border and padding. Render each sanitized source line with a width-constrained Lip Gloss style so long SQL wraps without horizontal scrolling. Preserve newline boundaries and convert unsafe control runes to replacement characters while retaining printable text and indentation tabs.

### PostgreSQL reconstruction boundary

PostgreSQL accepts only a relation in `public` with `relkind = 'r'`, no parent entry in `pg_inherits`, no child inheritance/partition entry, and no partition marker. It executes all introspection inside a `pgx.TxOptions{AccessMode: pgx.ReadOnly}` transaction with a catalog-oriented search path.

The adapter fetches metadata and emits it in this order:

1. `CREATE TABLE public.<quoted name>` and columns.
2. Primary-key, unique, check, exclusion, and foreign-key constraints inline in that statement.
3. `CREATE INDEX` statements returned by `pg_get_indexdef`, excluding `pg_constraint.conindid` indexes.

The column query uses `pg_attribute`, `pg_attrdef`, `pg_collation`, and related catalogs. It renders types with `format_type`, defaults/generated expressions with `pg_get_expr`, and includes nullability, collation, identity, and generated-column clauses. Query identity sequence options from `pg_sequence`; query serial sequence ownership through `pg_depend`. Use `pg_get_constraintdef` and `pg_get_indexdef` instead of reproducing PostgreSQL's constraint/index syntax in Go.

All relation names used in generated statements go through `pgx.Identifier{"public", name}.Sanitize()`. Catalog name lookups use query parameters, never interpolated user input.

## 5. Interfaces and state transitions

```text
Ctrl+G with selected table
  -> ddlRequest++ and new ddlModal{loading: true}
  -> loadTableDDL(database, table, session, request)
  -> tableDDLLoadedMsg
  -> matching result: modal.sql or modal.err; loading false
  -> stale result: ignore

Esc
  -> ddlModal = nil
```

The spinner runs while `ddlModal.loading` is true. `WindowSizeMsg` recalculates the wrapped SQL line count and clamps the modal offset. `footerText` includes `Ctrl+G table DDL` only when a selected table exists. `View` passes the current layout to the modal renderer and includes it in the existing overlay compositor.

## 6. Implementation tasks

### Task 1 — Extend the database contract and async command path

- **Why:** Give the app an engine-neutral operation and keep all database I/O outside Bubble Tea `Update`.
- **Files & changes:**
  - `internal/db/db.go` (edit `Database`): add the documented `TableDDL(context.Context, Table) (string, error)` method.
  - `internal/app/commands.go` (edit after `loadRows`): add `tableDDLLoadedMsg` and `loadTableDDL(database, table, session, request) tea.Cmd`. Create the same five-second context as `loadRows`; return the table name, SQL, session, request, and error.
  - `internal/app/test_helpers_test.go` (edit `fakeDatabase`): add `ddl string`, `ddlErr error`, call-count/table/deadline fields, and a `TableDDL` method mirroring `GetRows`' context recording.
  - `internal/app/commands_test.go` (new `TestLoadTableDDL` table test): assert success and error forwarding, the selected table, session/request correlation fields, and a command deadline.
- **Depends on:** —

### Task 2 — Build the read-only DDL modal

- **Why:** Isolate rendering, wrapping, scrolling, and terminal-safe SQL display from the root model.
- **Files & changes:**
  - `internal/app/ddl_modal.go` (new): define `ddlModal` and helpers:

    ```go
    type ddlModal struct {
		tableName string
		loading   bool
		sql       string
		err       error
		offset    int
	}

    func newDDLModal(tableName string) ddlModal
    func (m *ddlModal) finish(sql string, err error, layout appLayout)
    func (m *ddlModal) scroll(delta int, layout appLayout)
    func (m *ddlModal) jumpToEnd(layout appLayout)
    func (m ddlModal) view(layout appLayout, spinner string) string
    ```

    The renderer has loading, error, and successful-SQL states; uses the same rounded border and color palette as `dumpModal`; and derives `visibleRows` from its rendered height. Its header is `DDL · public.<table>` and its help is `↑/↓ scroll  •  PgUp/PgDown page  •  Esc close`.
  - `internal/app/text.go` (edit): add a multiline sanitizer used only by the DDL modal. It preserves `\n` and `\t`, replaces other control runes, and does not call `sanitizeText`, which deliberately removes newlines.
  - `internal/app/ddl_modal_test.go` (new): table-driven tests for loading/error/success render states, escaped control data, wrapping, all scroll boundaries, and short terminal dimensions.
- **Depends on:** Task 1.

### Task 3 — Integrate the modal into root state, keys, lifecycle, and footer

- **Why:** Make `Ctrl+G` work consistently from both panels and protect the UI from obsolete async messages.
- **Files & changes:**
  - `internal/app/model.go` (edit `Model`): add `ddlModal *ddlModal` and `ddlRequest uint64`.
  - `internal/app/keymap.go` (edit `keyMap` and `defaultKeyMap`): add `tableDDL key.Binding` with `key.WithKeys("ctrl+g")` and help text `"table DDL"`.
  - `internal/app/update.go`:
    - Add the DDL modal to the overlay guard after existing connection modals.
    - In `updateLifecycle`, handle `tableDDLLoadedMsg`; require matching session, request, active modal, and modal table name before calling `finish`.
    - In `updateKey`, before panel-specific query editor routing, handle `tableDDL`: obtain `navigator.selectedTable`; if a database and table exist, increment the request, create `newDDLModal`, and return `tea.Batch(loadTableDDL(...), m.startSpinner())`.
    - Add `updateDDLModal` to close on `Esc` and route the listed scroll keys. It never forwards keys to the background panel.
    - Include `m.ddlModal != nil && m.ddlModal.loading` in the spinner condition.
    - Clear the modal and invalidate in-flight results whenever the active connection is replaced or removed.
    - On window resize, clamp the DDL viewport when the modal is open.
  - `internal/app/view.go`:
    - Include `m.ddlModal != nil` in `View`'s overlay condition, add the DDL branch to `renderModalOverlay`, and call `m.ddlModal.view(m.layout, m.spinner())`.
    - Append `Ctrl+G table DDL` to connected table and raw-query footers only when `navigator.selectedTable()` succeeds.
  - `internal/app/update_key_test.go`, `internal/app/update_lifecycle_test.go`, and `internal/app/view_test.go`: add coverage for opens from data/raw-query modes, invalid no-op states, modal key isolation, current/stale result behavior, connection reset, spinner state, and footer text.
- **Depends on:** Tasks 1–2.

### Task 4 — Implement MySQL table DDL retrieval

- **Why:** Use the database server's canonical output for MySQL with minimal transformation.
- **Files & changes:**
  - `internal/db/mysql/mysql.go` (edit after `GetRows`): add:

    ```go
    func (m *mysqlDatabase) TableDDL(ctx context.Context, table db.Table) (string, error) {
		if table.Name == "" {
			return "", errors.New("table name is required")
		}
		query := "SHOW CREATE TABLE " + quoteIdentifier(table.Name)
		m.logger.Log(query)
		row := m.database.QueryRowContext(ctx, query)
		var returnedName, statement string
		if err := row.Scan(&returnedName, &statement); err != nil {
			return "", fmt.Errorf("show MySQL table DDL: %w", err)
		}
		return ensureTrailingSemicolon(statement), nil
	}
    ```

    Add `ensureTrailingSemicolon` beside `quoteIdentifier`; trim only trailing whitespace to decide whether a semicolon is needed, while preserving the server's returned SQL otherwise.
  - `internal/db/mysql/mysql_test.go` (edit): integration test a known `world` table; assert a `CREATE TABLE` prefix, expected quoted name, non-empty columns/indexes, and trailing semicolon. Add unit tests for empty names and semicolon normalization.
- **Depends on:** Task 1.

### Task 5 — Implement PostgreSQL metadata retrieval and DDL assembly

- **Why:** Produce structural DDL without an external dump executable while refusing relation structures the first release cannot render faithfully.
- **Files & changes:**
  - `internal/db/postgres/ddl.go` (new): keep catalog SQL, metadata structs, query helpers, and the pure assembler together. Define:

    ```go
    func (p *postgresql) TableDDL(ctx context.Context, table db.Table) (string, error)
    func loadTableDDLMetadata(ctx context.Context, tx pgx.Tx, table db.Table) (tableDDLMetadata, error)
    func buildTableDDL(metadata tableDDLMetadata) (string, error)
    ```

    `TableDDL` validates the name, starts a `pgx.TxOptions{AccessMode: pgx.ReadOnly}` transaction, sets the local search path, loads metadata, builds the script, and commits. Roll back on every earlier return.

    Use a relation lookup query joining `pg_class` and `pg_namespace`; reject missing relations, `relkind != 'r'`, `relispartition`, and any `pg_inherits` row where the relation is parent or child. Query `pg_attribute` rows ordered by `attnum`, excluding dropped/system attributes. Query serial-owned sequences through `pg_depend`/`pg_sequence`; query constraints through `pg_constraint` plus `pg_get_constraintdef`; and query standalone indexes through `pg_index` plus `pg_get_indexdef`, excluding `conindid` values.

    The assembler must quote the public table name and sequence names with `pgx.Identifier`, join statements with a blank line, add semicolons once, and return errors for metadata combinations it cannot represent. It must never emit comments, grants, ownership changes, triggers, or duplicate constraint indexes.
  - `internal/db/postgres/ddl_test.go` (new): pure table-driven tests for generated SQL ordering, quoted names, types, defaults, nullability, identity/generated clauses, owned sequences, every constraint category, standalone-index filtering, and unsupported metadata. Keep examples small so expected SQL is exact.
  - `internal/db/postgres/postgres_test.go` or `internal/db/postgres/query_test.go` (edit): local Compose integration tests against Chinook's `Album` and a temporary ordinary table created through `Execute`. Verify DDL contains the selected table declaration, constraints/indexes, no `GRANT`/`OWNER`/trigger text, fresh changes after `ALTER TABLE`, cancellation/error wrapping, and rejection after creating a partitioned or inherited test relation. Clean up every temporary test object with `t.Cleanup`.
- **Depends on:** Task 1.

### Task 6 — Document and verify the finished user-facing command

- **Why:** Keep user documentation aligned with the shortcut and ensure the whole repository passes its standard checks.
- **Files & changes:**
  - `README.md` (edit the navigation paragraph): add that `Ctrl+G` opens a read-only DDL modal for the selected table, works from both table-data and raw-query mode, and shows PostgreSQL/MySQL engine-specific structural DDL.
  - `docs/adr/0001-table-ddl-catalog-reconstruction.md` and `docs/glossary.md` (already created, uncommitted): retain these records with the implementation; do not create a `docs/superpowers/` document.
  - Run `gofmt -w` on changed Go files, `go test ./...`, `go vet ./...`, and `scripts/validate.sh` with the local Compose services running.
- **Depends on:** Tasks 2–5.

## 7. Testing

### Unit tests

- Command deadline, result payload, and error forwarding.
- DDL modal wrapping, multiline terminal sanitization, viewport math, key behavior, and resize clamping.
- Root-model key routing, modal isolation, unavailable states, stale result rejection, spinner lifecycle, and connection invalidation.
- MySQL empty-name validation and semicolon normalization.
- PostgreSQL pure assembler output for each column and structural object category, quoting, ordering, and unsupported structures.

### Integration tests

- MySQL Compose: `SHOW CREATE TABLE` returns a terminating canonical script for a known World table.
- PostgreSQL Compose: the Chinook `Album` script contains columns, primary/foreign keys, and indexes; DDL reflects a table altered after the first request; invalid names wrap database errors; partitioned/inherited relations return the explicit unsupported error.

## 8. Acceptance criteria

- Pressing `Ctrl+G` with a selected table opens a responsive, read-only modal from both right-panel modes.
- SQL wraps to the modal width and scrolls predictably with all documented keys; `Esc` closes it.
- The modal shows loading and contextual errors without changing the underlying panel or selection.
- Closing, reconnecting, or issuing a later request prevents stale async DDL from updating the modal.
- MySQL displays fresh `SHOW CREATE TABLE` DDL.
- Supported PostgreSQL ordinary public tables display executable structure without external dump tools, comments, grants, ownership, triggers, or duplicate constraint indexes.
- PostgreSQL partitioned/inherited relations fail clearly rather than returning partial output.
- New focused tests pass, then `scripts/validate.sh` passes with local Compose services.

## 9. Risks and open items

- PostgreSQL catalog reconstruction deliberately has a smaller support boundary than `pg_dump`; the adapter must reject any catalog feature it cannot render faithfully rather than silently omitting it.
- Foreign-key statements assume their referenced tables already exist when the displayed single-table script is executed; cross-table dependency ordering is explicitly outside this feature.
- The local PostgreSQL/MySQL Compose services are required for adapter integration tests; no remote credentials are used.
