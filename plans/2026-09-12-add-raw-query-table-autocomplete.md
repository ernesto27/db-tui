# Plan: Add Raw-Query Table Autocomplete for All Engines

- **Date:** 2026-09-12
- **Domain(s):** TUI frontend, SQL-dialect behavior
- **Author:** plan-from-spec (reviewed with repository user)
- **Status:** Draft

## 1. Summary

Bring MySQL, Oracle, SQLite, and SQL Server to parity with the existing
PostgreSQL raw-query table autocomplete. The editor will reuse the already
loaded active-schema base tables and insert an identifier quoted for the
connected engine. PostgreSQL remains the unchanged behavior baseline.

## 2. Scope

### In scope

- All five supported engines; candidates remain `navigator.tables`, maximum 5.
- Existing `FROM`, `JOIN`, `UPDATE`, `INTO`, and `TRUNCATE` contexts.
- Existing Up/Down navigation, Enter/Tab acceptance, and Escape dismissal.
- Existing engine SQL highlighters and engine-specific identifier quoting.
- Unit and root-model integration tests, then one `scripts/validate.sh` run.

### Out of scope / non-goals

- Views, columns, aliases, schema/database completion, fuzzy matching, or a
  general SQL parser.
- New metadata queries, adapter changes, `db.Database` changes, dependencies,
  or architecture-document changes.

## 3. Resolved decisions

| # | Question | Decision |
|---|---|---|
| 1 | Engines | PostgreSQL, MySQL, Oracle, SQLite, SQL Server. PostgreSQL stays the baseline. |
| 2 | Candidate source | Existing loaded base tables in the active navigator schema only. |
| 3 | Trigger and keys | Preserve current five trigger keywords plus Up/Down, Enter/Tab, Escape. |
| 4 | Inserted syntax | Double quotes for PostgreSQL/Oracle/SQLite; backticks for MySQL; brackets for SQL Server. Escape closing delimiters. |
| 5 | Availability | Hide completion when disconnected, tables load/fail, no match exists, lexical context is invalid, or engine is unknown. |
| 6 | I/O and contracts | No new I/O and no public-interface change. |
| 7 | Tests | Unit tests for dialect/prefix/replacement and a root-model key-flow integration test; live DB integration is inapplicable because adapters and I/O do not change. |
| 8 | Verification | Run `scripts/validate.sh` after all implementation and tests. |

## 4. Design

`Model.updateQueryEditor` already owns editor updates and accesses the connected
database. It will use `rawQueryHighlighter(m.database)` instead of checking
specifically for PostgreSQL. `queryModel` remains the owner of suggestion
state, matching, token replacement, cursor restoration, and overlay rendering.

```text
key press -> Model.updateQueryEditor
          -> rawQueryHighlighter(database)
          -> query.refreshTableCompletion(tables, engine, highlighter)
          -> completion state
Enter/Tab -> quote selected catalog name for retained engine -> replace token
```

`tableCompletionModel` retains the creating engine so later acceptance is
deterministic. App code continues to depend only on `db` and `sqlhighlight`,
never on an adapter.

## 5. Interfaces & contracts

```go
func (m *queryModel) refreshTableCompletion(
	tables []db.Table,
	engine string,
	highlighter sqlhighlight.Highlighter,
)

func tableCompletionPrefix(
	editor textarea.Model,
	highlighter sqlhighlight.Highlighter,
) (start, end int, prefix string, ok bool)

func tableCompletionKeywordBefore(
	runes []rune,
	start int,
	highlighter sqlhighlight.Highlighter,
) bool

func quoteTableCompletionIdentifier(engine, identifier string) string
```

`updateQueryEditor` must never call these with a nil highlighter; it dismisses
completion instead.

