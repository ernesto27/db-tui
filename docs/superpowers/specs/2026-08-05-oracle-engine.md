# Oracle engine v1

## Problem Statement

Users who work with Oracle Database cannot use db-tui for their normal database
browsing workflow. They can connect with external clients such as DBeaver, but
db-tui does not recognize Oracle as an engine, cannot create an Oracle
connection from the connection modal, and has no implementation of the
driver-neutral database operations for Oracle schemas.

## Solution

Add Oracle as a first-class db-tui engine. A user can select Oracle in the
connection modal or provide an Oracle Easy Connect DSN, connect to an Oracle
service such as the local `FREEPDB1` service, browse objects visible to the
current user, inspect table metadata and structural DDL, view bounded pages of
rows, execute raw SQL, and use the existing table and query export workflows.

The engine uses the pure-Go `go-ora/v2` database/sql driver so normal db-tui
builds do not require Oracle Instant Client or CGO. Oracle Data Pump/database
dumps remain an explicit unsupported-operation placeholder in this release, as
defined by ADR 0008.

## User Stories

1. As an Oracle developer, I want to select Oracle when creating a connection, so that I can connect without manually editing configuration files.
2. As an Oracle developer, I want the connection modal to default to port 1521, so that local and conventional Oracle connections require less manual entry.
3. As an Oracle developer, I want to enter a host, service name, port, username, and password, so that I can connect to a standard Easy Connect endpoint.
4. As an Oracle developer, I want to provide an Oracle-specific DSN, so that saved or advanced connection strings remain usable.
5. As an Oracle developer, I want connection failures to preserve my entered settings, so that I can correct service-name, credentials, or network errors without re-entering everything.
6. As an Oracle developer, I want the connected service name and host shown consistently by db-tui, so that I know which database session I am using.
7. As an Oracle developer, I want to browse the base tables visible to my current user, so that I can find data without knowing every table name beforehand.
8. As an Oracle developer, I want to browse ordinary views separately from tables, so that I understand which relations are query-backed.
9. As an Oracle developer, I want to browse materialized views separately from ordinary views, so that stored query results are distinguishable from base tables.
10. As an Oracle developer, I want relation lists to be deterministic and alphabetically ordered, so that navigator selection and filtering are predictable.
11. As an Oracle developer, I want to open a table, view, or materialized view and receive a bounded page of rows, so that large relations do not overwhelm the terminal or database.
12. As an Oracle developer, I want next-page detection to remain accurate, so that pagination controls neither stop early nor request an unbounded result set.
13. As an Oracle developer, I want SQL `NULL` values to remain distinguishable from empty strings, so that the row viewer and exports preserve data meaning.
14. As an Oracle developer, I want column inspection to show name, position, data type, identity status, nullability, defaults where available, and comments, so that I can understand a table before querying it.
15. As an Oracle developer, I want index inspection to list every indexed column and Oracle index type, so that I can understand access paths and multi-column indexes.
16. As an Oracle developer, I want table DDL displayed in the existing read-only DDL modal, so that I can inspect the server-defined structure without leaving db-tui.
17. As an Oracle developer, I want raw SQL execution to return bounded result rows or a command status, so that ad hoc work behaves consistently across supported engines.
18. As an Oracle developer, I want raw-query cancellation and connection errors to surface as actionable messages, so that long or invalid statements do not leave the TUI in an ambiguous state.
19. As an Oracle developer, I want CSV and JSON table exports to work through the existing export commands, so that I can extract data using the same workflow as other engines.
20. As an Oracle developer, I want query export to retain the existing SELECT-only safety rule, so that I do not accidentally export the result of a mutating statement.
21. As an Oracle developer, I want the dump command to state clearly that Oracle Data Pump is unsupported in v1, so that I do not mistake a missing external tool for a successful backup.
22. As a db-tui maintainer, I want Oracle operations to honor contexts and emit typed, wrapped errors, so that the application remains responsive and failures are diagnosable.
23. As a db-tui maintainer, I want Oracle support to compile without Oracle client libraries, so that contributors can build and test the project on a standard Go installation.
24. As a db-tui maintainer, I want the local Oracle Compose service to be the integration-test fixture, so that tests use deterministic demo data and no remote credentials.

## Implementation Decisions

