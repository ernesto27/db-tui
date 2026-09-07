# Spec: PostgreSQL query keyword highlighting

## Objective

Make common PostgreSQL SQL keywords visually distinct while users edit SQL in
the raw-query input. Highlighting is live, automatic, and deliberately narrow:
it identifies keywords only. It is neither a SQL parser nor a query validator.

## Scope

- Highlight only the editable input of the raw-query panel.
- Enable it only while the active database session reports the PostgreSQL
  engine.
- Leave the editor plain for every other engine and when no database session is
  active.
- Highlight case-insensitively, but only complete standalone keyword tokens.
- Leave quoted strings, quoted identifiers, line comments, block comments, and
  PostgreSQL dollar-quoted text uncolored.
- Let the existing text-selection style replace keyword colors in selected
  cells.

## Initial vocabulary

The first release recognizes these keywords:

```text
SELECT FROM WHERE JOIN INNER LEFT RIGHT FULL ON AS
INSERT INTO VALUES UPDATE SET DELETE
CREATE ALTER DROP TABLE
ORDER BY GROUP HAVING LIMIT OFFSET UNION ALL DISTINCT
AND OR NOT NULL CASE WHEN THEN ELSE END
```

The vocabulary is intentionally local and static. Adding a full PostgreSQL
keyword catalog, semantic analysis, schema-aware highlighting, or dialects for
other engines is out of scope.

## Design

`queryModel.editorView` is the rendering seam. The Bubbles textarea applies one
style to an entire editor line and cannot style token spans directly. The app
therefore scans the editor's source text while rendering and adds ANSI styling
to keyword spans before the existing selection overlay runs.

The scanner must preserve every source rune and newline. Its only output is the
set of keyword spans to color. It recognizes enough PostgreSQL lexical state to
exclude:

- single-quoted strings, including doubled single-quote escapes;
- double-quoted identifiers, including doubled double-quote escapes;
- `--` line comments;
- nested `/* ... */` block comments; and
- `$tag$ ... $tag$` and `$$ ... $$` dollar-quoted strings.

The scanner treats a keyword as a sequence of identifier characters bounded by
non-identifier characters. Consequently `select`, `Select`, and `SELECT`
highlight, while `selection` and `my_select` do not.

ANSI insertion must not participate in mouse-coordinate or source-index
mapping. Those calculations continue to use the raw textarea value. When text
is selected, the existing selection renderer strips ANSI from the selected
cells and applies `textSelectionStyle`, intentionally making selection the
visual priority.

## Structure

- `internal/app/query_highlighting.go` will own the PostgreSQL lexical scanner,
  keyword set, and pure ANSI-rendering helper.
- `internal/app/query_panel.go` will choose highlighted versus plain editor
  rendering from the active `db.Database` engine.
- `internal/app/query_selection.go` will preserve its current selection mapping
  and run after highlighted rendering.
- `internal/app/colors.go` will define `colorSQLKeyword` as a semantic color.

No adapter import, `db.Database` interface change, configuration setting, or
new dependency is required.

## Tests and verification

Application tests must prove that:

- supported keywords receive ANSI styling in PostgreSQL editor rendering;
- matching is case-insensitive and respects identifier boundaries;
- lookalikes in strings, quoted identifiers, comments, and dollar-quoted text
  are not styled;
- non-PostgreSQL and disconnected editor rendering remains plain;
- selection retains its existing ANSI-stripped selected-cell presentation and
  the plain-text selection value is unchanged.

Run `gofmt` on changed Go files and `scripts/validate.sh` once after the
implementation and tests are complete.

## Out of scope

- Coloring literals, identifiers, operators, functions, or result data.
- SQL completion, parsing, formatting, linting, or validation.
- Configurable colors, a user toggle, or persistent highlighting preferences.
- Highlighting for MySQL, Oracle, SQLite, SQL Server, or a disconnected editor.
