# Spec: MySQL query keyword highlighting

## Objective

Add live, keyword-only syntax highlighting to the raw-query editor for active
MySQL sessions. This restores parity with PostgreSQL highlighting and makes
MySQL queries easier to read without changing how they are entered or run.

## Tech Stack

- Go 1.26
- Bubble Tea v2 and Lipgloss v2
- Testify
- Existing internal `app/sqlhighlight` package
- MySQL 8.4 dialect rules, matching the local Compose fixture

## Commands

- Format: `gofmt -w <changed-go-files>`
- Verify: `scripts/validate.sh`
- Build: `go build ./...`
- Run local fixtures: `docker compose up -d`

## Project Structure

- `internal/app/sqlhighlight/scanner.go` owns shared, private rune traversal,
  keyword-span creation, and common lexical helpers.
- Dialect files own only their distinct lexical exclusions and keyword
  vocabularies; they reuse the shared scanner.
- `internal/app/` chooses the highlighter from the active database engine and
  applies ANSI styles to the raw-query editor.
- `internal/app/*_test.go` covers editor integration and selection rendering.
- `internal/app/sqlhighlight/*_test.go` covers MySQL lexical behavior.
- `docs/specs/` stores this specification.

## Code Style

Use a concrete, stateless highlighter with no database or UI dependency:

```go
// MySQL finds supported MySQL keyword spans in SQL source text.
type MySQL struct{}

var _ Highlighter = MySQL{}

func (MySQL) KeywordSpans(value string) []Span {
	// Return source-rune spans only; rendering remains in app.
}
```

Preserve raw rune positions. Selection rendering must continue to replace
keyword styling for selected cells.

## Testing Strategy

Use table-driven Testify tests to prove:

- MySQL reserved words are highlighted case-insensitively.
- Identifier lookalikes are not highlighted.
- Strings, backtick-quoted identifiers, `#` comments, valid `-- ` comments,
  and block comments are not highlighted.
- Reserved words after compact or whitespace-separated qualifier periods are
  treated as identifiers and are not highlighted.
- Escaped and doubled quotes/backticks preserve lexical boundaries.
- The raw-query view highlights PostgreSQL and MySQL only.
- Selection styling and ANSI-stripped editor content remain unchanged.

## Boundaries

- Always: preserve PostgreSQL behavior, use semantic existing colors, add
  focused automated tests, and run `scripts/validate.sh` once after completion.
- Ask first: add dependencies, add configuration, change database interfaces,
  change the query execution flow, or alter the accepted PostgreSQL behavior.
- Never: implement validation, autocomplete, formatting, diagnostics, or
  highlighting for another engine as part of this feature.

## Success Criteria

- A connected MySQL raw-query editor highlights complete MySQL 8.4 reserved
  keyword tokens with the existing SQL-keyword color.
- Keywords within strings, backtick identifiers, or comments stay uncolored.
- A reserved word following a MySQL qualifier period stays uncolored, including
  when whitespace surrounds the period.
- MySQL-specific lexical rules handle backtick escaping, backslash string
  escapes, `#` comments, and MySQL's whitespace-required `-- ` comments.
- PostgreSQL highlighting is unchanged.
- Disconnected, Oracle, SQLite, and SQL Server editors remain plain.
- Existing selection behavior still takes visual priority over keyword color.
- No dependencies, configuration settings, adapter imports, or `db.Database`
  interface changes are introduced.

## Open Questions

None. The keyword list is intentionally static and based on MySQL 8.4.
