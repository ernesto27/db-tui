# SQL autocomplete from loaded tables and columns

Date: 2026-08-25

## Executive conclusion

SQL autocomplete is feasible without replacing or forking the current Bubbles textarea. The best first implementation is an app-level completion controller around the existing editor, backed by a session-scoped schema catalog:

1. Reuse the table names that the navigator already loads.
2. Load columns lazily when the current statement references a table, deduplicate requests, and retain successful results for the database session.
3. Use a small, tolerant, dialect-aware lexer to recognize the token at the cursor, direct table contexts such as `FROM` and `JOIN`, and simple table aliases.
4. Rank deterministic prefix matches and insert the selected catalog identifier with engine-correct quoting.
5. Initially render a compact menu directly below the editor. A cursor-anchored overlay can be a later polish step because it requires more terminal-coordinate work.

This gives the high-value path—table completion and `alias.column` completion—without committing db-tui to a large parser, a textarea fork, eager catalog fan-out, or a language-server process. A full parser becomes worthwhile only if the product later promises accurate completion for nested scopes, CTE output columns, derived tables, correlated subqueries, and dialect-specific procedural SQL.

This report uses the following labels:

- **Verified** means the statement follows from current repository code or a cited primary source.
- **Recommendation** means it is the proposed design or an engineering inference from those facts.

## Current db-tui architecture

### Editor and event routing

**Verified.** `queryModel` directly owns a `textarea.Model`. The editor is created with no prompt or line numbers, is resized with the query panel, and shares the panel with query results. The query state currently contains execution, script, result-focus, and viewport state, but no completion state. See [`internal/app/query_panel.go`](../internal/app/query_panel.go), especially `queryModel`, `newQueryModel`, `resize`, and `view`.

**Verified.** Root key routing handles application shortcuts before forwarding remaining query-panel keys to `m.query.editor.Update(msg)`. `Tab` already toggles editor/results focus, `Ctrl+P` executes, and paste messages are forwarded separately to the textarea when the query editor is active. See [`internal/app/update.go`](../internal/app/update.go) and [`internal/app/keymap.go`](../internal/app/keymap.go). This means completion must be handled before the existing query shortcuts and textarea forwarding, but only while its menu is open or its explicit trigger is pressed.

