# Repository Guidelines

## Project Structure & Module Organization

This Go module (`github.com/ernestoponce27/db-tui`) is in its bootstrap stage. Root files contain project documentation and dependency smoke tests. Follow the planned package boundaries as implementation grows:

- `cmd/db-tui/`: executable entry point only.
- `internal/app/`: root Bubble Tea model, messages, updates, and views.
- `internal/db/`: driver-neutral database types and session interfaces.
- `internal/db/postgres/`: PostgreSQL-specific connection and introspection code.
- `internal/config/`: settings and connection profiles.
- `internal/ui/`: real, reusable UI components; do not add placeholder packages.

Keep dependency direction `cmd -> app -> db/config`; database and configuration packages must not import the application.

## Build, Test, and Development Commands

- `go test ./...`: run the complete test suite.
- `go test -race ./...`: detect data races in asynchronous UI/database code.
- `go vet ./...`: run standard static analysis.
- `gofmt -w .`: format all Go source files before review.
- `test -z "$(gofmt -l .)"`: verify formatting without changing files.
- `go mod tidy && go mod verify`: normalize and verify module dependencies.
- `go build ./...`: compile all packages. Once the command package exists, use `go run ./cmd/db-tui` for local execution.

## Coding Style & Naming Conventions

Use standard Go formatting and idioms. Package names should be short, lowercase, and singular. Exported identifiers require clear Go doc comments. Name tests `TestBehavior` and files `*_test.go`. Keep Bubble Tea `Update` methods pure: represent database and filesystem work as asynchronous commands returning typed messages. Accept `context.Context` for database calls, preserve wrapped errors with `%w`, and bound retained query results.

## Testing Guidelines

Use Go's `testing` package and table-driven tests where inputs vary. Add tests at the lowest practical package layer, especially for layout edge cases, cancellation, result truncation, DSN redaction, and SQL `NULL` handling. No numeric coverage threshold is established; every behavior change should include focused regression coverage.

## Commit & Pull Request Guidelines

The repository has no commit history yet. Use concise, imperative subjects, optionally prefixed with a backlog ID (for example, `T-003: add package skeleton`). Keep commits scoped to one task. Pull requests should explain intent, list verification commands, link relevant issues/tasks, and include terminal screenshots for visible UI changes. Never commit passwords, credential-bearing DSNs, local config, logs, or generated binaries.
