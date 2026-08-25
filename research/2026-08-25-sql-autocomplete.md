# Research: SQL autocomplete from loaded tables and columns

- **Date:** 2026-08-25
- **Question:** How should `db-tui` add responsive, context-aware SQL autocomplete using the tables and columns available from the active database connection?
- **Depth:** standard
- **Scope / constraints:** Go 1.26; Bubble Tea v2; Bubbles `textarea`; PostgreSQL, MySQL, Oracle, and SQLite; preserve the repository's Elm-style update loop and driver-neutral package boundaries; research only, no implementation.

## TL;DR

Build the first version as a small, pure-Go, dialect-aware lexical/context engine around the existing `textarea.Model`. Do not adopt a full SQL parser, language server, or a database-specific completion engine yet: none of the current Go options provides a stable, lightweight, schema-injected completion API spanning all four supported engines [12–18]. Keep a parser-facing resolver interface so a more semantic engine can be added later, while retaining lexical completion as the fallback for incomplete SQL.

Use the relations already loaded into the navigator and add a session-scoped, lazy column cache. Completion must only filter an in-memory snapshot while the user types; missing columns should be fetched asynchronously through `tea.Cmd`, keyed by the current connection session, and then cached [2–5, 9–11]. This follows both the repository's existing request/session guards and the architecture used by mature database clients.

For the first UX, make `Ctrl+Space` the reliable manual trigger, open automatically after `alias.`, use Up/Down to select, Tab or Enter to accept, and Esc to close. Render a small fixed suggestion list below the editor before attempting a cursor-anchored popup. The pinned Bubbles textarea has no completion API and does not expose enough screen-coordinate state to anchor a popup without duplicating soft-wrap math or switching cursor modes [1, 6–8].

The main caveat is semantic scope. Simple `FROM`/`JOIN` aliases and qualified columns are manageable with a tolerant scanner. Correct completion for nested queries, CTE output columns, correlated subqueries, and dialect-specific syntax is a substantially larger second phase [9–14].

## Key findings

- **The current editor can support a custom completion controller, but not built-in suggestions.** Bubbles `textarea` exposes the value, logical row/column, insertion, and cursor information, but unlike `textinput` it has no suggestions API [1, 6]. (fact)
- **Tables are already connection-scoped; columns are not cached.** The navigator retains tables, views, materialized views, and functions, while `ListColumns` results currently live only in inspection/edit modals [2–5]. (fact)
- **Database I/O must not happen during a keystroke update.** Mature clients use cached/introspected metadata and background refresh, and DBeaver explicitly stopped issuing metadata requests during automatic typing completion because of performance concerns [9–11]. (fact)
- **No current Go dependency is a clean four-dialect answer.** The closest options are either database-specific, CGO-backed, parser-only, pre-v1/internal, broad in dependency cost, or still planning completion [12–18]. (fact)
- **A tolerant lexical engine is the best first implementation.** It can complete simple clauses and aliases for all supported engines, continues to work on incomplete SQL, and preserves a clean upgrade path to semantic parsing. (judgment)
- **Manual invocation plus dot activation is the safest initial UX.** DataGrip, DBeaver, pgcli, and mycli all preserve an explicit completion trigger even when automatic completion is enabled [9–11, 24]. (fact and judgment)

## Current `db-tui` architecture

### Editor and key routing

`queryModel` owns a `textarea.Model`, query result state, focus state, and execution state [2]. The root update path intercepts application bindings before forwarding unhandled key presses to `textarea.Update` [3]. This matters because:

- Tab currently toggles editor/results focus.
- Ctrl+P executes the query.
- Enter is forwarded to the textarea and inserts a newline.
- Up/Down move the textarea cursor unless results are focused.
- Paste is handled separately through `tea.PasteMsg`.

A completion controller therefore belongs alongside `queryModel`. When its list is open, it must consume Up, Down, Tab, Enter, and Esc before the existing query-focus and textarea branches. When it is closed, existing behavior must remain unchanged. Suggestions should be recomputed after both printable key input and paste.

The pinned Bubbles `v2.1.1` textarea offers `Value`, `Line`, `Column`, `LineInfo`, `ScrollYOffset`, `InsertString`, and `SetValue` [1, 6]. Its `Word` method is whitespace-delimited, so it does not understand SQL punctuation such as `alias.column`, commas, parentheses, operators, or quoted identifiers [6]. SQL token extraction must be implemented separately.