**Verified.** The query view is rendered as a string inside the right panel. Root rendering already uses Lip Gloss composition for modal overlays, but not for content within the query editor. See [`internal/app/view.go`](../internal/app/view.go). Lip Gloss v2 supports positioned, z-ordered layers and a compositor, so a cursor-anchored menu is technically possible; it is additional layout work rather than a missing rendering primitive. See the pinned upstream [`layer.go` at v2.0.5](https://github.com/charmbracelet/lipgloss/blob/v2.0.5/layer.go).

### Metadata lifecycle

**Verified.** On connection initialization, db-tui concurrently loads tables, views, materialized views, and—where supported—functions. `tablesLoadedMsg` stores tables in `navigatorModel.tables`. The navigator is therefore the current in-memory owner of loaded table names. See [`internal/app/model.go`](../internal/app/model.go), [`internal/app/commands.go`](../internal/app/commands.go), [`internal/app/update.go`](../internal/app/update.go), and [`internal/app/navigator.go`](../internal/app/navigator.go).

**Verified.** Columns are not presently cached. `loadColumns` calls `Database.ListColumns` for one `db.Table`, and `columnsLoadedMsg` is accepted only when a matching columns modal, selected table, session, and request still exist. The result is stored in that modal and disappears with it. Existing lifecycle tests verify rejection of stale modal results. See [`internal/app/commands.go`](../internal/app/commands.go), [`internal/app/update.go`](../internal/app/update.go), and [`internal/app/update_lifecycle_test.go`](../internal/app/update_lifecycle_test.go).

**Verified.** The `Database` interface already exposes `Engine`, `ListTables`, and `ListColumns`; autocomplete does not require a new adapter method for a lazy implementation. `db.Table` contains only `Name`, with no catalog or schema field. See [`internal/db/db.go`](../internal/db/db.go).

**Verified.** The adapters expose deliberately narrow namespaces:

| Engine | Loaded table scope | Column lookup scope |
|---|---|---|
| PostgreSQL | Base tables in `public` | Relation name in `public` |
| MySQL | Base tables in `DATABASE()` | Table in `DATABASE()` |
| Oracle | `user_tables`, excluding materialized views | `user_tab_columns` for the current user |
| SQLite | Non-internal tables from `sqlite_master` | `pragma_table_xinfo(table)` |

The queries are in [`internal/db/postgres/postgres.go`](../internal/db/postgres/postgres.go), [`internal/db/mysql/mysql.go`](../internal/db/mysql/mysql.go), [`internal/db/oracle/oracle.go`](../internal/db/oracle/oracle.go), and [`internal/db/sqlite/sqlite.go`](../internal/db/sqlite/sqlite.go). Therefore, first-release completion should describe itself as completion for the currently loaded/default namespace, not general cross-schema completion.

### Existing asynchronous safety pattern

**Verified.** Database work runs in `tea.Cmd` functions with timeouts and returns typed messages. The root model increments `session` when the connection changes and ignores stale messages from earlier sessions; request counters provide finer-grained protection for row, modal, and query operations. This is the correct pattern to extend for catalog loading. See [`internal/app/commands.go`](../internal/app/commands.go) and `updateLifecycle` in [`internal/app/update.go`](../internal/app/update.go).

## Bubbles textarea constraints and extension options

The repository pins `charm.land/bubbles/v2 v2.1.1` and `charm.land/bubbletea/v2 v2.0.8` in [`go.mod`](../go.mod).

### What the pinned textarea provides

**Verified.** The v2.1.1 textarea exposes `Value`, `Line`, `Column`, `LineInfo`, `InsertString`, `MoveToBegin`, `MoveToEnd`, `CursorUp`, `CursorDown`, and `SetCursorColumn`, as well as its normal `Update` and `View`. Its logical buffer is private, but the public cursor API is enough for a wrapper to inspect and update the buffer. See the pinned upstream [`textarea.Model` implementation](https://github.com/charmbracelet/bubbles/blob/v2.1.1/textarea/textarea.go#L247-L347), [value insertion methods](https://github.com/charmbracelet/bubbles/blob/v2.1.1/textarea/textarea.go#L481-L492), and [cursor methods](https://github.com/charmbracelet/bubbles/blob/v2.1.1/textarea/textarea.go#L631-L729).

**Verified.** The textarea has no built-in suggestion model. Bubbles' single-line `textinput` does have `ShowSuggestions`, `SetSuggestions`, match accessors, and a current-suggestion index, but replacing the multiline SQL editor with `textinput` would lose multiline editing. Compare the pinned [`textarea` source](https://github.com/charmbracelet/bubbles/blob/v2.1.1/textarea/textarea.go) with [`textinput` suggestions at v2.1.1](https://github.com/charmbracelet/bubbles/blob/v2.1.1/textinput/textinput.go#L137-L151).

**Verified.** `textarea.Cursor()` can return the rendered cursor's visual coordinates when virtual-cursor mode is disabled, but the top-level application then has to offset and propagate that real cursor. See the pinned [`Cursor` implementation and example](https://github.com/charmbracelet/bubbles/blob/v2.1.1/textarea/textarea.go#L1598-L1635). This is useful for a later cursor-anchored overlay, but it expands the first implementation's rendering surface.

### Prefix replacement approaches

There are three practical ways to accept a candidate:

1. **Append only the unmatched suffix.** If the user typed `Alb` and the desired insertion is exactly `Album`, call `InsertString("um")`. This is the smallest and safest path because the textarea preserves its cursor and viewport. It cannot add identifier quotes around the already-typed prefix, correct catalog case, or replace a partial qualified token.
2. **Rebuild the value and restore the cursor.** Convert the editor value to runes, replace the completion span, call `SetValue`, then restore the logical position using `MoveToBegin`, `CursorDown` until `Line()` reaches the target logical row, and `SetCursorColumn`. The loop must check `Line()` rather than call `CursorDown` once per logical row because `CursorDown` also traverses soft-wrapped visual rows. The public v2.1.1 API supports this. Replacement must be rune-based, not byte-based, because textarea columns index its rune buffer. This handles quoting and arbitrary token replacement but needs careful multiline, soft-wrap, and Unicode tests.
3. **Synthesize backspace/delete key messages and then insert.** This uses only normal editor behavior but is slow for long prefixes, makes command handling harder to reason about, and is fragile at line boundaries. It should not be used.

**Recommendation.** Use a hybrid of options 1 and 2: append a suffix only when the desired insertion literally extends the current prefix; otherwise perform one rune-slice replacement and restore the cursor. Keep this logic behind a small editor-adapter function so a future Bubbles upgrade or editor replacement affects one place. A textarea fork is not justified for the first release.

## Recommended architecture

Separate completion into two models with different lifetimes:

```text
database session
    │
    ├── schemaCatalog
    │     ├── loaded table names
    │     └── per-table column state: unloaded/loading/ready/failed
    │
    └── queryModel
          ├── textarea.Model
          └── transient completion UI: context/candidates/selection/revision
```

### 1. Session-scoped schema catalog

**Recommendation.** Add a root-owned `schemaCatalog`, rather than putting metadata in `queryModel`. The cache is database-session state and can also serve the existing columns modal later; the query menu is ephemeral UI state.

A suitable shape is:

```go
type columnLoadState uint8

const (
    columnsUnloaded columnLoadState = iota
    columnsLoading
    columnsReady
    columnsFailed
)

type tableColumns struct {
    state      columnLoadState
    columns    []db.Column
    err        error
    generation uint64
}

type schemaCatalog struct {
    tables  []db.Table
    columns map[string]tableColumns
}
```

The real key should use exact catalog spelling, not a lowercased string. Matching can have a folded secondary index, but exact names must remain distinguishable—especially because MySQL table-name sensitivity depends on platform and `lower_case_table_names`. MySQL documents that table/database and alias sensitivity can vary while column names are case-insensitive. See [MySQL identifier case sensitivity](https://dev.mysql.com/doc/refman/8.4/en/identifier-case-sensitivity.html).

Catalog lifecycle:

- Reset the catalog on connection change at the same point where navigator and query state are reset.
- Seed table names when the current session's `tablesLoadedMsg` succeeds.
- Mark a table `loading` before returning a column-load command, preventing duplicate calls from repeated keystrokes.
- Return a general catalog message carrying `session`, exact table name, per-table `generation`, columns, and error.
- Always cache a current-session result even if the menu has since closed. Reopen or recompute the menu only if the editor is still focused and its current completion context still requests that table.
- Allow retry from `failed`; do not make one table's failure disable table completion or cached columns for other tables.
- Populate the same cache from a columns-modal request, or route both modal and autocomplete requests through one catalog loader. Avoid two independent request-counter systems for the same table.

**Recommendation.** Preserve the existing five-second metadata timeout initially. Do not start one `ListColumns` command for every table on connect. Bubble Tea explicitly defines `Batch` as concurrent with no result ordering, so an N-table batch can create an N-way burst against the database. See the pinned [`tea.Batch` contract](https://github.com/charmbracelet/bubbletea/blob/v2.0.8/commands.go#L9-L26) and implementation in [`tea.go`](https://github.com/charmbracelet/bubbletea/blob/v2.0.8/tea.go#L918-L950).

The preferred policy is:

- Table candidates are available as soon as tables load.
- Columns load on demand when a recognized statement references a table.
- Optionally prefetch only the active navigator table or the small set of tables found in the current statement.
- If a future feature warms the full catalog, use an explicitly bounded worker count or adapter-level bulk query—not an unbounded `tea.Batch` over tables.

### 2. Pure completion engine

**Recommendation.** Keep tokenization, context inference, matching, ranking, quoting, and replacement-span calculation pure. No database call belongs in this code or in an `Update` method.

Suggested internal types:

```go
type completionKind uint8 // table or column

type completionCandidate struct {
    kind       completionKind
    display    string
    detail     string
    insertText string
    tableName  string
}

type completionContext struct {
    startRune int
    endRune   int
    prefix    string
    qualifier string
    tableRefs []tableReference
    kind      completionKind
    valid     bool
}
```

The completion engine should consume:

- the complete editor value;
- the cursor's logical row and rune column;
- the active engine;
- loaded tables and ready column entries.

It should return context, candidates, and tables whose columns are needed. That makes the same logic easy to table-test.

### 3. Tolerant SQL context scanning

**Recommendation.** Do not require syntactically complete SQL. Autocomplete necessarily operates while a statement is incomplete, which is exactly when a strict parser is most likely to reject it. Implement a small lexer plus a shallow scope scanner.

Minimum lexical states:

- whitespace and punctuation;
- bare identifiers and keywords;
- single-quoted strings with doubled quotes;
- quoted identifiers;
- `--` line comments and `/* ... */` block comments;
- semicolons outside strings/comments, to isolate the statement containing the cursor;
- engine-specific literal and identifier forms listed in the dialect section below.

The lexer must suppress completion while the cursor is inside a string literal or comment. This is not optional polish: PostgreSQL supports dollar-quoted strings and nested block comments ([PostgreSQL lexical structure](https://www.postgresql.org/docs/current/sql-syntax-lexical.html)); MySQL has engine-specific comment rules ([MySQL comments](https://dev.mysql.com/doc/refman/8.4/en/comments.html)); Oracle supports `q'…'` alternative string delimiters ([Oracle literals](https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/Literals.html)); SQLite supports `--` and non-nesting `/* ... */` comments ([SQLite comment syntax](https://www.sqlite.org/lang_comment.html)). Exotic forms can be delivered in stages, but unsupported forms must be documented and regression-tested as they are added.

The first shallow scanner should support direct relation references in:

- `FROM table [AS] alias`;
- comma-separated `FROM` items;
- `JOIN table [AS] alias`;
- `UPDATE table [AS] alias`;
- `INSERT INTO table`;
- `DELETE FROM table [AS] alias`.

It should stop an implicit alias at clause/join keywords, punctuation, or another relation separator. PostgreSQL documents that a table alias becomes the relation's name within the query, which is why `a.` must resolve through the alias map rather than continue suggesting against the original name. See [PostgreSQL table and column aliases](https://www.postgresql.org/docs/current/queries-table-expressions.html#QUERIES-TABLE-ALIASES).

First-release context rules:

| Cursor context | Candidates | Insertion |
|---|---|---|
| After `FROM`, `JOIN`, `UPDATE`, `INSERT INTO`, or `DELETE FROM` | Loaded tables | Quoted table component |
| After `alias.` or `table.` | Columns for the resolved direct table | Quoted column component only |
| Expression position with one direct table in scope | That table's columns | Quoted column component |
| Expression position with multiple direct tables | Columns from all ready tables | Prefer `alias.`-qualified insertion; annotate owning table |
| Inside string/comment | None | None |
| Unknown derived table, CTE, or nested scope | None or only safe table candidates | Never guess a backing table |

Analyze the whole current statement, not only text before the cursor. For example, while completing the select list in `SELECT alb FROM Album`, the table reference occurs after the cursor but still provides useful context.

CTEs, subquery output columns, derived tables, set operations, and correlated nested scopes should be explicitly deferred. A wrong column suggestion is worse than no suggestion because it teaches the user that scope analysis is trustworthy when it is not.

### 4. Candidate matching and ranking

**Recommendation.** Start deterministic and conservative:

1. exact case-sensitive prefix;
2. case-insensitive prefix;
3. optionally, case-insensitive substring only after an explicit trigger.

Within a tier, preserve catalog order or sort by folded display name. Do not add fuzzy edit-distance scoring in the first release; it makes ranking harder to predict and test without materially improving the common SQL-prefix workflow.

Limit the visible menu to approximately eight rows and show total count when truncated. Candidate labels should distinguish kind and owner, for example:

```text
› "AlbumId"     column · Album
  "ArtistId"    column · Album
  "Title"       column · Album
```

For duplicate unqualified column names across joined tables, retain both entries and insert a qualifier. If the query supplies aliases, use the exact alias spelling already present in the SQL; do not quote or normalize it anew.

### 5. Identifier quoting by engine

**Recommendation.** Insert catalog-backed names using exact catalog spelling and dialect quoting by default. This is noisier than minimal quoting, but it avoids reserved-word checks and case-folding mistakes and is especially helpful for the mixed-case identifiers common in the Chinook database.

| Engine | Inserted identifier form | Verified considerations |
|---|---|---|
| PostgreSQL | `"name"`, double embedded `"` | Unquoted identifiers fold to lower case; double-quoted identifiers are case-sensitive and always identifiers. [PostgreSQL lexical structure](https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS) |
| MySQL | `` `name` ``, double embedded backticks | Backtick is the normal identifier quote. Double quotes work only with `ANSI_QUOTES`; table-name case behavior is configuration/platform dependent. [MySQL schema object names](https://dev.mysql.com/doc/refman/8.4/en/identifiers.html) and [case sensitivity](https://dev.mysql.com/doc/refman/8.4/en/identifier-case-sensitivity.html) |
| Oracle | `"name"` | Nonquoted identifiers are interpreted as uppercase; quoted identifiers are case-sensitive, and Oracle does not permit a double quote within an identifier. [Oracle object naming rules](https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/Database-Object-Names-and-Qualifiers.html) |
| SQLite | `"name"`, double embedded `"` | Double quotes are the standard identifier form; brackets and backticks are compatibility forms. SQLite recommends quoting keywords. [SQLite keywords and quoting](https://www.sqlite.org/lang_keywords.html) |

Qualified names must quote components separately, never the combined `alias.column` string. MySQL's manual states this rule directly for multipart identifiers. See [MySQL identifier qualifiers](https://dev.mysql.com/doc/refman/8.4/en/identifier-qualifiers.html).

If minimal quoting is later offered as a setting, it needs engine-specific reserved-word data and a round-trip case rule. It should not be approximated with one shared ASCII regular expression.

## Menu and keyboard UX

### Recommended first release

**Recommendation.** Render a compact completion menu immediately below the textarea and above query results. This fixed anchor is stable under soft wrapping, resizing, and textarea scrolling. It also avoids switching the current virtual cursor to a top-level real cursor solely to discover coordinates.

Open the menu automatically after a non-empty eligible prefix. Also provide an explicit trigger for empty-prefix discovery. `Ctrl+Space` is familiar but must be verified across the terminal input modes db-tui supports; provide a tested fallback such as `Alt+/` if it is not distinguishable in a target terminal.

While the menu is open:

- `Up`/`Down`: move selection and do not move the textarea cursor.
- `Tab` or `Enter`: accept the selected candidate. `Tab` only loses its existing editor/results-toggle meaning while the menu is visibly open.
- `Esc`: close the menu without modifying SQL.
- Typing a printable rune, backspace, delete, or paste: update the editor first, then recompute the menu.
- Left/right movement, page movement, editor blur, result focus, modal opening, query execution, new script, or panel switch: close or recompute; never leave a menu associated with an old cursor context.
- If required columns are loading, show one non-selectable `loading columns for …` row and keep table completion usable.
- If loading fails, close the loading state and allow the explicit trigger to retry; avoid persistent error noise in the query editor.

The footer should mention the explicit completion trigger only when the query editor is focused. Existing `Tab editor/results` help remains accurate because the menu itself makes the temporary `Tab` override visible.

### Cursor-anchored overlay as a later refinement

**Recommendation.** A true popup under the cursor is a second-stage UI improvement. The pinned textarea can expose visual cursor coordinates when configured for a real cursor, and Lip Gloss can compose positioned layers. Doing this correctly requires:

- switching and propagating the textarea cursor through `tea.View`;
- adding query-panel/header/padding offsets;
- clamping the menu above or below the cursor at panel edges;
- respecting soft-wrap and textarea viewport offsets;
- ensuring the popup does not leak across panel borders or obscure critical result rows.

The fixed menu should be implemented first so completion semantics and lifecycle can stabilize independently of coordinate math.

## Alternatives and tradeoffs

| Alternative | Advantages | Costs and conclusion |
|---|---|---|
| Wrapper + tolerant lexer + lazy cache | Small dependency surface; works on incomplete SQL; fits existing Bubble Tea architecture; supports all four engines incrementally | Requires careful lexer, replacement, and scope tests. **Recommended.** |
| Eagerly load every table's columns | All column candidates immediately available; simplest runtime lookup | N metadata calls at connection time; naive `tea.Batch` runs them concurrently; slow/error-prone for large schemas. Reject for the first release. |
| Cache only columns opened in the existing modal | Almost no new I/O | Completion quality depends on unrelated prior UI actions and feels inconsistent. Useful as an additional cache source, not the loading policy. |
| Fork/replace Bubbles textarea | Full control over buffer editing, ghost text, and popup coordinates | Ongoing editor maintenance, Unicode/wrapping/clipboard regressions, divergence from upstream. Not justified by the v2.1.1 public APIs. |
| Full dialect parser in-process | Better AST scopes after successful parsing | Autocomplete input is commonly incomplete; no single native Go choice cleanly covers all four dialects. `pg_query_go` uses PostgreSQL server parser source and cgo, so it is PostgreSQL-specific ([upstream repository](https://github.com/pganalyze/pg_query_go)); Vitess has its own yacc parser used for Vitess/MySQL planning ([official Vitess parser documentation](https://vitess.io/docs/contributing/contributing-to-ast-parser/)). Multiple parsers would raise build and maintenance cost. Defer. |
| Incremental parser such as Tree-sitter | Designed to update syntax trees and remain useful around errors | Still needs suitable maintained grammars and dialect-specific semantic/scope logic, plus new native/runtime dependencies. Tree-sitter's own documentation describes incremental parsing and useful results in the presence of syntax errors ([official introduction](https://tree-sitter.github.io/tree-sitter/)). Re-evaluate for advanced scopes. |
| External SQL language server | Potentially rich completion and diagnostics | Process management, protocol, versioning, connection/schema synchronization, and packaging across platforms are disproportionate to local catalog completion. Not recommended now. |

## Concrete implementation map

Likely files and responsibilities:

- New `internal/app/schema_catalog.go`: session catalog, exact-name lookup, per-table load states, request deduplication, and invalidation.
- New `internal/app/sql_completion.go`: lexer, statement isolation, simple table/alias scope, context detection, candidate ranking, quoting, completion state, replacement spans, and menu rendering.
- New `internal/app/sql_completion_test.go`: pure table-driven lexer/context/ranking/replacement/view tests.
- [`internal/app/query_panel.go`](../internal/app/query_panel.go): add transient completion state, include the menu in query rendering, clamp its width/height, and clear it from reset/focus transitions.
- [`internal/app/model.go`](../internal/app/model.go): own the catalog and initialize/reset it with the database session.
- [`internal/app/commands.go`](../internal/app/commands.go): add a catalog-oriented column message/command, or generalize the existing column command without coupling acceptance to the modal.
- [`internal/app/update.go`](../internal/app/update.go): route open-menu keys before normal query shortcuts, recompute after editor updates and paste, request missing columns, accept guarded results, and reset on connection/panel changes.
- [`internal/app/keymap.go`](../internal/app/keymap.go): add the explicit completion binding after its terminal representation is verified.
- [`internal/app/view.go`](../internal/app/view.go): conditional footer help; later, only if chosen, cursor propagation and popup compositing.
- [`internal/app/test_helpers_test.go`](../internal/app/test_helpers_test.go), [`internal/app/commands_test.go`](../internal/app/commands_test.go), and update/lifecycle tests: fake database observations, deadlines, stale results, and key-routing behavior.

No database-adapter edit is required for the recommended lazy path. Adapter work becomes appropriate only if profiling shows that per-table `ListColumns` calls are too expensive and a bulk metadata method is worth adding to all implementations.

## Staged delivery

### Stage 1: table completion and editor adapter

- Add the transient completion state and pure prefix matcher.
- Seed catalog tables from `tablesLoadedMsg`.
- Complete tables in direct relation positions.
- Implement dialect quoting and the hybrid suffix/rebuild replacement path.
- Render the fixed menu and route open-menu keys.

This stage validates the textarea integration without adding database traffic.

### Stage 2: lazy columns and aliases

- Add per-table column cache states and guarded async commands.
- Parse simple direct `FROM`/`JOIN`/DML references and aliases.
- Complete `alias.column` and unqualified columns for direct tables.
- Reuse cache entries for the columns modal and deduplicate simultaneous requests.

### Stage 3: dialect hardening and UX polish

- Complete literal/comment state coverage for each engine.
- Add duplicate-column qualification, loading/retry presentation, resize clamping, footer help, and explicit metadata refresh/invalidation.
- Decide from user feedback whether a cursor-anchored overlay is worth the additional cursor/rendering change.

### Stage 4: advanced scopes, only if demanded

- CTE names and declared output columns.
- Derived-table/subquery result columns.
- Nested and correlated scope resolution.
- Functions and engine-specific keywords.
- Evaluate an incremental or per-dialect parser using measured failure cases from real user SQL.

Following repository policy, implementation should be completed before automated checks are run, and the relevant verification should run once at the end (normally `scripts/validate.sh`). Manual UX testing remains with the user.

## Automated test strategy

Every behavior change needs focused coverage at the lowest practical layer.

### Pure lexer and context tests

Use table-driven tests across PostgreSQL, MySQL, Oracle, and SQLite for:

- cursor before, inside, and after identifiers;
- `FROM`, comma lists, each join form, `UPDATE`, `INSERT INTO`, and `DELETE FROM`;
- aliases with and without `AS`, quoted table names, and qualified column prefixes;
- semicolon-separated statements with the cursor in each statement;
- escaped/doubled identifier quotes;
- single-quoted strings, line comments, block comments, PostgreSQL dollar quotes/nested comments, MySQL backticks/comment forms, Oracle `q'…'`, and SQLite brackets;
- incomplete SQL, unmatched quotes, unmatched parentheses, and partial keywords;
- CTEs/subqueries returning no unsafe guessed column candidates.

### Replacement and Unicode tests

- suffix-only insertion preserves cursor, focus, value, and viewport behavior;
- replacement changes only the intended rune span;
- multiline cursor restoration at first/middle/last line and end-of-line, including logical lines that soft-wrap;
- non-ASCII identifiers before and inside the replaced token;
- combining characters and wide runes do not cause byte/rune slicing panics;
- embedded quote escaping for every engine;
- pasting multiline SQL recomputes from the final pasted value and never leaves a stale menu;
- catalog names containing terminal control characters are not offered for insertion and are sanitized in labels.

Logical cursor indices and terminal cell widths are different concerns. Replacement must use runes; menu sizing and clipping must use Lip Gloss display width.

### Catalog and command tests

- first request transitions `unloaded → loading` and returns one command;
- repeated keystrokes while loading do not issue duplicate calls;
- commands call `ListColumns` with a deadline and exact table identity;
- current-session success becomes `ready` even if the menu closed;
- stale session or generation results cannot populate a new connection's catalog;
- failure becomes retryable and does not discard other tables' ready columns;
- connection reset clears table and column metadata;
- a modal and completion request for the same table share one in-flight load.

### Update and key-routing tests

- normal textarea keys are unchanged when the menu is closed;
- open-menu `Up`/`Down`, `Tab`, `Enter`, and `Esc` are consumed by completion;
- closed-menu `Tab` still toggles editor/results exactly as today;
- `Ctrl+P`, new script, panel changes, modal openings, result focus, and connection changes close the menu;
- a late column response updates cache but does not reopen completion for a changed cursor/prefix;
- a matching late response recomputes candidates only when the editor still requests that table;
- paste routing remains correct for both open and closed menus;
- resize keeps menu dimensions within the query panel.

### View tests

Assert stable semantic fragments rather than full ANSI snapshots where possible:

- selected marker, candidate kind/owner, truncated-count indicator, and loading row;
- sanitization of database-provided labels;
- menu omission when disconnected, editor blurred, results focused, or context invalid;
- narrow and short layout behavior.

The recommended path does not need new database integration tests because it reuses the already tested `ListColumns` contract. Adapter tests are required only if the interface or metadata queries change.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Stale metadata after `CREATE`, `ALTER`, `DROP`, or external schema changes | Reset on reconnect; add an explicit metadata refresh; later invalidate after recognized successful DDL. Do not pretend the connection-lifetime cache is always fresh. |
| Incorrect suggestions in complex SQL scopes | Limit the first scanner to direct relations; withhold unsafe columns for CTEs/derived/nested scopes; add parser support only from observed cases. |
| Too many metadata calls | Lazy loads, in-flight deduplication, caching, and optional bounded prefetch. Never fan out an unbounded batch. |
| Old results contaminating a new connection | Carry and validate session plus per-table generation on every catalog message. |
| Key conflicts | Completion intercepts keys only while visibly open; preserve current behavior otherwise; test the explicit trigger on supported terminals. |
| Unicode corruption | Store replacement spans as rune offsets and restore logical row/rune column; use display-width functions only for rendering. |
| Popup layout artifacts | Fixed under-editor menu first; width/height clamp; defer cursor anchoring until semantics are stable. |
| Terminal escape/control characters in catalog names | Sanitize labels and decline to offer identifiers containing unsafe terminal control characters. Do not silently mutate a catalog name into different SQL. |
| Quoting changes user style | Prefer semantic correctness and exact catalog spelling initially; consider a tested minimal-quoting setting later. |
| Completion delays after `alias.` | Show a loading row, retain table candidates, cache the result, and make repeated triggers idempotent. |

## Acceptance criteria for the first useful release

The feature is ready when all of the following are true:

- Loaded tables appear immediately in valid relation contexts without extra database calls.
- Typing or explicitly requesting completion never blocks the Bubble Tea update loop.
- `alias.` requests columns at most once while loading and shows cached candidates afterward.
- Switching connections makes every prior catalog result harmless.
- Candidate acceptance preserves all text outside the current identifier, including multiline Unicode SQL.
- Inserted names are valid exact identifiers for PostgreSQL, MySQL, Oracle, and SQLite under the quoting rules above.
- No completion appears inside recognized literals or comments.
- Existing query execution, paste, scripts, focus toggle, results scrolling, and global shortcuts behave exactly as before whenever the menu is closed.
- Automated tests cover context, replacement, async lifecycle, stale messages, paste, Unicode, dialect quoting, key conflicts, and layout edges.

The recommended scope intentionally stops short of “SQL IDE” completeness. It delivers the most valuable interaction—discovering loaded relations and their columns—while keeping the architecture consistent with db-tui's current Bubble Tea and database boundaries.
