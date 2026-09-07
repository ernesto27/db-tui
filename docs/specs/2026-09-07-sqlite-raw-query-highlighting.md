# Spec: SQLite Raw Query Highlighting

## Objective

Add SQLite-aware SQL keyword highlighting to the Raw Query editor. When the
active connection uses SQLite, the editor highlights SQLite keywords using the
same keyword color and rendering path already used for PostgreSQL and MySQL.

A SQLite user can distinguish SQL structure from identifiers and values while
editing a raw query. Existing PostgreSQL and MySQL highlighting must remain
unchanged.

## Tech Stack

- Go
- Bubble Tea v2 and Lipgloss v2
- Existing `internal/app/sqlhighlight` package
- SQLite adapter identified by `db.EngineSQLite`

## Commands

Build:

```sh
go build ./...
```

Test:

```sh
go test ./...
```

Repository validation:

```sh
scripts/validate.sh
```

## Project Structure

```text
internal/app/view.go
  Selects the Raw Query highlighter from the connected database engine.

internal/app/sqlhighlight/sqlite.go
  Defines the SQLite highlighter, SQLite keyword vocabulary, and SQLite lexical
  region handling.

internal/app/sqlhighlight/sqlite_test.go
  Verifies SQLite keyword spans and lexical exclusions.

internal/app/query_highlighting_test.go
  Verifies SQLite is enabled through the application rendering path.
```

## Code Style

Follow the existing concrete highlighter pattern:

```go
type SQLite struct{}

var _ Highlighter = SQLite{}

func (SQLite) KeywordSpans(value string) []Span {
	return keywordSpans(value, sqliteKeywords, skipSQLiteRegion, nil)
}
```

Use `gofmt`, exported Go-doc comments, table-driven tests, rune-indexed spans,
and the shared scanner helpers where their behavior matches SQLite.

## Testing Strategy

- Unit-test `SQLite.KeywordSpans` with case-insensitive SQLite keywords.
- Verify no partial identifier matches occur.
- Verify strings, double-quoted identifiers, backtick identifiers,
  bracket-quoted identifiers, line comments, and block comments are skipped.
- Verify scanning resumes after each skipped lexical region.
- Extend the app rendering test so a SQLite connection renders `SELECT` with
  `colorSQLKeyword`.
- Run `scripts/validate.sh` once after implementation and tests are complete.

## Boundaries

- Always:
  - Preserve existing PostgreSQL and MySQL behavior.
  - Keep highlighting purely in `internal/app` and independent of adapters.
  - Add automated tests for every new SQLite behavior.
- Ask first:
  - Adding dependencies.
  - Changing the database interface or adapter contracts.
  - Changing SQLite keyword policy beyond the documented SQLite vocabulary.
- Never:
  - Highlight text inside SQLite strings, identifiers, or comments.
  - Modify database execution behavior.
  - Commit secrets, generated binaries, or unrelated changes.

## Success Criteria

- A connected SQLite Raw Query editor highlights recognized SQLite keywords.
- SQLite highlighting uses the existing SQL keyword visual style.
- SQLite strings, quoted identifiers, and comments do not produce keyword spans.
- PostgreSQL, MySQL, unsupported engines, and disconnected state retain their
  current highlighting behavior.
- Focused unit and application rendering tests pass.
- `scripts/validate.sh` passes.

## Open Questions

None. The SQLite keyword vocabulary will follow SQLite's documented keyword
list, with SQLite's lexical handling for quotes and comments.
