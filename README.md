# db-tui

A keyboard-first terminal client for PostgreSQL, MySQL, Oracle, SQLite, and SQL Server.

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

## Run a query without the TUI

Pass both `-q` (the SQL query) and `-c` (the connection DSN) to print a JSON
array of rows to standard output. This mode accepts `postgres://` or
`postgresql://`, `mysql://`, `oracle://`, and `sqlserver://` DSNs, or the path
to an existing SQLite database file. The query must contain `SELECT` and runs
in a read-only transaction.

```sh
db-tui \
  -q 'SELECT id, name FROM customers ORDER BY id LIMIT 10' \
  -c 'postgres://user:password@localhost:5432/appdb?sslmode=disable'
```

Run `db-tui -h` for the concise command usage. Omit both flags to start the
interactive application.

## Features

- Save and switch between PostgreSQL, MySQL, Oracle, SQLite, and SQL Server connections.
- Browse tables, views, materialized views, and functions when supported by
  the connected database.
- Filter database objects and inspect table data in bounded pages.
- View table DDL, columns, and indexes.
- Edit or delete a selected row when its table has a usable primary key.
- Write and execute SQL in the raw-query panel.
- Save and reopen SQL scripts for each connection.
- Export a table or successful query results as CSV or JSON.
- Create timestamped SQL dumps for PostgreSQL, MySQL, SQLite, and SQL Server.
  SQL Server dumps require access to the database's local Docker container;
  Oracle dumps are not supported.
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
| `Ctrl+O` | Choose the database object category |
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
| Mouse wheel | Scroll table data or query results |

### Raw SQL

| Key | Action |
| --- | --- |
| `Ctrl+P` | Execute the SQL in the editor |
| `Ctrl+N` | Clear the editor and start a new script |
| `Ctrl+H` | Open saved scripts for the current connection |
| `Ctrl+E` | Export successful query results as CSV or JSON |
| `Tab` | Switch between the editor and results |
| `Up` / `Down`, `k` / `j`, `PgUp` / `PgDown` | Scroll results when results have focus |
