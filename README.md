# db-tui

A keyboard-first terminal client for PostgreSQL, MySQL, Oracle, SQLite, SQL Server, and Redis.

## Install

db-tui currently provides a Linux x86_64 installer:

```sh
curl -fsSL https://raw.githubusercontent.com/ernesto27/db-tui/main/scripts/install.sh | bash
```

Then start the app:

```sh
db-tui
```

## Get started

1. Start `db-tui`.
2. Press `Ctrl+N` to create a connection.
3. Choose the database engine with Left/Right.
4. Enter the connection details, or provide an engine-specific DSN.
5. Press `Enter` to test, save, and open the connection.
6. Browse database objects, select a table, then press `Enter` to load its rows.

Saved connections are available with `Ctrl+L`.

SQLite connections use a local database-file path, such as
`/data/reporting.db`. PostgreSQL, MySQL, Oracle, and SQL Server can use either
the form fields or an engine-specific DSN.

Redis connections use a host and port (with optional username and password),
or a `redis://` DSN. The local Compose fixture listens on `127.0.0.1:6380`.
A Redis DSN can end in `/n` to select logical database `n`; without it,
the connection uses `db0`.
The sidebar lists only Redis logical databases that contain keys (`db0`, `db1`,
and so on). Select one to load keys into a resizable key/value/type/TTL table.
Redis shows 400 keys per page in scan order. Down or the mouse wheel at the
bottom loads the next page; PgDown and PgUp move between pages.
With the key grid focused, press `c` to copy the highlighted row's key, value,
type, and TTL separated by tabs. A “Copied to clipboard” toast appears briefly.
Use `Ctrl+R` to enter one raw Redis command at a time. Commands run against
the selected logical database, may change data without a confirmation prompt,
and refresh the key table and sidebar after success. `SELECT n` opens `dbn`;
an empty selected database remains open in the table but is absent from the sidebar.
Command replies appear as scrollable plain text. Hash fields appear as
`field: value` lines rather than Go `map[...]` output.

## Run a query without the TUI

For SQL engines, use the `query` subcommand with either `-q` (the SQL query) or `-f` (a file
containing the SQL query), plus `-c` (the connection DSN), to print rows to
standard output. This mode accepts `postgres://` or `postgresql://`, `mysql://`,
`oracle://`, and `sqlserver://` DSNs, or the path to an existing SQLite database
file. The query must contain `SELECT` and runs in a read-only transaction.

### DSN examples

| Engine | DSN |
| --- | --- |
| PostgreSQL | `postgres://user:password@db.example.com:5432/appdb?sslmode=require` |
| MySQL | `mysql://user:password@db.example.com:3306/appdb` |
| SQLite | `/path/to/app.db` |
| SQL Server | `sqlserver://user:password@db.example.com:1433?database=appdb&encrypt=true` |

```sh
db-tui query \
  -q 'SELECT id, name FROM customers ORDER BY id LIMIT 10' \
  -c 'postgres://user:password@localhost:5432/appdb?sslmode=disable'
```

### Read a query from a file

Use `-f` or `--fileQuery` with the connection DSN to run SQL stored in a file.
Do not combine it with `-q` or `--query`.

```sh
db-tui query \
  -f ./report.sql \
  -c 'postgres://user:password@localhost:5432/appdb?sslmode=disable'
```

To use a connection already saved in the TUI, pass its exact name with
`--connection`. The CLI reads the current user's
`~/.config/db-tui/config.json`:

```sh
db-tui query --connection reporting -f ./report.sql
```

If both `--connection` and a nonempty `--dsn` are supplied, `--dsn` takes
precedence.

Add `-t` or `--format` to choose the output format: `json` (the default) prints an array of
row objects, and `csv` prints a header row followed by one record per row.

```sh
db-tui query \
  -q 'SELECT id, name FROM customers ORDER BY id LIMIT 10' \
  -c 'postgres://user:password@localhost:5432/appdb?sslmode=disable' \
  -t csv
```

CSV writes an empty field for SQL `NULL` and preserves the column order of the
query.

Run `db-tui query -h` for query-command usage. Run `db-tui` without a
subcommand to start the interactive application.

## Run a Redis command without the TUI

Use `db-tui redis` with `-d` or `--dsn` and `-c` or `--command` to run
one command against a standalone Redis server. The command runs in the
logical database selected by the DSN. Its reply is printed as plain text
to standard output.