Accepting a strict-prefix completion is straightforward: preserve the user's typed prefix and call `InsertString` with only the unmatched suffix. Replacing an arbitrary range is awkward in v2.1.1 because `SetValue` resets the buffer/cursor and there is no public logical-row setter or selection API [6]. Bubbles v2.2.x adds selection support, but still does not add autocomplete and does not make programmatic token-range replacement a compelling reason to upgrade immediately [7].

### Metadata lifecycle

On connection, the model asynchronously loads tables, views, materialized views, and functions. Successful messages populate the navigator, and the connection `session` protects against stale results [3–5]. PostgreSQL currently scopes relations to `public`; MySQL uses the active database; Oracle uses the current schema; SQLite reads its local catalog [4, 5]. The existing neutral types carry only an object name, not a schema-qualified identity.

Columns follow a different path. `loadColumns` calls `Database.ListColumns`, but the resulting `columnsLoadedMsg` is accepted only when the matching columns modal remains open. The data is then stored in the modal and discarded when it closes [3, 4]. Row editing performs a similar one-off load. Autocomplete needs an independent message and cache rather than coupling to either modal.

The database contract documents `ListColumns` in terms of `db.Table` [5]. Some adapters' catalog SQL may also work for views, but that behavior is not part of the current contract. A table-only first release is therefore the safest scope. Completing view columns should follow a deliberate contract decision, such as broadening the argument to a neutral relation type, rather than depending on accidental adapter behavior.

### Rendering

The query panel currently renders a heading, textarea, status, and results as one string [2]. Lip Gloss v2 already provides layers and a compositor, which the app uses for centered modal overlays [8]. However, the textarea defaults to a virtual cursor. Exact screen coordinates for a completion popup would require duplicating the textarea's soft-wrap/viewport calculations, or switching to its real cursor and propagating offsets through the top-level `tea.View` [1, 6, 8].

For v1, a fixed list directly below the editor is lower-risk and testable at narrow terminal sizes:

```text
RAW QUERY
SELECT a.tra
┌──────────────────────────────────────────┐
│ › TrackId       column · Album           │
│   Name          column · Album           │
│   Composer      column · Album           │
└──────────────────────────────────────────┘
Results ...
```

Limit the list to approximately six visible candidates and show candidate kind plus owning relation. A cursor-anchored overlay can be a later UX refinement after completion behavior is stable.

## Recommended user experience

### Triggers

1. `Ctrl+Space` always opens or refreshes completion when the query editor is focused.
2. Typing `.` opens completion immediately when the left side resolves to a table or alias.
3. Ordinary typing does not auto-open in the first release. It can be added later behind a setting after latency and visual-noise testing.
4. No suggestions appear inside string literals or comments.

This is intentionally more conservative than desktop IDE defaults. DataGrip supports automatic display but keeps `Ctrl+Space` as a fallback, while DBeaver supports both semantic and simpler engines and documents performance tradeoffs [9, 10]. pgcli's smart completion demonstrates the core terminal behavior: table candidates after `FROM`, column candidates after a known relation, and a broad fallback when smart completion is unavailable [11].

### Navigation and acceptance

- Up/Down moves the selected suggestion while the list is open.
- Tab accepts the selected suggestion. With no list open, Tab retains its current editor/results behavior.
- Enter accepts the selected suggestion while the list is open. Otherwise it continues to insert a newline.
- Esc closes the list without modifying SQL.
- Left/Right, cursor movement, execution, connection changes, and panel changes close stale suggestions.

`Ctrl+Space` is representable by the pinned Bubble Tea/Ultraviolet key stack, including the legacy NUL mapping. It should still receive focused TUI tests because terminal protocols can report control combinations differently [1].

### Candidate display and ranking

Each candidate should separate display text from insert text:

```go
type completionCandidate struct {
    Kind       completionKind
    Display    string
    InsertText string
    Detail     string
    Owner      string
}
```

Recommended ranking:

1. Exact case-sensitive prefix in the required syntactic category and current scope.
2. Case-insensitive prefix in the required category and current scope.
3. Other prefix matches from visible relations.
4. Functions and dialect/common SQL keywords.
5. Optional subsequence/fuzzy matches in a later phase.

Use deterministic alphabetical tie-breaking. When multiple relations expose the same column, show the owner and prefer qualified insertion rather than guessing. DataGrip and pgcli both use context to restrict candidates; their more advanced ranking, history, alias generation, and fuzzy matching are useful later enhancements rather than v1 requirements [9, 11].

