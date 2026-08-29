# Repository Guidelines

## Architecture Reference

Before planning or changing code, read `ARCHITECTURE.md`.
Treat it as the source of truth for package boundaries, runtime flow, state
ownership, database-adapter contracts, and shared engineering conventions.
When a change alters those architectural rules, update `ARCHITECTURE.md` in
the same approved change.

## Project Structure & Module Organization

`db-tui` is a Go terminal database client built with Bubble Tea. Keep production code within these package boundaries:

- `cmd/db-tui/`: executable wiring and program startup only.
- `internal/app/`: root Bubble Tea model, messages, updates, and terminal views.
- `internal/db/`: driver-neutral types and the `Database` interface.
- `internal/db/{postgres,mysql,oracle,sqlite}/`: engine-specific connection,
  introspection, query, export, and row-operation adapters.
- `internal/config/`: private per-user connection and application settings.
- `internal/{csvexport,jsonexport}/`: engine-neutral export serialization.
- `internal/logger/`: safe, synchronized SQL and application logging.
- `internal/version/`: embedded application version metadata.
- `docker/`: local, reproducible database fixtures and initialization data.
- `scripts/validate.sh`: repository formatting, vetting, and test checks.

Keep dependencies directed inward: `cmd` may wire concrete dependencies; `app` depends on `db`; database adapters implement `db` and must not import `app`. Add reusable UI code only when it has a real consumer.

## Build, Test, and Development Commands

- `docker compose up -d`: start the local Chinook PostgreSQL service on `127.0.0.1:5433`.
- `go run ./cmd/db-tui`: run the terminal client against that service.
- `go test ./...`: run all tests; PostgreSQL integration tests require the Compose service.
- `scripts/validate.sh`: check formatting, run `go vet`, tests, and race tests.
- `go build ./...`: compile every package. Run `go mod tidy && go mod verify` after dependency changes.

## Coding Style & Naming Conventions

Use `gofmt`; the formatter determines indentation. Keep package names short, lowercase, and singular; name tests `TestBehavior` in `*_test.go` files. Exported identifiers need Go doc comments. Keep Bubble Tea `Update` methods free of I/O: use `tea.Cmd` values that return typed messages. Database methods accept `context.Context`, wrap propagated errors with `%w`, and keep result pages bounded.

## Testing Guidelines

Every new feature must include automated tests that verify its expected behavior. A feature is not complete until its tests are added and passing. Use Go's `testing` package, `testify/assert`, and table-driven subtests for varied inputs. Add focused regression coverage at the lowest practical layer, especially for paging bounds, identifier quoting, cancellation, layout edges, SQL `NULL`, and query-log behavior. No numeric coverage threshold is set, but every behavior change needs relevant tests. Do not make integration tests depend on remote databases or credentials.

Implement the requested change before running automated checks. Do not use test-first development or run incremental test commands unless the user explicitly asks; run the relevant automated verification once at the end. The user performs manual testing.

## Change Approval

Before making any edit or addition, show the proposed changes as a preview and wait for the user's explicit approval.

## Commit & Pull Request Guidelines

Recent commits use concise imperative subjects (for example, `add basic view of tables`). Keep each commit scoped to one task. Pull requests should state intent, link relevant work, list verification commands, and include terminal screenshots for visible UI changes. Never commit passwords, credential-bearing DSNs, local config, logs, or generated binaries.

## Specifications

Save specifications and design decisions as Markdown files in this repository. Do not publish them to GitHub issues, pull requests, or an external issue tracker unless the user explicitly asks.


IMPORTANT do not use superpowers plugin as default, do not use git worktree or commit

<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them, including dynamic-dispatch hops grep can't follow. Name a file or symbol in the query to read its current line-numbered source. If it's listed but deferred, load it by name via tool search.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` prints the same output.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
<!-- CODEGRAPH_END -->
