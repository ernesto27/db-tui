# db-tui

`db-tui` is a keyboard-driven terminal database client written in Go with Bubble Tea. It supports PostgreSQL, MySQL, and SQLite for browsing database objects, writing SQL, inspecting bounded query results, and creating database dumps.

## Requirements

- Go 1.26 or newer
- A PostgreSQL or MySQL server, or a local SQLite database file
- `pg_dump`, `mysqldump`, or `sqlite3` when using the corresponding dump command

## Development

Run the current test suite:

```sh
go test ./...
```

SQLite adapter tests use the checked-in Employee fixture at
`docker/sqlite/employee.db`. Tests open it read-only; no database is
downloaded or modified at test time.

## Application version

The version shown in the TUI header is defined in
[`internal/version/version.json`](internal/version/version.json). Update its
`version` value, then rebuild db-tui for the change to take effect.

Project direction and implementation work are documented in [PLAN.md](PLAN.md) and [TASKS.md](TASKS.md).

## Local PostgreSQL

Start the local Chinook demo database:

```sh
docker compose up -d
docker compose ps
```

It listens only on `127.0.0.1:5433`. Connect with either command:

```sh
docker compose exec postgres psql -U db_tui -d chinook
psql 'postgres://db_tui@localhost:5433/chinook?sslmode=disable'
```

The Chinook dump is in `docker/postgres/init/001_chinook.sql`. PostgreSQL runs files in that directory only when its `postgres-data` volume is empty. Run `docker compose down` to stop the service. To remove the database and import the dump again, run `docker compose down -v` before starting it again. This development service uses trust authentication and binds to localhost only.

## Database connections

On startup, db-tui creates `$HOME/.config/db-tui/config.json` when necessary. Press `Ctrl+N` to create a connection or `Ctrl+L` to open saved connections. Select PostgreSQL, MySQL, or SQLite with Left/Right while the Engine field is focused. Server engines accept the individual connection fields or an engine-specific DSN; SQLite accepts a local database-file path only, stored in the existing `dsn` setting.

PostgreSQL accepts URL DSNs:

```text
postgres://user:password@host:5432/database
```

MySQL accepts both the native `go-sql-driver/mysql` form and URL form:

```text
user:password@tcp(host:3306)/database?parseTime=true
mysql://user:password@host:3306/database?parseTime=true
```

SQLite uses the pure-Go `modernc.org/sqlite` driver, so connecting does not
require CGO or a system SQLite library. Enter a local path such as:

```text
docker/sqlite/employee.db
/data/reporting.db
```

The `sqlite3` executable is required only when using the SQLite database dump
command; browsing, queries, DDL, and CSV/JSON exports do not require it.

When a DSN is provided it takes precedence over the individual fields. The app tests the connection before saving it and immediately switches to the successful connection. Connection failures stay in the modal and preserve the entered values.

Press `Ctrl+R` to open the raw SQL panel. Enter a statement for the connected database and press `Ctrl+P` to run it; plain Enter adds a new line. After a query runs, its results receive focus and can be scrolled with Up/Down, j/k, PgUp/PgDown, or the mouse wheel. Press Tab to switch between results and the editor. Query results display at most 100 rows. Press `Ctrl+T` to return to the selected table's data view. Press `Ctrl+G` to open a read-only DDL modal for the selected table from either panel. In the table-data view, Tab cycles focus between the table list, the table filter, and the data list; Ctrl+F opens the filter directly.

Passwords are stored as plaintext in the connection fields or DSN. The configuration file is private to the current user (`0600`), but it is not encrypted. Do not commit passwords or credential-bearing database URLs.

## License

Licensed under the [MIT License](LICENSE).
