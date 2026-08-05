# ADR 0008: Defer Oracle database dumps in the initial engine

## Status

Accepted on 2026-08-05.

## Context

db-tui's `Database` interface exposes a dump operation. Oracle's native export
workflow uses Data Pump tools such as `expdp`, which are external executables
with their own installation, directory-object, privilege, and credential
requirements. Those requirements are not present in the local Oracle container
or in the current application distribution.

## Decision

The initial Oracle engine will support connection, relation discovery, paged row
browsing, raw SQL, table metadata, and table DDL. Its dump method will return a
clear unsupported-operation error instead of invoking Data Pump or attempting a
partial substitute.

## Consequences

- Oracle support can be built and tested without requiring Oracle client tools.
- The UI retains a consistent command surface and reports the unavailable
  capability explicitly.
- A future Oracle export feature requires a dedicated design for Data Pump
  executable discovery, connection parameters, permissions, and output paths.

## Alternatives considered

### Invoke `expdp` now

Rejected because it would make a normal db-tui build depend on external Oracle
client tooling and environment-specific database privileges.

### Omit Oracle from the existing dump command

Rejected because the engine interface requires an implementation and an explicit
error is clearer than silently hiding behavior by engine.