## Metadata strategy

Use a catalog owned by the current query session:

```text
connection established
        │
        ├── tables/views/functions already load into navigator
        │
        └── completion catalog records available relation names
                    │
typing alias. ──────┤
                    ▼
             columns cached?
               │         │
              yes        no
               │         │
               ▼         ▼
       filter locally   tea.Cmd → ListColumns
                             │
                             ▼
                  session/request-guarded message
                             │
                             ▼
                   cache + recompute candidates
```

Suggested cache states per relation:

- `missing`: no request has started.
- `loading`: exactly one request is in flight.
- `ready`: immutable columns are available for local filtering.
- `failed`: retain a non-blocking error and allow explicit retry.

The cache must reset on connection changes and carry the connection `session` in every asynchronous response. A request sequence is also useful if explicit refresh can supersede an earlier request. Candidate generation receives a snapshot and performs no database calls.

Do not preload every table's columns during initial connection. Large schemas can create substantial latency and query volume; mature tools expose introspection scope, incremental refresh, prefetch settings, or background refresh precisely because metadata volume varies widely [9–11]. Lazy loading after `alias.` gives immediate value while bounding work.

Schema-changing SQL creates a staleness problem. The first release should provide an explicit completion-metadata refresh action. A later phase can detect successful `CREATE`, `ALTER`, `DROP`, and `RENAME` statements and refresh affected metadata, similar to targeted DDL introspection in DataGrip [9].

## Completion engine design

### API boundary

Keep candidate computation pure:

```go
Complete(sql string, cursorByte int, dialect Dialect, catalog Catalog) Result
```

`Result` should include candidates and the byte span of the incomplete token. The adapter from `textarea` must convert its logical rune row/column into a UTF-8-safe global byte offset. Keeping the pure engine independent from Bubble Tea makes its edge cases easy to table-test and leaves UI state in `internal/app`.

For v1 acceptance, insert only the unmatched suffix. Retain the replacement span in the engine API so a later version can support case correction, quoting, and replacement without redesigning context analysis.

### Tolerant lexer

Scan only the current statement, where semicolons inside strings/comments/quoted identifiers do not count as boundaries. Recognize:

- whitespace and punctuation;
- unquoted and quoted identifiers;
- single-quoted strings;
- line and block comments;
- numbers, bind parameters, and operators;
- parentheses depth;
- a partial token at the cursor.

Dialect policy must cover at least:

- PostgreSQL double-quoted identifiers, lowercase folding, dollar-quoted strings, and nested block comments [19].
- MySQL backtick identifiers, doubled backticks, and the `ANSI_QUOTES` caveat [20].
- Oracle double-quoted identifiers, uppercase normalization for unquoted names, and qualified object names [21].
- SQLite double quotes, backticks, square brackets, and its permissive keyword behavior [22].

The lexer should return no candidates while the cursor is inside a string/comment. In malformed input it should preserve any safely known context rather than treating a parse error as total completion failure.

### Context and alias resolution

For the innermost parenthesis depth/current statement:

- After `FROM`, `JOIN`, `UPDATE`, or `INTO`, suggest tables first.
- Record simple relation references in `FROM`/`JOIN`, with `AS` or bare aliases.
- After `qualifier.`, resolve aliases before physical table names and return only that relation's cached columns.
- In `SELECT`, `WHERE`, `ON`, `HAVING`, `GROUP BY`, and `ORDER BY`, suggest columns from visible relations, then functions/keywords.
- With one visible relation, rank its unqualified columns highest.
- With multiple visible relations, expose ownership and prefer qualified insert text for ambiguous names.

Stop v1 at simple scopes. Do not claim correct handling for CTE output columns, derived tables, correlated subqueries, select-list aliases, set operations, procedural blocks, or every dialect-specific clause. These should be explicit later milestones.

### Graceful fallback

Use layered precision:

1. Alias/table-scoped columns when context is understood and metadata is ready.
2. Tables or cached columns appropriate to the current top-level clause.
3. Loaded catalog object names.
4. Common/dialect keywords and loaded functions.
5. Words already present in the script, optionally in a later release.

DBeaver's separate semantic, legacy, combined, and script-word engines validate this pattern: semantic accuracy is valuable, but simpler fallback remains necessary when dialect syntax defeats deeper parsing [10].

## Option comparison

