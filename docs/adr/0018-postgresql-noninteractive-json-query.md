# ADR 0018: Run PostgreSQL SELECT queries as non-interactive JSON

## Status

Accepted on 2026-09-07.

## Context

db-tui is primarily an interactive terminal client. Scripts and shell
pipelines also need a small, predictable way to run a PostgreSQL query through
a caller-provided DSN and consume its rows as JSON, without starting Bubble Tea
or mixing terminal UI output into machine-readable output.

The initial scope is intentionally narrow: PostgreSQL only, a caller-provided
DSN, and a simple read-query policy. It must not silently emit a
partial JSON document when a query fails after returning some rows.

## Decision

`db-tui -q '<SQL>' -c '<PostgreSQL-DSN>'` will run a query without starting the
TUI, then close the database session and terminate. For example:

```sh
db-tui -q 'SELECT * FROM users' -c 'postgres://user:password@host:5432/dbname?sslmode=require'
```

`-c` supplies the PostgreSQL DSN and is required with `-q`. The mode constructs
only the PostgreSQL adapter; other database engines are unsupported in the
initial release. The command must not log the DSN or include it in an error
message, because it can contain credentials.

The command accepts one `SELECT` statement and permits one optional trailing
semicolon followed only by whitespace. This is a deliberately simple lexical
restriction. The command will not use PostgreSQL read-only transaction mode or
a full SQL parser in this release.

On success, stdout contains one JSON array of row objects followed by a
newline. Each object uses its database column label as the key. SQL `NULL`
becomes JSON `null`. Timestamps, binary values, and precision-sensitive decimal
values use the stable JSON encodings established by ADR 0002; booleans and
ordinary numeric values remain JSON booleans and numbers. Duplicate column
labels are an error, because an object cannot represent them without losing a
value.

The command fully materializes its JSON result in memory and writes stdout only
after successful completion. It does not stream rows to stdout and does not use
a temporary spool file. This keeps successful output valid JSON and ensures
failures leave stdout empty.

Errors, including database or server-configured timeout errors, are written to
stderr. The command exits non-zero on every failure. This change introduces no
new client-side query deadline.

## Consequences

### Positive

- Shell pipelines receive JSON only on stdout.
- A failed query cannot leave a partial JSON array on stdout.
- Calling the command never initializes the interactive Bubble Tea program.
- An explicit DSN makes automation independent of saved connection
  configuration and the TUI selection.
- The PostgreSQL-only scope keeps the first adapter path focused.

### Negative

- Large result sets consume memory proportional to the complete JSON document.
- The lexical `SELECT` restriction is not a complete read-only security
  boundary: a `SELECT` can invoke a database function with side effects.
- The mode has no application-defined deadline; long-running queries rely on
  cancellation or database/server timeout configuration.
- A DSN supplied on the command line can be visible in shell history and the
  process list when it contains credentials.
- Users cannot use this mode with non-PostgreSQL DSNs yet.

## Alternatives considered

### Stream the JSON array directly to stdout

Rejected because a later query error or timeout would leave a syntactically
invalid, partial JSON document in a pipeline.

### Spool completed JSON to a temporary file

Rejected for the initial release. It would avoid the memory cost while
preserving all-or-nothing stdout, but adds filesystem lifecycle and permission
handling beyond the requested simple implementation.

### PostgreSQL read-only transactions and full SQL parsing

Deferred. They would provide stronger read-only guarantees but exceed the
initial simple `SELECT`-only policy.
