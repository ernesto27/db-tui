# ADR 0003: Report elapsed time for raw queries

## Status

Accepted on 2026-07-29.

## Context

The raw-query panel needs to show how long a submitted statement took, including when it fails. The requested scope covers every supported database engine and requires executing the submitted SQL once, unchanged.

PostgreSQL, MySQL, and SQLite do not expose a portable, exact per-statement server-execution duration through their ordinary execution APIs. Obtaining an engine-reported duration would require database-specific instrumentation or a changed/repeated statement, such as `EXPLAIN ANALYZE`; neither is in scope.

## Decision

The application will measure elapsed time immediately around `db.Database.Execute` in the asynchronous raw-query command. The measured duration begins immediately before calling `Execute` and ends when it returns, whether it returns a result or an error.

Raw queries have a 20-minute context deadline. The existing five-second timeout remains in place for loading tables, rows, and DDL. While a raw query is executing, further submissions are ignored so the application cannot stack queries against the connection pool. User-initiated cancellation is not included in this change.

This is client-observed elapsed time, not a server-only metric. It includes the driver call, network transfer where applicable, and the adapter's bounded result reading/decoding. The original SQL is executed exactly once and unchanged. The database interface and individual database adapters will not gain timing-specific behavior.

The raw-query panel will retain the latest completed request's elapsed time and render it next to the existing result metadata:

- row results: in the `Results` header beside row count and command status;
- command-only and zero-row results: beside the command status;
- failures, including cancellation or timeout: beside the failure label before the error text.

The UI will use Go's human-readable duration form and label it `Execution time`, while the implementation and documentation call the metric client-observed elapsed time to avoid implying server-only measurement. A new submission or connection reset clears the previous duration. Stale completion messages continue to be ignored, including their duration.

## Consequences

### Positive

- Every supported engine reports a duration without configuration or elevated database privileges.
- Failed, cancelled, and timed-out executions retain useful elapsed-time context.
- The submitted SQL remains safe from instrumentation-induced semantic changes and runs only once.
- Timing stays in the application layer, leaving the driver-neutral database interface unchanged.
- Long-running administrative or analytical statements have a practical execution window without weakening the responsiveness budget for table browsing.

### Negative

- The displayed value is not an exact database-server execution duration.
- Network latency and client-side row materialization affect the value.
- The value covers only the raw-query execution command, not rendering time.
- A running raw query cannot currently be cancelled from the UI and may keep the panel busy for up to 20 minutes.

## Alternatives considered

### Server-reported per-statement timing

Rejected for now because no unchanged-query, portable mechanism supplies it across PostgreSQL, MySQL, and SQLite.

### `EXPLAIN ANALYZE`

Rejected because it is engine-specific, changes the execution flow and output, and does not satisfy the requirement to run the original query once unchanged.

### Timing inside every database adapter

Rejected because it duplicates behavior across adapters and expands the database boundary without improving the cross-engine accuracy of the metric.
