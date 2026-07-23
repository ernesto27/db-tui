# Repository Guidelines

## Project Structure & Module Organization

`db-tui` is a Go terminal database client built with Bubble Tea. Keep production code within these package boundaries:

- `cmd/db-tui/`: executable wiring and program startup only.
- `internal/app/`: root Bubble Tea model, messages, updates, and terminal views.
- `internal/db/`: driver-neutral types and the `Database` interface.
- `internal/db/postgres/`: pgx-backed PostgreSQL connection, introspection, and row retrieval.
- `internal/querylog/`: safe, synchronized SQL query logging.
- `docker/postgres/init/`: Chinook demo-database initialization SQL.
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

## Commit & Pull Request Guidelines

Recent commits use concise imperative subjects (for example, `add basic view of tables`). Keep each commit scoped to one task. Pull requests should state intent, link relevant work, list verification commands, and include terminal screenshots for visible UI changes. Never commit passwords, credential-bearing DSNs, local config, logs, or generated binaries.