| Option | Four dialects | Incomplete SQL | Dependency/runtime cost | Schema-aware completion API | Fit |
|---|---:|---:|---:|---:|---|
| Custom tolerant lexer/context engine | Yes, via policies | Strong | Low; pure Go | Built for `db-tui` catalog | **Recommended for v1** |
| `sqls-server/sqls` | Broad, including current Oracle work | Moderate | Broad driver/LSP graph; pre-v1 | Completion is internal; server owns connections | Reference or spike only [12] |
| Tree-sitter SQL | Depends on community grammar | Strong | Official Go binding uses CGO/shared grammars | Semantic alias/catalog resolver still custom | Revisit if syntax services expand [13] |
| PostgreSQL/MySQL-specific parsers | No | Usually weak on partial DML | Medium to high; some CGO | Parser/AST, not completion | Do not use as the cross-dialect core [14–16] |
| Bytebase Omni | PostgreSQL ready; others WIP/absent | Parser-dependent | Pure Go, zero deps | Completion listed as planned | Promising project to monitor [17] |
| GoSQLX | Claims all four with recovery | Potentially good | Broad current dependency graph | No schema-aware completion API | Future isolated benchmark [18] |

`sqls` is the closest existing Go implementation, but its own README warns that it has no stable release, its completion implementation cannot be imported as a public package, and it expects its own database configuration [12]. Full parsers provide accurate ASTs for their target dialects but generally report errors on the incomplete statements that dominate editor completion. Tree-sitter provides error tolerance but adds grammar/runtime packaging and still leaves catalog semantics to the application [13–16].

Bytebase's production completers illustrate the semantic ceiling: current PostgreSQL and Oracle implementations contain substantial dialect-specific logic for byte/rune positions, grammar candidates, CTEs, aliases, dotted qualifiers, and metadata [17, 23]. That complexity argues for an intentionally narrow v1 rather than pretending a small parser dependency provides complete intelligence.

## Recommended phased delivery

### Phase 1: reliable catalog completion

- `Ctrl+Space` manual trigger.
- Auto-trigger after `.`.
- Tables after `FROM`/`JOIN`/`UPDATE`/`INTO`.
- Lazy table-column cache.
- Simple aliases and `alias.column` completion.
- Strict prefix matching and suffix-only insertion.
- Fixed suggestion list below the editor.
- Session-safe async metadata results.
- Correct handling of comments, strings, multiple statements, quoted identifiers, and UTF-8 cursor offsets.

This phase provides the majority of daily value without claiming full SQL semantics.

### Phase 1.1: polish and metadata control

- Explicit metadata refresh and retry after errors.
- Optional auto-open after a configurable minimum prefix.
- Identifier-aware insertion/quoting per engine.
- Functions and dialect keyword catalogs.
- View columns after the neutral relation contract is clarified.
- Fuzzy/abbreviation matching, capped and deterministically ranked.

### Phase 2: semantic scopes

- Nested query scopes and self-joins.
- CTE and derived-table output columns.
- Correlated subqueries.
- Select-list alias visibility by clause.
- `INSERT` column lists and `*` expansion.
- Parser-backed resolver when it succeeds, with the lexical engine retained as fallback.

### Later possibilities

- Foreign-key-aware join suggestions.
- Usage/frecency ranking.
- Snippets and function signatures.
- Cursor-anchored popup.
- Background prefetch policies for selected schemas.

## Testing strategy

### Pure engine tests

Use table-driven tests for:

- clause classification and prefix extraction;
- aliases with and without `AS`;
- qualified and ambiguous columns;
- nested parentheses and multiple statements;
- comments, open strings, and no-completion lexical states;
- quoted identifiers and doubled quote characters;
- UTF-8 identifiers and byte/rune offsets;
- stable ranking and candidate caps;
- PostgreSQL dollar quotes/nested comments, MySQL backticks, Oracle casing, and SQLite bracket identifiers.

### Bubble Tea/model tests

- Ctrl+Space only acts while the query editor is focused.
- Tab/Enter accept only while suggestions are open.
- Existing Tab focus switching and Enter-newline behavior remain unchanged when closed.
- Up/Down navigate candidates without moving the editor cursor while open.
- Esc, panel changes, execution, and connection changes close suggestions.
- Paste recomputes completion state.
- Column loads use `tea.Cmd`, show loading state, deduplicate in-flight requests, and ignore stale session/request messages.
- Small terminals cap or hide the list without corrupting the query/result layout.

### Adapter/contract tests

