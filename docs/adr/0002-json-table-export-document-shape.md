# ADR 0002: Export tables as named JSON documents

## Status

Accepted on 2026-07-28.

## Context

db-tui already exports a selected table's complete data set as CSV. JSON export must preserve the column names that give each value meaning and produce a document that is immediately useful without consulting a separate header row.

## Decision

JSON table export will read every row from the selected table and write one JSON document. Its sole top-level property is the selected table name. The property's value is an array of row objects, with database column names as object keys. SQL `NULL` values are represented by JSON `null`.

Values that JSON cannot represent directly use stable encodings: timestamps are RFC 3339 strings, binary values are base64 strings, and precision-sensitive decimal values are strings. Boolean and ordinary numeric values remain JSON booleans and numbers.

Documents are pretty-printed with two-space indentation.

On the table-data panel, `Ctrl+E` opens a picker for CSV or JSON, followed by the existing confirmation and progress flow for the selected format. Raw-query export remains CSV-only in this release.

The picker defaults to CSV. `Up`/`Down` and `j`/`k` change the selection, `Enter` advances to confirmation, and `Esc` cancels. JSON files use the existing export destination and timestamped safe-table-name convention, with a `.json` extension.

For example, exporting `Artist` begins as:

```json
{
  "Artist": [
    {"ArtistId": 1, "Name": "AC/DC"},
    {"ArtistId": 2, "Name": "Accept"}
  ]
}
```

The export is not limited by the data-grid page size.

## Consequences

### Positive

- Column names travel with each JSON value.
- The document identifies the exported table without separate metadata.
- Exported data includes all table rows, matching CSV export scope.
- CSV remains available without a separate key binding.
- Consumers can decode non-native database values consistently across database engines.
- Pretty printing makes exports practical to inspect and review manually.
- The default selection preserves the current CSV export path for habitual users.

### Negative

- Repeated keys make the document larger than a column-and-row matrix.
- A full table is materialized or streamed outside the bounded grid page, so very large tables can produce large files.

## Alternatives considered

### Bare array of row objects

Rejected because it lacks the table identity requested for the document.

### Columns plus row arrays

Rejected because consumers must combine positions with a separate column list to interpret values.

### JSON export for raw queries

Deferred. The requested scope is selected tables, and retaining CSV-only raw-query export keeps this initial UI and database-interface change focused.
