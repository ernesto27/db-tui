# db-tui

`db-tui` is a keyboard-driven terminal database client written in Go with Bubble Tea. The first release targets PostgreSQL and focuses on the core workflow: browse database objects, write SQL, and inspect bounded query results.

## Requirements

- Go 1.26 or newer
- PostgreSQL support will be added during the first implementation milestone

## Development

Run the current test suite:

```sh
go test ./...
```

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

## Local configuration and secrets

On startup, db-tui creates `$HOME/.config/db-tui` and `$HOME/.config/db-tui/config.json` when necessary. The generated file contains an empty PostgreSQL DSN:

```json
{
  "postgresql": {
    "dsn": ""
  }
}
```

Set `postgresql.dsn` to the desired connection string. If the file is invalid, the DSN is empty, or the database cannot be reached, db-tui displays the error in the TUI.

Press `Ctrl+L` in the TUI to open the PostgreSQL connection modal. You can paste a complete DSN, or enter the host, database name, port, username, and optional password. When a DSN is provided it takes precedence over the individual fields and is saved as `postgresql.dsn`. When you connect with the individual fields, those fields are saved in `postgresql` instead—no generated DSN is written to the configuration file. The app tests the connection before saving it and immediately switches to the successful connection. Connection failures stay in the modal and preserve the entered values.

Press `Ctrl+R` to open the raw SQL panel. Enter any PostgreSQL statement and press `Ctrl+P` to run it; plain Enter adds a new line. After a query runs, its results receive focus and can be scrolled with Up/Down, j/k, PgUp/PgDown, or the mouse wheel. Press Tab to switch between results and the editor. Query results display at most 100 rows. Press `Ctrl+T` to return to the selected table's data view.

For this first version, a password is stored as plaintext inside the DSN. The configuration file is private to the current user (`0600`), but it is not encrypted. Do not commit passwords or credential-bearing database URLs.

## License

Licensed under the [MIT License](LICENSE).