- Introduce an `oracle` engine identifier alongside the existing PostgreSQL, MySQL, and SQLite identifiers. The connection configuration normalizes and persists it exactly like the other server engines.
- Add Oracle to the connection modal's engine cycle, display name, and default-port rules. The structured connection fields generate an Oracle Easy Connect URL for a service name; an explicit DSN takes precedence unchanged.
- Use `go-ora/v2` through `database/sql` and verify connectivity with `PingContext`. The adapter owns its `*sql.DB` and the synchronized query logger, closes both resources, and wraps propagated errors with `%w`.
- Model the initial Oracle connection target as a service name, not a SID. The local acceptance target is `localhost:1522/FREEPDB1` with the `db_tui` demo user.
- Scope discovery to objects visible in the current user's schema. Use Oracle user data-dictionary views for base tables, views, and materialized views; keep each relation type in the existing navigator section that matches its semantics.
- Treat Oracle materialized views as data-only relations in v1. They support discovery and bounded row browsing, but do not gain table-specific DDL, metadata, index, export, refresh, or management actions.
- Use bounded Oracle row limiting with one extra row to compute `RowPage.HasMore`. Validate the existing page offset and limit bounds before constructing the query, and quote relation identifiers rather than interpolating untrusted identifiers.
- Implement column and index inspection from Oracle user data-dictionary views. Map Oracle identity and index-type metadata into the existing `Column` and `IndexColumns` shapes; leave a field blank when Oracle does not expose an equivalent value safely through the selected metadata view.
- Obtain table structural DDL from Oracle metadata APIs for tables in the current schema. Return an explicit error when the object is missing, inaccessible, or Oracle cannot provide its DDL.
- Reuse the current bounded raw-query and CSV/JSON export semantics. Oracle result scanning must preserve column order and SQL `NULL`; statements that do not return rows produce a command status.
- Implement the mandatory dump operation as the explicit unsupported Oracle dump placeholder defined in ADR 0008. It must never shell out to `expdp` or claim to have produced a dump.
- Add the Oracle adapter to the executable's connection factory. No existing adapter may import application packages; the application continues to depend only on the driver-neutral database contract.

## Testing Decisions

- The primary seam is the existing `db.Database` contract. Tests verify observable connection, discovery, pagination, metadata, DDL, execution, export, dump-placeholder, and close behavior rather than adapter internals.
- Add focused unit tests for Oracle DSN parsing and validation, including standard Easy Connect URLs, missing usernames, hosts, service names, malformed ports, and explicit DSN precedence. Follow the existing PostgreSQL and MySQL connection tests.
- Extend connection-modal and persisted-connection tests to assert Oracle selection, the 1521 default port, connection DSN construction, display naming, and engine normalization. These tests must exercise the modal and connection-setting behavior rather than private UI fields where an existing higher-level assertion is available.
- Add Oracle integration tests using the local Compose `FREEPDB1` fixture and the checked-in countries/cities/currencies data. Verify successful connection, the `COUNTRIES` relation, a 196-row count, deterministic discovery, and representative `NULL` handling.
- Add integration coverage for a first page, a later page, an empty relation, invalid page bounds, and a page whose final record determines `HasMore`. Keep all result pages at or below `db.MaxPageSize`.
- Add integration coverage for columns, indexes, table DDL, ordinary views if present, and materialized-view discovery when the fixture supplies one. Assert externally meaningful metadata and DDL characteristics, not exact incidental whitespace from Oracle.
- Add raw-query tests for result-producing SQL, a non-query command status, SQL errors, and cancellation. Preserve the existing maximum raw-result-row behavior.
- Add tests that Oracle CSV/JSON table exports and SELECT query exports use the existing document shapes, and that the dump method returns the documented unsupported-operation error.
- Run `go test ./...`, `go vet ./...`, and the repository validation script with the Oracle Compose profile active. Integration tests must use the local service only and must not require remote databases, wallets, Oracle Instant Client, or credentials outside the checked-in demo fixture.

## Out of Scope

- Oracle Data Pump, `expdp`, import/restore workflows, and any database-dump implementation.
- Oracle Instant Client, CGO, `godror`, wallet-based connections, mutual TLS, and Autonomous Database-specific setup.
- SID-oriented connection UX, multi-schema browsing, cross-schema DDL, schema switching, and DBA catalog exploration.
- Stored procedure/function browsers, PL/SQL debugging, explain plans, scheduler administration, and user/role management.
- Materialized-view refresh or management actions.
- Changes to PostgreSQL, MySQL, or SQLite behavior beyond the shared engine selector and connection factory.

## Further Notes

- This spec follows ADR 0008 and the glossary term **Oracle dump placeholder**.
- Oracle object names are normally stored in uppercase unless they were created as quoted identifiers. The adapter must preserve returned names and quote them correctly when issuing relation queries.
- The current Oracle Compose service is intentionally opt-in and binds to localhost. It is suitable for the integration fixture but does not expand the runtime support promise beyond ordinary Easy Connect connections.
- The specification is local only by request; it is not published to an external issue tracker.