If Phase 1 remains table-only and reuses `ListColumns`, no new per-adapter query should be necessary. If view completion or schema-qualified objects are included, add explicit adapter tests and update neutral types/contracts rather than relying on current catalog-query accidents.

## Recommendation

Proceed with Phase 1 as a custom completion controller inside `internal/app` plus a pure lexical/context engine. Reuse the existing `Database.ListColumns` through a new completion-specific asynchronous message path, and store a cache keyed by the current connection session and relation name. Keep all database I/O in `tea.Cmd`; keep candidate filtering synchronous and memory-only.

Choose conservative UX defaults: explicit `Ctrl+Space`, automatic `alias.` completion, strict prefix ranking, suffix-only acceptance, and a fixed list below the editor. This works with the pinned dependencies and avoids prematurely changing editor components or adding a parser dependency.

Design the context resolver as an interface from the beginning, then revisit `sqls` parsing helpers, GoSQLX, or Bytebase Omni only for a Phase 2 spike. Do not replace the lexical engine: incomplete or dialect-specific SQL will always need a graceful fallback.

## Risks, unknowns, and open questions

- **Scope correctness:** CTEs, subqueries, alias shadowing, and procedural SQL can make a lexical engine confidently wrong. V1 must intentionally under-complete outside its supported grammar.
- **Column contract:** `ListColumns(db.Table)` does not formally promise view-column introspection.
- **Schema identity:** Current `db.Table`/`db.View` values lack schema fields, so multi-schema completion is out of scope without a neutral model change.
- **Metadata staleness:** DDL executed in the raw editor or another client can invalidate the cache. Start with explicit refresh; later add targeted invalidation.
- **Identifier insertion:** Strict suffix insertion cannot repair quoting or casing. Separate display/insert text and retain replacement spans for the later quoting phase.
- **Terminal keys:** Ctrl+Space is supported by the pinned input stack but should be regression-tested across legacy and enhanced terminal protocols.
- **Layout:** A fixed completion list reduces result space. A cursor popup improves proximity but increases coordinate and resize complexity.
- **Dependency recency:** Bubbles v2.2.1 was released one day before this report. It adds useful selection capabilities but no autocomplete; avoid bundling an unrelated dependency upgrade into Phase 1 without a separate evaluation [7].
- **Context7:** The preferred Context7 documentation channel was unavailable in this environment. The report substitutes the exact installed module source, tagged official repositories/releases, official product documentation, and the current repository source.

## Sources

