# db-tui Architecture

Read this file before planning or changing code. It defines package
boundaries, runtime flow, and shared engineering conventions. `AGENTS.md`
defines the contribution workflow

## Core principles

- Keep dependencies explicit and directed inward.
- Use one root Bubble Tea model and one event loop.
- Keep application behavior independent of database engines.
- Bound and cancel database operations.
- Give each state invariant one clear owner.
- Add abstractions only for real consumers.
- Test every behavior change at the lowest practical layer.

## Package boundaries

```text
cmd/db-tui
  |-- internal/app ------> internal/db
  |       |-- internal/config
  |       `-- internal/version
  `-- internal/db/{postgres,mysql,oracle,sqlite}
          |-- internal/db
          |-- internal/{csvexport,jsonexport}
          `-- internal/logger
```

- `cmd/db-tui` is the composition root: load configuration, select an adapter,
  construct the model, run Bubble Tea, and close the final database session.
- `internal/app` owns TUI state, messages, commands, updates, layout, views,
  panels, and modals. It must not import a database adapter.
- `internal/db` owns engine-neutral types and the `Database` interface. It
  must not depend on the application or an adapter.
- Adapters own driver, catalog, SQL dialect, DSN, export, and dump behavior.
  They implement `db.Database` and must not import `internal/app`.
- Config, export, logging, and version packages remain focused support code.
- `docker/` contains local fixtures, not production logic.
- Do not expand `utils/` when code has a clearer owner.

## Bubble Tea flow

`app.Model` is the only application-level `tea.Model`.

```text
Init -> tea.Cmd performs I/O -> typed tea.Msg
     -> Update validates and mutates state -> View renders state
```

`Update` and `View` perform no database, filesystem, network, process, or
blocking I/O. Return a `tea.Cmd` for the work and a typed result message. Views
must not mutate state or start work.

The root coordinates shared state and input. Navigator, data, query, function,
and modal types own local invariants but never access the database directly.

Process messages in this order:

1. lifecycle and asynchronous results;
2. active modal or overlay;
3. global input; and
4. focused keyboard or mouse input.

`appLayout` owns terminal geometry, visible rows, and mouse hit testing.
Recalculate it on resize and clamp affected selections and offsets.

## Asynchronous work

Results must carry enough identity to reject stale work:

- `connectionAttempt` for connection attempts;
- `session` for the active database session; and
- feature request counters for work within a session.

Apply results only when their identities still match current state. Close a
stale successful connection instead of adopting or leaking it.

Database methods accept `context.Context` first. Commands use bounded contexts
and call their cancel functions. Close rows, files, logs, pools, and processes
at the layer that acquires them.

## Database contract

`internal/db.Database` is the application-to-engine boundary. Never expose
driver-specific types through it. Changing it requires neutral types, all four
adapter implementations, updated fakes and callers, and focused tests.

- Keep interactive rows and query results bounded.
- Validate page offsets and limits before building SQL.
- Preserve column/value order and represent SQL `NULL` as `nil`.
- Validate and engine-quote identifiers; bind data values as parameters.
- Test empty, mixed-case, reserved-word, and malicious identifiers.
- Require the complete primary key for updates and deletes through
  `db.ValidatePrimaryKeyWhere`; display position is not row identity.

## UI text and colors

- Do not hardcode raw colors in views or update logic.
- Define colors in `internal/app/colors.go` with semantic names such as
  `colorError`, `colorAccent`, or `colorModalBackground`.
- Do not duplicate user-visible text, labels, help text, or status messages.
- Keep shared text in named constants or focused helpers in `internal/app`.
- Feature-specific text may remain with its feature when used once.
- Tests should reference shared text definitions where practical.

## Safety and Go conventions

- Never commit credentials, local config, logs, dumps, or private data.
- Keep config/log directories `0700` and sensitive files `0600`.
- Credentials are plaintext; do not imply encryption.
- Sanitize database text and errors before terminal rendering.
- Restrict export names to one safe filename component.
- Never log passwords or complete DSNs.
- Require confirmation for destructive operations.
- Use `gofmt`, standard Go names, and consistent initialisms.
- Document exported declarations and non-obvious intent.
- Handle errors explicitly and wrap propagated errors with context and `%w`.
- Do not use `panic` for normal failures.
- Prefer concrete types until a real boundary requires an interface.
- Avoid mutable package globals for application state.

## Testing and verification

Every behavior change needs automated tests. Prefer table-driven subtests with
`testify/assert` or `testify/require`.

- Test validation and conversion in the owning package.
- Test model transitions, stale results, focus, layout, and rendering in
  `internal/app` with a fake database.
- Test SQL, quoting, scanning, cancellation, and engine behavior in adapters.
- Use only local, reproducible fixtures for integration tests.
- Cover paging, identifiers, cancellation, SQL `NULL`, primary keys,
  permissions, terminal edges, and adapter parity where relevant.

Implement the approved change and tests before running verification once:

```sh
scripts/validate.sh
```

After dependency changes, also run `go mod tidy`, `go mod verify`, and
`go build ./...`.
