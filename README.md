# db-tui

`db-tui` is a keyboard-driven terminal database client written in Go with Bubble Tea. The first release targets PostgreSQL and focuses on the core workflow: browse database objects, write SQL, and inspect bounded query results.

The project is currently in the planning and bootstrap stage.

## Requirements

- Go 1.26 or newer
- PostgreSQL support will be added during the first implementation milestone

## Development

Run the current test suite:

```sh
go test ./...
```

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

Do not commit passwords or credential-bearing database URLs. Local environment, connection, secret, and configuration files are ignored by Git; future connection profiles will resolve passwords from environment variables or an interactive prompt.

## License

Licensed under the [MIT License](LICENSE).
