# Spec: Use saved connections from the CLI

## Objective

Let a person at a terminal run `query` or `dump` using the name of a connection
already saved by the TUI. The CLI reads the current user's
`~/.config/db-tui/config.json` and connects with the same engine and settings
the TUI would use. This avoids copying a DSN into each command.

## Commands

```sh
db-tui query --connection reporting -f report.sql
db-tui query --connection reporting -q 'SELECT 1' -t csv
db-tui dump --connection reporting
```

The option is `--connection` on `query` and `dump`, with no short alias. Existing
DSN options remain: `query -c/--dsn` and `dump -d/--dsn`. When both a nonempty
DSN and a saved name are supplied, the DSN wins; the CLI does not read or
validate the saved connection. Query text/file, output format, read-only
execution, dump format, and dump destination remain unchanged.

## Saved connection resolution

1. Read the current user's normal config path only when `--connection` is
   selected. There is no custom config-path option.
2. Match `Connection.Name` exactly and case-sensitively. If names repeat, use
   the first matching entry in file order.
3. Use `Connection.Engine` to choose the adapter. This matters for explicitly
   saved DSNs that cannot be identified by the CLI's DSN-prefix detector.
4. If `Settings.DSN` is nonblank, use that DSN. Otherwise construct the DSN
   from the saved host, port, database, username, and password exactly as the
   TUI does. SQLite uses the saved file path in `Settings.DSN`.
5. The connection's `Status` and `Environment` metadata do not change
   selection or execution.

The shared saved-settings conversion must have one owner so CLI and TUI
behavior cannot drift. The CLI must not import `internal/app` for this purpose.

## Errors and side effects

- A blank saved name, absent source (`--dsn` and `--connection` both omitted),
  or unknown saved name is a usage error (exit code 2). An unknown-name error
  identifies the requested name but does not list other saved names.
- If `--connection` is selected and the config file is absent, unreadable, or
  malformed, report a runtime error (exit code 1). Reading from the CLI must
  not create an empty config file or directory.
- Invalid settings in a selected saved entry produce a runtime error before
  opening a database connection. Error text must not contain the password or
  full credential-bearing DSN.
- Existing DSN flag validation and connection errors keep their current
  behavior. An empty DSN is treated as absent.

## Scope

In scope: saved-name lookup for both subcommands, shared connection-setting
conversion, and help/README updates. Out of scope:
creating, editing, deleting, listing, or renaming saved connections from the
CLI; changing config schema; changing adapters or the `db.Database` interface;
new output formats; and a custom config path.

## Acceptance criteria

1. Both subcommands work with a unique saved name, including entries saved
   from form fields and entries saved with an explicit DSN.
2. Case-sensitive exact lookup and first-match behavior are verified.
3. Both subcommands retain DSN support, and a nonempty DSN wins when both
   options appear.
4. Missing, malformed, or invalid config entries fail without connecting or
   exposing credentials. A missing config file is not created.
5. Existing query and dump behavior remains unchanged apart from the new
   connection source. The existing validation suite passes. Focused tests
   cover saved connection resolution and credential-safe failures.

## Resolved decisions

| Question | Decision |
| --- | --- |
| Which commands? | `query` and `dump` |
| Which config? | Current user's `~/.config/db-tui/config.json` |
| Which saved entry forms? | Explicit DSN and TUI form fields |
| Name matching? | Exact and case-sensitive |
| Duplicate names? | First matching entry |
| DSN and saved name together? | Nonempty DSN wins |
| Missing config? | Error without creating a file (default chosen after request to stop questions) |
| Unknown name error? | Requested name only (default chosen after request to stop questions) |
| Short alias? | None (default chosen after request to stop questions) |
