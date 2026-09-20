# Implementation Plan: Cobra CLI Foundation

## Overview

Replace the custom standard-library flag parser in `cmd/db-tui/main.go` with a
Cobra v1.10.2 command hierarchy. Preserve interactive startup at the root and
move non-interactive queries to `db-tui query` with command-local flags.

## Architecture Decisions

- Keep the Cobra root-command factory in the `cmd/db-tui` composition root and
  split command concerns into focused files.
- Define `query`-local `query`, `dsn`, and `format` flags with short aliases
  `q`, `c`, and `t`.
- Use Cobra's `RunE`, `NoArgs`, and paired-flag validation in the query command.
- Inject package-local callbacks so tests do not start a terminal program or
  access a database.
- Retain exit code 2 for usage errors and 1 for query/runtime errors.

## Source Basis

- `RunE` error handling:
  https://github.com/spf13/cobra/blob/v1.10.2/site/content/user_guide.md#returning-and-handling-errors
- Local short/long flags and flag groups:
  https://github.com/spf13/cobra/blob/v1.10.2/site/content/user_guide.md#working-with-flags
- Positional-argument validation:
  https://github.com/spf13/cobra/blob/v1.10.2/site/content/user_guide.md#positional-and-custom-arguments

## Task

Implement the Cobra command factory, add Cobra to the module files, and cover
the interactive root and query paths, command-local flags, invalid input, help,
error classes, and output newline behavior in `cmd/db-tui/main_test.go`.

## Verification

Run once after implementation:

```sh
go mod tidy
go mod verify
go build ./...
scripts/validate.sh
```