| Engine | `order` insertion | escaping |
|---|---|---|
| PostgreSQL, Oracle, SQLite | `"order"` | `"` → `""` |
| MySQL | `` `order` `` | `` ` `` → `` `` `` |
| SQL Server | `[order]` | `]` → `]]` |

## 6. Behavior & states

| State | Result |
|---|---|
| Connected supported engine, loaded tables, allowed keyword prefix | Show up to five matches. |
| Disconnected, loading/error, unknown engine, invalid lexical region, no match | Hide completion. |
| Visible + Up/Down | Move selected index within bounds. |
| Visible + Enter/Tab | Replace full active token with engine-quoted selected name, move cursor after it, dismiss. |
| Visible + Escape | Dismiss with no text change. |

Existing query execution and focus changes keep dismissing completion.

## 7. Implementation tasks

### Task 1 — Generalize completion to the active SQL dialect

- **Why:** Current completion hard-codes the PostgreSQL lexer, quote function,
  and UI gate.
- **Files & changes:**

  - `internal/app/query_autocomplete.go` (edit):

    ```diff
     type tableCompletionModel struct {
    +    engine   string
         matches  []db.Table
         selected int
         start    int
         end      int
         visible  bool
     }

    -func (m *queryModel) refreshTableCompletion(tables []db.Table) {
    -    start, end, prefix, ok := tableCompletionPrefix(m.editor)
    +func (m *queryModel) refreshTableCompletion(
    +    tables []db.Table, engine string, highlighter sqlhighlight.Highlighter,
    +) {
    +    start, end, prefix, ok := tableCompletionPrefix(m.editor, highlighter)
    @@
    -    m.completion = tableCompletionModel{matches: matches, start: start, end: end, visible: true}
    +    m.completion = tableCompletionModel{
    +        engine: engine, matches: matches, start: start, end: end, visible: true,
    +    }
    @@
    -    replacement := []rune(quotePostgreSQLIdentifier(completion.matches[completion.selected].Name))
    +    replacement := []rune(quoteTableCompletionIdentifier(
    +        completion.engine, completion.matches[completion.selected].Name,
    +    ))
    @@
    -func tableCompletionPrefix(editor textarea.Model) (start, end int, prefix string, ok bool) {
    +func tableCompletionPrefix(editor textarea.Model, highlighter sqlhighlight.Highlighter) (start, end int, prefix string, ok bool) {
    @@
    -    if !tableCompletionCodePosition(runes, start) || !tableCompletionKeywordBefore(runes, start) {
    +    if !tableCompletionCodePosition(runes, start) || !tableCompletionKeywordBefore(runes, start, highlighter) {
    @@
    -func tableCompletionKeywordBefore(runes []rune, start int) bool {
    -    spans := (sqlhighlight.PostgreSQL{}).KeywordSpans(string(runes[:start]))
    +func tableCompletionKeywordBefore(runes []rune, start int, highlighter sqlhighlight.Highlighter) bool {
    +    spans := highlighter.KeywordSpans(string(runes[:start]))
    @@
    -func quotePostgreSQLIdentifier(identifier string) string {
    +func quoteTableCompletionIdentifier(engine, identifier string) string {
    +    switch engine {
    +    case db.EngineMySQL:
    +        return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
    +    case db.EngineSQLServer:
    +        return "[" + strings.ReplaceAll(identifier, "]", "]]") + "]"
    +    default:
             return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
    +    }
     }
    ```

    Keep matching, overlay, cursor, and dismissal behavior unchanged; use
    `gofmt` for final formatting.

  - `internal/app/update.go` (edit, `Model.updateQueryEditor`):

    ```diff
    -if m.database != nil &&
    -    m.database.Engine() == db.EnginePostgreSQL &&
    -    !m.loading &&
    -    m.tableLoadErr == nil {
    -    m.query.refreshTableCompletion(m.navigator.tables)
    -} else {
    +if m.database == nil || m.loading || m.tableLoadErr != nil {
         m.query.completion.dismiss()
    +    return command
     }
    +highlighter := rawQueryHighlighter(m.database)
    +if highlighter == nil {
    +    m.query.completion.dismiss()
    +    return command
    +}
    +m.query.refreshTableCompletion(m.navigator.tables, m.database.Engine(), highlighter)
    ```

    `rawQueryHighlighter` already maps all supported engines; do not create a
    second engine-to-highlighter map.

- **Depends on:** —

### Task 2 — Add dialect and interaction regression coverage

- **Why:** Each engine must show a suggestion and insert executable identifier
  syntax without regressing lexical or availability safeguards.
- **Files & changes:**

  - `internal/app/query_autocomplete_test.go` (edit): import
    `internal/app/sqlhighlight`; pass an engine and highlighter to all direct
    `refreshTableCompletion`/`tableCompletionPrefix` calls (existing tests use
    PostgreSQL arguments). Replace `TestQuotePostgreSQLIdentifier` with:

    ```go
    func TestQuoteTableCompletionIdentifier(t *testing.T) {
        tests := []struct {
            name, engine, identifier, want string
        }{
            {"PostgreSQL", db.EnginePostgreSQL, `has"quote`, `"has""quote"`},
            {"MySQL", db.EngineMySQL, "has`quote", "`has``quote`"},
            {"Oracle", db.EngineOracle, `has"quote`, `"has""quote"`},
            {"SQLite", db.EngineSQLite, `has"quote`, `"has""quote"`},
            {"SQL Server", db.EngineSQLServer, "has]quote", "[has]]quote]"},
        }
        for _, test := range tests {
            t.Run(test.name, func(t *testing.T) {
                assert.Equal(t, test.want,
                    quoteTableCompletionIdentifier(test.engine, test.identifier))
            })
        }
    }
    ```

    Add valid `SELECT * FROM al` prefix cases for MySQL, Oracle, SQLite, and
    SQL Server. Add invalid cases inside a MySQL `#` comment and backtick
    identifier, an Oracle alternative quote, and a SQL Server bracket
    identifier. Extend selection acceptance with every engine's expected
    replacement: `"Album"`, `` `Album` ``, `"Album"`, `"Album"`, `[Album]`.
    Preserve the current PostgreSQL Enter, Tab, and middle-token tests.

  - `internal/app/update_key_test.go` (edit): replace the single PostgreSQL
    positive / MySQL-negative pair with a table-driven positive root-model
    assertion for all five engines. For each: focus the query editor, load
    `Album` into `navigator.tables`, type `a` after `SELECT * FROM `, and
    assert a visible completion whose first match is `Album`. Keep the
    existing loading-state negative test.

    ```go
    for _, engine := range []string{
        db.EnginePostgreSQL, db.EngineMySQL, db.EngineOracle,
        db.EngineSQLite, db.EngineSQLServer,
    } {
        t.Run("opens completion for "+engine, func(t *testing.T) {
            model := New(config.Config{}, ConnectionSettings{}, nil)
            model.database = &fakeDatabase{engine: engine}
            model.panel = panelQuery
            model.navigator.tables = []db.Table{{Name: "Album"}}
            model.query.editor.SetValue("SELECT * FROM ")
            _ = model.query.focusEditor()

            got, command := updateModel(t, model, keyPress('a', "a", 0))

            assert.Nil(t, command)
            require.True(t, got.query.completion.visible)
            assert.Equal(t, "Album", got.query.completion.matches[0].Name)
        })
    }
    ```

- **Depends on:** Task 1.

### Task 3 — Run final verification

- **Why:** Repository policy requires complete validation after implementation.
- **Files & changes:** None.
- **Command:**

  ```sh
  scripts/validate.sh
  ```

- **Depends on:** Tasks 1–2.

## 8. Testing

### Unit tests

- Prefix detection with all five highlighters.
- Representative MySQL, Oracle, and SQL Server comments/quoted regions plus
  current PostgreSQL cases.
- Delimiter escaping and Enter/Tab output for all five engines.
- Existing maximum-match, navigation, overlay, Escape, and loading tests.

### Integration tests

- A root `Model.Update` key-flow test wires the selected engine,
  `rawQueryHighlighter`, navigator table source, and completion state together
  for all five engines.
- A live database integration test is inapplicable: no adapter or database I/O
  changes.

## 9. Acceptance criteria

- A matching loaded table completes after `SELECT * FROM ` for each engine.
- Acceptance emits `"Album"` for PostgreSQL/Oracle/SQLite, `` `Album` `` for
  MySQL, and `[Album]` for SQL Server.
- Closing delimiters in catalog names escape correctly.
- Completion stays hidden in unavailable/invalid states.
- `scripts/validate.sh` passes.

## 10. Risks & open items

- Active-schema base tables only is intentional; cross-schema completion needs
  broader metadata and display disambiguation.
- Trigger positions remain deliberately narrow; this change does not add SQL
  parsing.
- **Open items:** None.