1. [REPO] [`go.mod`](../go.mod) — pinned Go, Bubble Tea, Bubbles, and Lip Gloss versions — accessed 2026-08-25.
2. [REPO] [`internal/app/query_panel.go`](../internal/app/query_panel.go) — query editor and result model — accessed 2026-08-25.
3. [REPO] [`internal/app/update.go`](../internal/app/update.go) — key routing, lifecycle messages, and session reset — accessed 2026-08-25.
4. [REPO] [`internal/app/commands.go`](../internal/app/commands.go) — asynchronous relation/column/query commands — accessed 2026-08-25.
5. [REPO] [`internal/db/db.go`](../internal/db/db.go) and adapter implementations under [`internal/db`](../internal/db) — neutral catalog contract and engine-specific introspection — accessed 2026-08-25.
6. [DOC] [Bubbles `textarea` v2.1.1 tagged source](https://github.com/charmbracelet/bubbles/blob/v2.1.1/textarea/textarea.go) and [Bubbles v2.1.1 release](https://github.com/charmbracelet/bubbles/releases/tag/v2.1.1) — accessed 2026-08-25.
7. [DOC] [Bubbles v2.2.0 selection release](https://github.com/charmbracelet/bubbles/releases/tag/v2.2.0), [v2.2.1 textarea docs](https://pkg.go.dev/charm.land/bubbles/v2@v2.2.1/textarea), and [v2.2.1 release](https://github.com/charmbracelet/bubbles/releases/tag/v2.2.1) — accessed 2026-08-25.
8. [DOC] [Bubble Tea v2.0.8 tagged source](https://github.com/charmbracelet/bubbletea/blob/v2.0.8/tea.go) and [Lip Gloss v2.0.5 layers](https://github.com/charmbracelet/lipgloss/blob/v2.0.5/layer.go) — accessed 2026-08-25.
9. [DOC] [DataGrip 2026.2 code completion](https://www.jetbrains.com/help/datagrip/auto-completing-code.html), [metadata and introspection](https://www.jetbrains.com/help/datagrip/introspection.html), and [introspection levels](https://www.jetbrains.com/help/datagrip/introspection-levels.html) — updated August 2026; accessed 2026-08-25.
10. [DOC] [DBeaver SQL Assist and Auto Complete](https://dbeaver.com/docs/dbeaver/SQL-Assist-and-Auto-Complete/), [SQL Code Editor](https://dbeaver.com/docs/dbeaver/SQL-Code-Editor/), and [24.3 release notes](https://dbeaver.com/dbeaver-ultimate-24-3/) — accessed 2026-08-25.
11. [REPO] [pgcli README](https://github.com/dbcli/pgcli), [completer](https://github.com/dbcli/pgcli/blob/main/pgcli/pgcompleter.py), [completion refresher](https://github.com/dbcli/pgcli/blob/main/pgcli/completion_refresher.py), and [key bindings](https://github.com/dbcli/pgcli/blob/main/pgcli/key_bindings.py) — accessed 2026-08-25.
12. [REPO] [`sqls-server/sqls`](https://github.com/sqls-server/sqls), [`parser/parseutil` v0.2.48](https://pkg.go.dev/github.com/sqls-server/sqls@v0.2.48/parser/parseutil), and [`go.mod`](https://github.com/sqls-server/sqls/blob/master/go.mod) — accessed 2026-08-25.
13. [REPO] [Tree-sitter Go binding](https://github.com/tree-sitter/go-tree-sitter), [parser list](https://github.com/tree-sitter/tree-sitter/wiki/List-of-parsers), and [DerekStride SQL grammar](https://github.com/DerekStride/tree-sitter-sql) — accessed 2026-08-25.
14. [REPO] [`pg_query_go`](https://github.com/pganalyze/pg_query_go) and [`libpg_query` changelog](https://github.com/pganalyze/pg_query/blob/main/CHANGELOG.md) — PostgreSQL parser — accessed 2026-08-25.
15. [REPO] [Vitess SQL parser](https://github.com/vitessio/vitess/blob/main/go/vt/sqlparser/parser.go) and [Vitess releases](https://github.com/vitessio/vitess/releases) — MySQL parser — accessed 2026-08-25.
16. [REPO] [TiDB parser README](https://github.com/pingcap/tidb/blob/master/pkg/parser/README.md) and [Go package](https://pkg.go.dev/github.com/pingcap/tidb/pkg/parser) — MySQL-compatible parser — accessed 2026-08-25.
17. [REPO] [Bytebase Omni](https://github.com/bytebase/omni) — multi-engine parser/catalog status — accessed 2026-08-25.
18. [REPO] [GoSQLX repository](https://github.com/ajitpratap0/GoSQLX), [releases](https://github.com/ajitpratap0/GoSQLX/releases), and [`go.mod`](https://github.com/ajitpratap0/GoSQLX/blob/main/go.mod) — accessed 2026-08-25.
19. [DOC] [PostgreSQL 18 lexical structure](https://www.postgresql.org/docs/current/sql-syntax-lexical.html) — identifiers, strings, comments, and tokens — accessed 2026-08-25.
20. [DOC] [MySQL 8.4 schema object names](https://dev.mysql.com/doc/refman/8.4/en/identifiers.html) and [identifier qualifiers](https://dev.mysql.com/doc/refman/8.4/en/identifier-qualifiers.html) — accessed 2026-08-25.
21. [DOC] [Oracle Database object names and qualifiers](https://docs.oracle.com/en/database/oracle/oracle-database/23/sqlrf/Database-Object-Names-and-Qualifiers.html) — accessed 2026-08-25.
22. [DOC] [SQLite keywords and identifier quoting](https://www.sqlite.org/lang_keywords.html) — accessed 2026-08-25.
23. [REPO] [Bytebase PostgreSQL completer](https://github.com/bytebase/bytebase/blob/main/backend/plugin/parser/pg/completion.go) and [Oracle/PLSQL completer](https://github.com/bytebase/bytebase/blob/main/backend/plugin/parser/plsql/completion.go) — production semantic-completion complexity reference — accessed 2026-08-25.
24. [REPO] [mycli default configuration](https://github.com/dbcli/mycli/blob/main/mycli/myclirc), [completion refresher](https://github.com/dbcli/mycli/blob/main/mycli/completion_refresher.py), and [changelog](https://github.com/dbcli/mycli/blob/main/changelog.md) — completion triggers, metadata prefetch, and background refresh — accessed 2026-08-25.