```sh
db-tui redis -d 'redis://127.0.0.1:6379/1' -c 'PING'
```

This prints `PONG`. Redis commands execute immediately, including commands
that change data. This subcommand requires a DSN and does not use saved
connection names. Run `db-tui redis -h` for usage.

## Create a database dump without the TUI

Use the `dump` subcommand with `-d` or `--dsn`, or with `--connection` and an
exact saved connection name, to write a timestamped database dump in the
current directory.

```sh
db-tui dump \
  -d 'postgres://user:password@example.com:5432/appdb?sslmode=require'
```

```sh
db-tui dump --connection reporting
```



Run `db-tui dump -h` for dump-command usage.

## Features

- Save and switch between PostgreSQL, MySQL, Oracle, SQLite, SQL Server, and Redis connections.
- Browse Redis logical databases and their keys, and run raw Redis commands in the TUI.
- Run a Redis command without opening the TUI.
- Browse tables, views, materialized views, and functions when supported by
  the connected database.
- Browse installed PostgreSQL extensions from the object chooser.
- Filter database objects and inspect table data in bounded pages.
- View table DDL, columns, and indexes.
- Edit or delete a selected row when its table has a usable primary key.
- Copy the highlighted table, SQL query-result, or Redis key row with `c` while
  its data panel has focus. Copies all loaded values separated by literal tabs, without
  headers or escaping, using the terminal's clipboard support.
  A brief “Copied to clipboard” toast appears after the copy command is sent.
- Write and execute SQL in the raw-query panel, with confirmation before
  executing statements that delete data.
- Autocomplete current-schema PostgreSQL table names in the raw-query editor.
- Save and reopen SQL scripts for each connection.
- Export a table or successful query results as CSV or JSON.
- Create timestamped database dumps from the TUI or `db-tui dump` for
  PostgreSQL, MySQL, SQLite, and SQL Server. SQL Server dumps require access
  to the database's local Docker container; Oracle dumps are not supported.
- Rename saved connections and set their environment label.

## Keyboard reference

### Global

| Key | Action |
| --- | --- |
| `Ctrl+N` | Create a connection; in the raw-query panel, start a new script |
| `Ctrl+K` | Open or close keyboard shortcuts |
| `Ctrl+L` | Open saved connections |
| `Ctrl+R` | Open the raw-query panel |
| `Ctrl+T` | Return to table data |
| `Ctrl+O` | Choose the database object category, including PostgreSQL extensions when available |
| `Ctrl+F` | Filter database objects |
| `Ctrl+G` | Open actions for the selected table or connection |
| `Ctrl+S` | Change the maximum page size |
| `Ctrl+D` | Create a database dump |
| `Ctrl+E` | Export the selected table or query results |
| `Tab` | Switch focus between visible panels |
| `q` or `Ctrl+C` | Quit (`q` types a character while the SQL editor has focus) |

### Browse tables and data

| Key | Action |
| --- | --- |
| `Up` / `Down` or `k` / `j` | Move through objects or rows |
| `Left` / `Right` | Move focus between the object list and data; scroll data columns |
| `PgUp` / `PgDown` | Move one page |
| `Home` / `End` | First / last visible database object |
| `Enter` | Load the selected table or view a selected function |
| `r` | Refresh the current table data |
| `e` | Edit the selected row |
| `d` | Delete the selected row (confirmation required) |
| `c` | Copy the selected row while the data panel has focus |
| Mouse wheel | Scroll table data or query results |

### Raw SQL

| Key | Action |
| --- | --- |
| `Ctrl+P` | Execute SQL; queries containing `DELETE` require confirmation |
| `Ctrl+N` | Clear the editor and start a new script |
| `Ctrl+H` | Open saved scripts for the current connection |
| `Ctrl+E` | Export successful query results as CSV or JSON |
| `Up` / `Down` | Move through PostgreSQL table suggestions when shown |
| `Tab` / `Enter` | Accept the highlighted table suggestion; otherwise, `Tab` switches editor/results focus |
| `Esc` | Close a table suggestion or cancel a `DELETE` confirmation |
| `Up` / `Down`, `k` / `j` | Select a result row when results have focus |
| `PgUp` / `PgDown` | Move result-row selection by one visible page |
| `c` | Copy the selected row when results have focus |
