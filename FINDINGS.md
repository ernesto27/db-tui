# SQLite title-row editing finding

Editing a row in SQLite's `title` table can fail with:

```text
no row matched the WHERE clause; the row may have been modified or deleted
```

## Cause

`title` has the composite primary key `(emp_no, title, from_date)`. In the
fixture, `from_date` is declared as `DATE` and stored as text such as
`1986-06-26`.

The SQLite driver returns this value to the TUI as a `time.Time`. The edit
flow keeps that original value for the primary-key `WHERE` clause. When bound
back to SQLite, it is represented as a timestamp such as
`1986-06-26 00:00:00 +0000 UTC`, which does not equal the date-only text in
the database. The update therefore affects zero rows.

## Suggested fix

Normalize SQLite values from columns declared as `DATE` to `YYYY-MM-DD` when
rows are read, before they reach the edit modal. Add a regression test using
a composite key that includes a date-only column and verify an update using
the loaded key values succeeds.

The attempted normalization was intentionally rolled back so this finding can
be addressed separately.

# MySQL matching rows versus changed rows

`mysqlDatabase.UpdateRow` currently treats `RowsAffected() == 0` as proof that
the primary-key `WHERE` clause matched no row. That is incorrect with the
default `go-sql-driver/mysql` configuration: it reports changed rows, rather
than matching rows.

For example, updating a numeric value from `1780000` to the equivalent input
`01780000` can match the row while leaving its stored value unchanged. MySQL
then reports zero changed rows, and the TUI incorrectly reports that the row
was deleted or modified.

## Suggested fix

Set `driverConfig.ClientFoundRows = true` in `mysql.Connect` before creating
the connector. The driver will then report matching rows, making the existing
zero-row stale/deleted check correct.

Add a regression test that updates a numeric city value with an equivalent
string representation and expects `UpdateRow` to succeed.

# PostgreSQL UpdateRow primary-key validation

`postgresql.UpdateRow` currently builds its `WHERE` clause from whatever
columns the caller supplies. Unlike the MySQL, SQLite, and Oracle adapters, it
does not verify that those columns are exactly the table's complete primary
key. A caller could therefore supply an incomplete composite key or a
non-unique column and unintentionally update multiple rows.

## Suggested fix

Before executing the update, discover the table's primary-key columns and
require `whereColumns` to contain that complete key with no unrelated columns.
Reject tables without a primary key, empty `setColumns` or `whereColumns`, and
invalid primary-key values with explicit errors instead of relying on
PostgreSQL to reject malformed SQL. Keep the existing zero-row check and add
tests for missing keys, incomplete composite keys, non-key columns, empty
inputs, and a successful complete-key update.
