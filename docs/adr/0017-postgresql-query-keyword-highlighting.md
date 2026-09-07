# ADR 0017: Render a PostgreSQL keyword set in the raw-query editor

## Status

Accepted on 2026-09-07.

## Context

The raw-query panel currently renders all editor text in one color. Users want
live SQL syntax assistance, but the first release must be limited to PostgreSQL
and common keywords so it does not become a parser, a dependency addition, or a
cross-engine abstraction.

The Bubbles textarea supports whole-widget styles only; it has no token-span
styling API. The query editor also has an application-owned mouse selection
layer that maps raw source runes to terminal cells and deliberately renders
selected cells with a uniform selection style.

## Decision

Use a local PostgreSQL-aware lexical scanner during raw-query editor rendering.
It colors the agreed common query words and every PostgreSQL 18 reserved word
missing from that initial set with the semantic `colorSQLKeyword` color. Do not
add PostgreSQL words that may be identifiers: that would color user-defined
table or column names without understanding SQL context.

Keep SQL-only scanning in `internal/app/sqlhighlight`. Its narrow
`Highlighter` interface returns source spans, and `PostgreSQL` is the only
implementation in this release. The `app` package owns ANSI insertion and
terminal-cell mapping, so the scanner does not depend on Bubble Tea selection
or rendering types. New dialects require an explicit implementation and app
engine gate; this is not a dialect registry or automatic multi-engine feature.

Highlighting is automatic when, and only when, the active database session's
engine is PostgreSQL. It applies only to editable raw-query input. The scanner
matches complete identifier tokens case-insensitively and avoids strings,
quoted identifiers, comments, and dollar-quoted text. It does not validate,
parse, or semantically analyze SQL.

Render ANSI keyword spans before applying the existing selection overlay. The
selection overlay continues to replace the selected cells' keyword colors so a
selection has consistent contrast and copied SQL remains plain text.

## Consequences

- PostgreSQL queries gain useful live visual structure with no new dependency.
- Other engines and disconnected sessions remain visually unchanged.
- The scanner remains small, deterministic, and unit-testable, but its keyword
  list must be expanded deliberately as the feature grows.
- Rendering must preserve source rune positions and terminal widths so the
  existing selection behavior remains correct.
- Future multi-dialect highlighting should add a separate deliberate design;
  this decision does not create a dialect-plugin abstraction prematurely.

## Alternatives considered

### Add a full SQL lexer or syntax-highlighting dependency

Rejected for the first release. The required keyword-only behavior does not
justify a new dependency or the complexity of adapting it to the textarea's
whole-widget rendering model.

### Use a generic cross-engine keyword set

Rejected because the requested behavior is PostgreSQL-only. A generic set would
blur dialect ownership and imply unsupported engine behavior.

### Color keyword-shaped text with a regular expression

Rejected because it would falsely color content inside strings, quoted
identifiers, comments, and PostgreSQL dollar-quoted text.

### Preserve keyword colors inside a text selection

Rejected because the established selection style intentionally takes visual
priority and provides consistent readability.
