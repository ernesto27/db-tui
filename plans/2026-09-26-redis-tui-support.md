# Plan: Redis support in the TUI

- **Date:** 2026-09-26
- **Domain(s):** Go backend, terminal UI, local database integration
- **Author:** plan-from-spec (reviewed with the user)
- **Status:** Implementation in progress; Redis client tests added after user review

## 1. Summary

Add standalone Redis connections to db-tui's interactive application. A Redis connection has a concrete client, separate from the SQL-oriented `db.Database` interface. The navigator lists only logical databases with keys, the existing resizable data grid displays 400 keys per page in the selected database, and the existing raw-query panel executes one Redis command at a time. The workflow follows the TablePlus screenshot discussed during the interview. Redis connections use the existing JSON connection store.

## 2. Scope

### In scope

- New-connection form: Redis engine; host and port with optional ACL username/password; optional `redis://` DSN; saved under the existing connection JSON shape. Host defaults to `127.0.0.1`, port to `6379`, and the initial logical database is `db0`.
- Standalone Redis, including password-only and ACL authentication. The Compose fixture is Redis 7.4.
- Sidebar entries only for logical databases with keys, discovered through `INFO keyspace` after each table load. `CONFIG GET databases` is required for database index validation; connection setup fails with a clear error if the server denies it or returns an invalid count.
- A read-only table with columns `key`, `value`, `type`, and `ttl`. One row represents one key. Strings, hashes, lists, sets, sorted sets, and streams from Redis 7.4 render in the `value` cell. The existing grid's column resizing and terminal sizing apply.
- An internal `SCAN` cursor with duplicate-key removal and 400 keys per page. Down or the mouse wheel at the bottom and PgDown load the next page; PgUp returns to a cached previous page. No key sorting or separate detail view.
- One raw command per execution, parsed with quoted arguments. It targets the selected logical database. `SELECT n` opens that database and reloads the table, even if it is absent from the sidebar. A successful command refreshes the table and sidebar. Replies appear as scrollable plain text in the existing query panel.
- No Redis command confirmation, Redis read-only toggle, or Redis script persistence in this version.

### Out of scope / non-goals

- Redis Cluster, TLS, Redis Sentinel, Redis modules, and type additions after Redis 7.4.
- Editing or deleting keys directly from the data grid; a separate key detail screen.
- `db-tui query` and `db-tui dump` support for Redis. Their SQL contracts stay as they are.
- Incremental display within a page, key sorting, key filtering, and live subscription output. These can be designed later.

## 3. Resolved decisions

| # | Question | Decision |
|---|---|---|
| 1 | What should match TablePlus? | Logical databases in the sidebar, one key/value/type/TTL row per key, and a raw command area. |
| 2 | Can the browser mutate keys? | No. Raw commands may mutate keys. |
| 3 | Is a separate value detail view needed? | No; use the existing resizable data grid. |
| 4 | Which logical databases appear? | Only databases with keys; empty databases remain selectable by raw `SELECT`. |
| 5 | How much data is displayed? | Up to 400 keys and their complete values per page. Fetch the next page on demand in scan order; cache earlier pages. |
| 6 | Which Redis deployment and types? | Standalone Redis; support built-in types in the Redis 7.4 Compose fixture. No TLS or Cluster. |
| 7 | How are connections entered and stored? | Add Redis to the new-connection form, including an optional DSN, and save with existing JSON config. |
| 8 | Must Redis implement `db.Database` or a new interface? | Neither. Use a concrete Redis client in a separate `internal/redis` package. |
| 9 | What if `CONFIG GET databases` is denied? | Fail connection setup. |
| 10 | Where do commands run? | Existing query panel; one command at a time against the sidebar's selected database. |
| 11 | What does raw `SELECT n` do? | Open that database and reload its keys; it appears in the sidebar only when it contains keys. |
| 12 | When does the table refresh? | After a successful raw command. |
| 13 | Are confirmations or read-only mode needed? | No Redis confirmation prompt and no Redis read-only toggle at this stage. |
| 14 | Are Redis commands saved as scripts? | No. |
| 15 | Does the CLI need Redis? | No; TUI only. |
| 16 | What sample data exists for switching? | The approved Compose change seeds a string and a hash in `db1`; `db0` already has fixtures. |

## 4. Design

### Boundaries and ownership

`internal/redis` owns the concrete `Client`, Redis protocol dependency, connection parsing, authentication, logical-database clients, scanning, value formatting, and raw replies. It never imports `internal/app`. `cmd/db-tui` wires `redis.Connect` into the root model. `internal/app` may hold `*redis.Client` alongside its existing `db.Database`, but exactly one is active. The root model owns the selected Redis logical database, request identity, loading state, and the conversion from Redis rows/results to existing grid/query-panel render models. This is a deliberate change to `ARCHITECTURE.md`; it avoids adding Redis-shaped methods or SQL stubs to `db.Database`.

The Redis client is concrete because there is one implementation. Connection setup remains injectable with a function value so application transition tests do not need to create a network connection. No Redis interface is introduced.

### Connection behavior

The Redis form shows Engine, Host, Port, Username, Password, and DSN (optional). A nonempty DSN takes precedence over the other fields, as it does for current SQL engines. A generated URL uses `redis://`, percent-encodes credentials, and has no database path because the sidebar controls database selection. Existing config file permissions (`0700` directory, `0600` file) apply; credentials remain plaintext as with existing engines. Passwords and full DSNs are never logged or rendered in errors.

The adapter opens and pings `db0`, requests `CONFIG GET databases`, validates a positive count, and creates clients for other database indexes on demand. Each per-database Go client has its own `DB` option; raw `SELECT` is handled as a UI navigation command rather than mutating a pooled connection. The active connection changes only after successful setup and JSON save. Failed setup preserves the previous connection.

### Key table

On entering or refreshing `dbN`, start a new `SCAN` cursor and load up to 400 existing keys. Keep its position, pending names, and duplicate-key set across pages; read one additional existing key to determine whether another page exists. For each displayed key, read `TYPE`, `PTTL`, and its full value using the command appropriate to that type (`GET`, `HGETALL`, `LRANGE`, `SMEMBERS`, `ZRANGE WITHSCORES`, or `XRANGE`). Use JSON-style compact text for collections and plain text for readable strings. Escape invalid UTF-8 and terminal control characters; never pass raw control bytes into the grid. Show an em dash for keys without expiry, otherwise a human-readable TTL. If a key expires between scan and read, omit that row. Propagate permission and network errors with context. Values are fetched in the scan order returned by Redis; sets and hash fields may be formatted deterministically without sorting keys globally.

The scan and value reads happen in a `tea.Cmd` with a bounded context. `Update` only applies a result when the session, database, page index, and Redis load request match. Switching databases or connections invalidates older results. Up to 400 rows appear when a page finishes loading. Previous pages are cached; only the most recent cursor is retained for forward scans. Switching databases, refreshing, or running a command resets that cache. Individual collection values may still be large. The UI stays responsive while loading, and a timeout or error is shown if the load fails.

### Commands and replies

Parse one line of Redis command syntax with spaces, single/double quotes, and backslash escaping. Reject an empty command, unmatched quote, or a second command separated by a newline. Send the parsed arguments with go-redis `Do` on the selected database client. Render scalar replies directly, arrays one item per line, and maps as sorted `field: value` lines; represent nil explicitly. Preserve newlines while removing terminal controls. Wrap and scroll long replies in the query panel. Render errors through the existing sanitized error view. Treat a successful `SELECT n` specially: validate `n`, open that database, show `OK`, and reload `dbN`. Run all other commands immediately, without a confirmation or read-only gate. Refresh the key table and populated-database sidebar after success; if refresh fails, retain the command result and show the table error.

The existing query editor labels become engine-aware: Redis uses “Redis command” and “Write a Redis command…” rather than SQL language. SQL highlighting/autocomplete and SQL script save/export controls remain SQL-only. The Redis editor is unsaved when the connection closes.

## 5. Interfaces & contracts

### Concrete Redis package

```go
// internal/redis/client.go
package redis

type Settings struct {
    Host     string
    Port     int
    Username string
    Password string
    DSN      string
}

type Key struct {
    Name  string
    Value string
    Type  string
    TTL   string
}

type Result struct {
    Raw            string
    CommandTag     string
    SelectDatabase *int
}

type Client struct {
    // private go-redis options, per-database clients, and mutex
}

func Connect(ctx context.Context, settings Settings) (*Client, error)
func (c *Client) Host() string
func (c *Client) DatabaseCount() int
func (c *Client) LoadKeyPage(ctx context.Context, database, limit int, cursor ScanCursor) (KeyPage, error)
func (c *Client) Execute(ctx context.Context, database int, command string) (Result, error)
func (c *Client) Close() error
```

`Connect` rejects an invalid DSN, absent host, out-of-range port, failed ping/auth, or inaccessible/invalid database count. `LoadKeyPage` and `Execute` reject an index outside `[0, DatabaseCount())` before sending a command. Errors wrap the underlying cause with `%w` and contain no credentials.

### Application connection state

```go
// internal/app/connection.go, internal/app/model.go
type RedisConnectFunc func(context.Context, redis.Settings) (*redis.Client, error)

type connectionFinishedMsg struct {
    database db.Database
    redis    *redis.Client
    settings ConnectionSettings
    attempt  uint64
    err      error
}

type Model struct {
    // existing fields
    redis         *redis.Client
    connectRedis  RedisConnectFunc
    redisDatabase int
    redisRequest  uint64
}
```

`connectionFinishedMsg` contains either `database` or `redis`, never both. Stale successful results close whichever session they carry. A Redis result always carries the root session and Redis request number; query results retain the existing query request identity.

### JSON shape

```json
{
  "name": "Redis connection (127.0.0.1)",
  "engine": "redis",
  "settings": {
    "hostname": "127.0.0.1",
    "database": "",
    "username": "",
    "password": "",
    "port": "6380",
    "dsn": ""
  },
  "status": false
}
```

The existing `config.Connection` and `config.Settings` fields are reused. A DSN-based connection stores `settings.dsn` and does not require the other fields. The logical database selection is session UI state and is not persisted.

## 6. Behavior & states

1. Selecting Redis in the connection form changes visible fields, default port, validation, and labels. Enter tests the connection; a successful new or edited connection is saved and adopted. A failed test leaves the modal open and preserves the old active connection.
2. After adoption, set `redisDatabase = 0`, clear SQL table/function state, and start a `db0` load. Discover populated database entries using `INFO keyspace`. Empty databases show the grid headers and an empty-state message when opened through `SELECT`.
3. Selecting a sidebar database changes the active index, invalidates the prior load, and starts a new first page of up to 400 keys. Down or the mouse wheel at the bottom and PgDown fetch the next page; PgUp returns to a cached page. While loading, show progress text/spinner; on failure, show a sanitized error. Terminal resize and column resizing reuse the current grid math.
4. Opening the query panel shows Redis wording. Ctrl+P parses and executes one command. Syntax errors stay in the panel. A command completion may change the selected database (`SELECT`) and then refreshes the key table. The command result remains visible during refresh.
5. Reconnecting, deleting, or replacing a Redis connection closes its clients and rejects stale load/query results. Switching back to a SQL engine restores existing SQL views and controls. Quit closes the active session.
6. Redis grid edit/delete keys and actions, dump/export, SQL scripts, SQL highlighter/completion, and read-only toggle are hidden or disabled for Redis. Raw commands can still mutate data. The confirmed no-confirmation behavior is recorded as an explicit Redis exception to the existing SQL safety convention in `ARCHITECTURE.md`.

## 7. Implementation tasks

Every task is an implementation step for a later approved change. No task is implemented by this plan.

### Task 1 — Add Redis connection fields and concrete adapter wiring

- **Why:** Make Redis selectable, validate its connection details, and persist them without changing SQL connection behavior.
- **Files & changes:**
  - `go.mod`, `go.sum` (edit): add `github.com/redis/go-redis/v9`; run `go mod tidy` and `go mod verify` after implementation.
  - `internal/config/config.go` (edit, `Connection.Target`): admit `db.EngineRedis` only for saved Redis connections; a nonempty `settings.dsn` must have a `redis://` scheme; otherwise construct `redis://` from host, valid port, and optional URL-encoded credentials, without requiring database name or username. Keep current SQL branches as they are.
    ```go
    if engine == db.EngineRedis {
        if dsn := strings.TrimSpace(settings.DSN); dsn != "" {
            parsed, err := url.Parse(dsn)
            if err != nil || parsed.Scheme != "redis" || parsed.Hostname() == "" {
                return "", "", errors.New("valid redis:// DSN is required")
            }
            return engine, dsn, nil
        }
        // Validate hostname and port; username and password are optional.
        return engine, redisURL(settings).String(), nil
    }
    ```
    `redisURL` is a new unexported helper beside `Connection.Target`; it uses `net.JoinHostPort` and `url.User`/`url.UserPassword` so IPv6 and special credential bytes are encoded correctly.
  - `internal/db/db.go` (edit, engine constants only): add `EngineRedis = "redis"`. Do not change `db.Database`.
  - `internal/app/connection_modal.go` (edit, `connectionEngines`, `defaultPortForEngine`, `connectionInputsForEngine`, `setDSNPlaceholder`, `connectionSettings`, `view`): add Redis with port `6379`; show Host, Port, Username, Password, and optional DSN, but no Database name; use the existing password masking. The `connectionSettings()` branch calls the Redis validation path.
    ```go
    if m.engine() == db.EngineRedis {
        return []connectionInput{engineInput, hostInput, portInput,
            usernameInput, passwordInput, dsnInput}
    }
    ```
  - `internal/app/connection.go` (edit): add `RedisConnectFunc`, a Redis branch in `connectConnection`, and a Redis field on `connectionFinishedMsg`. Add a shared `closeFinishedConnection` helper to close stale SQL or Redis results. Keep current attempt checks.
  - `internal/app/connections_modal.go` (edit): reuse existing JSON field conversion; name Redis connections from engine and host/DSN host when database name is empty. Restore Redis fields on edit/open.
  - `cmd/db-tui/interactive.go` (edit): pass `redis.Connect` through the updated `app.New` constructor. Do not modify the SQL-only CLI commands.
  - `internal/app/view.go` (edit, `engineDisplayName`): render `Redis`.
- **Depends on:** —

### Task 2 — Implement the concrete Redis client and paged key loading

- **Why:** Keep Redis operations out of SQL adapters and load 400 keys at a time into the table.
- **Files & changes:**
  - `internal/redis/client.go` (new): define the `Settings`, `Key`, `Result`, and concrete `Client` declarations from section 5. Parse a DSN with `go-redis` URL options or construct `redis.Options` from fields. `Connect` pings `db0`, calls `CONFIG GET databases`, parses a positive count, and closes opened clients on failure. `clientFor(database)` validates range and lazily opens a client with the same credentials and `DB: database`. `Close` closes all opened clients once.
    ```go
    func (c *Client) clientFor(database int) (*goredis.Client, error) {
        if database < 0 || database >= c.databaseCount {
            return nil, fmt.Errorf("Redis database %d is out of range", database)
        }
        c.mu.Lock()
        defer c.mu.Unlock()
        if client := c.clients[database]; client != nil {
            return client, nil
        }
        options := c.options
        options.DB = database
        client := goredis.NewClient(&options)
        c.clients[database] = client
        return client, nil
    }
    ```
  - `internal/redis/client.go` (same file): implement `LoadKeyPage`. Start cursor 0; call `SCAN` until 400 existing keys and one lookahead key have loaded or scanning finishes; retain scan position, pending names, and visited names for the next page. Check `ctx.Err()` on each iteration and value read; use `TYPE` to dispatch to complete value reads; call `PTTL`; omit keys that have expired; return no partial success on other errors. Format value text in `internal/redis/client.go`.
    ```go
    for {
        names, next, err := client.Scan(ctx, cursor, "*", 0).Result()
        if err != nil { return nil, fmt.Errorf("scan Redis keys: %w", err) }
        for _, name := range names {
            if _, seen := visited[name]; seen { continue }
            visited[name] = struct{}{}
            key, exists, err := loadKey(ctx, client, name)
            if err != nil { return nil, err }
            if exists { keys = append(keys, key) }
        }
        cursor = next
        if cursor == 0 { return keys, nil }
    }
    ```
  - `internal/redis/client.go` (same file): write `formatValue(kind, response)` and `formatTTL(duration)` helpers. Strings are readable text or escaped binary; hash/list/set/zset/stream content is compact JSON-style text; normalize control bytes before display; unknown Redis 7.4 type returns a clear error rather than silently hiding content.
- **Depends on:** Task 1.

### Task 3 — Add Redis navigator and grid state

- **Why:** Reuse the existing table display while keeping Redis selection separate from SQL relations.
- **Files & changes:**
  - `internal/app/model.go` (edit): add `redis *redis.Client`, `connectRedis RedisConnectFunc`, `redisDatabase int`, and `redisRequest uint64`; extend `New` to receive the connector; `Close` closes whichever active session exists. No `db.Database` implementation is requested from Redis.
  - `internal/app/redis_state.go` (new): define Redis database cursor/selection, cached pages, scan cursor, and load status. Convert each Redis key page to `db.RowPage` for the shared grid. Keep database-index validation here and reset state on connection changes.
    ```go
    func redisRowPage(page redis.KeyPage) db.RowPage {
        rows := make([][]any, len(page.Keys))
        for i, key := range page.Keys {
            rows[i] = []any{key.Name, key.Value, key.Type, key.TTL}
        }
        return db.RowPage{Columns: []string{"key", "value", "type", "ttl"}, Rows: rows, HasMore: page.HasMore}
    }
    ```
  - `internal/app/redis.go` (edit): add a bounded `loadRedisPage` command with a typed message carrying `session`, `request`, `database`, page index, page, and error. Database changes increment the request counter; stale results are ignored. Cache prior pages for PgUp while keeping only the latest scan cursor for fetching more.
  - `internal/app/update.go` (edit): handle Redis connection completion/adoption and lifecycle messages; route sidebar keyboard/mouse activation to Redis database selection; route refresh to Redis reload; disable SQL table actions while Redis is active. Keep SQL paths as they are.
  - `internal/app/redis.go` (edit): when Redis is active, render only populated `dbN` entries from `INFO keyspace` and move/activate the logical-database cursor without presenting SQL sections. Keep the existing navigator geometry and focus behavior.
  - `internal/app/redis.go` (edit): use Redis-specific loading/empty/error labels and the `dataModel` grid for Redis; route page navigation while suppressing SQL edit/delete affordances. `internal/app/data_grid.go` resizing remains shared and unchanged.
  - `internal/app/view.go` (edit): choose Redis navigation/data status when `m.redis != nil`; show Redis connection name/host in the header.
- **Depends on:** Tasks 1–2.

### Task 4 — Execute raw Redis commands in the existing query panel

- **Why:** Provide the second requested workflow without SQL parsing or another screen.
- **Files & changes:**
  - `internal/redis/client.go` (same file): implement `parseCommand(string) ([]string, error)` for one command with quotes/escapes and no embedded newline. In `Client.Execute`, handle `SELECT n` by validating the index and returning `SelectDatabase`; otherwise call `client.Do(ctx, arguments...)`, then turn nil/scalar/array/map replies into terminal-safe plain text. Preserve an underlying Redis error with `%w` and never include credentials in it.
    ```go
    if strings.EqualFold(arguments[0], "SELECT") {
        if len(arguments) != 2 { return Result{}, errors.New("SELECT requires one database index") }
        index, err := strconv.Atoi(arguments[1])
        if err != nil || index < 0 || index >= c.databaseCount {
            return Result{}, errors.New("Redis database index is out of range")
        }
        return Result{Raw: "OK", CommandTag: "SELECT", SelectDatabase: &index}, nil
    }
    reply, err := client.Do(ctx, stringArguments(arguments)...).Result()
    if err != nil { return Result{}, fmt.Errorf("execute Redis command: %w", err) }
    return Result{Raw: rawReply(reply), CommandTag: arguments[0]}, nil
    ```
  - `internal/app/commands.go` (edit): add a bounded `executeRedisCommand` `tea.Cmd` whose typed result includes session and query request identifiers.
  - `internal/app/update.go` (edit, `startQuery`, query completion): if Redis is active, skip SQL DELETE detection and SQL read-only mode, call `executeRedisCommand`, show the result, update sidebar for `SELECT`, and restart Redis paging after any success. Do not call `saveSQLScript` for Redis.
  - `internal/app/query_panel.go` (edit): accept engine-aware labels for editor placeholder, headings, empty instructions, and error/status text. Render Redis replies as wrapped, scrollable plain text without a result grid. Keep SQL completion and highlighting disabled when Redis is active.
  - `internal/app/view.go`, `internal/app/update.go` (edit): hide/disable SQL-only query actions (saved scripts and SQL query export) and the read-only toggle while Redis is active.
- **Depends on:** Tasks 2–3.

### Task 5 — Focused automated tests and documentation

- **Why:** Prove the new behavior and keep architectural rules accurate.
- **Files & changes:**
  - `internal/config/config_test.go` (edit): table tests for Redis field URL, DSN precedence, password-only and ACL credentials, IPv6, invalid host/port/DSN, and JSON save/load. Assert that error text does not expose credentials.
  - `internal/redis/client_test.go` (new): unit cases for connection options, database-count and keyspace parsing, database index bounds, key deduplication and paging, every seeded Redis 7.4 value type, TTL display, binary/control-byte escaping, quoting/escapes/unmatched quotes/multiline rejection, SELECT handling, nil/scalar/array/map replies, cancellation, and command errors. Integration cases require the local Compose Redis service at `127.0.0.1:6380` and fail if it is unavailable. Use isolated test key prefixes and remove test keys in cleanup.
  - `internal/app/connections_modal_test.go`, `internal/app/update_lifecycle_test.go`, `internal/app/query_panel_test.go`, `internal/app/view_test.go` (edit): verify Redis form fields, JSON round trip, successful/failed/stale connection transitions, database switch, stale key-load rejection, grid columns/resizing, Redis wording, one-command execution, and SQL behavior parity. Construct typed result messages for state tests; no Redis interface or remote server is needed.
  - `cmd/db-tui/main_test.go` or existing command tests (edit): verify Redis remains TUI-only and that existing SQL CLI DSN detection is unchanged.
  - `ARCHITECTURE.md` (edit): add `internal/redis`, document the concrete-client dependency from app, exclusive SQL/Redis active state, asynchronous Redis request identities, full-load behavior, and the user-approved no-confirmation Redis raw-command exception.
  - `README.md` (edit): list Redis in TUI connection setup, document `redis://` DSN/port `6380` fixture, logical-database navigation, 400-key paging, raw-command behavior, and TUI-only scope. Label SQL-only CLI instructions clearly.
  - `compose.yaml` (existing approved edit): retain the already-added `db1` string/hash seeds; do not create another Redis service.
- **Depends on:** Tasks 1–4.

## 8. Testing

- **Unit tests:** connection validation/serialization, command parsing, reply and value formatting, TTL/expiry behavior, duplicate scan keys, request staleness, Redis UI transitions, labels, resizing, and absence of Redis SQL actions. Use Go `testing` plus `testify/assert`/`require`, with table-driven subtests for varied inputs.
- **Integration tests:** require the local Compose Redis fixture, discover the database count, load `db0`/`db1`, verify each Redis 7.4 core type and TTL, execute read/write/SELECT commands, and close. Missing Compose Redis is a test failure. No remote database or credential dependency. SQL integration tests continue under their existing setup.
- **Final validation:** after implementing all tasks and tests, run `go mod tidy`, `go mod verify`, `go build ./...`, and `scripts/validate.sh` once at the end. The user performs manual TUI testing.

## 9. Acceptance criteria

1. Creating, saving, reopening, editing, and switching to a Redis connection works with host/port/auth fields or a `redis://` DSN. JSON file permissions and existing SQL connections remain correct.
2. A successful standalone connection lists only populated `dbN` entries. Denied `CONFIG GET databases` produces a clear connection error without replacing an active session.
3. `db0` displays the existing fixtures; `db1` displays the approved two fixtures. An empty database opened through `SELECT` shows an empty state without a sidebar entry. Key/value/type/TTL columns resize like SQL table columns.
4. Every Redis 7.4 core type in the fixture displays its complete value safely. The UI shows at most 400 keys per page in scan order; Down and PgDown load more, and PgUp returns to cached pages.
5. A raw command runs once on the selected database, displays its reply, and refreshes the table. `SELECT 1` moves the sidebar to `db1`; no confirmation appears. SQL-only controls are unavailable in Redis mode.
6. Switching connections or databases rejects late results and closes stale clients. Unit, local integration, build, and repository validation checks pass.

## 10. Risks & open items

- **Large values remain expensive:** The 400-key page size bounds each request, but each collection value is read in full. Cached pages consume memory as the user browses. The adapter uses cancellable operations and a visible error on timeout; incremental display within a page is deferred.
- **`SCAN` is not a snapshot:** Redis may change during traversal. Duplicates are removed, but concurrent additions/deletions can affect what appears. Refresh obtains a new view.
- **Command breadth:** Ordinary request/reply commands are supported through `Do`; long-lived subscription/monitor streams cannot be presented as a finite result grid. They can time out or return an error; a streaming console is deferred.
- **Redis credentials:** They use the existing plaintext config model. No encryption is claimed, and errors/logs must redact secrets.
- **No confirmation:** Raw `DEL`, `FLUSHDB`, `FLUSHALL`, and other writes execute without a confirmation in Redis mode, as explicitly requested. The architecture document must record this exception.
- **Post-7.4 types/modules:** Not required. An unrecognized type must produce an explicit load error, so it is not mistaken for an empty value.

### Primary references used for design

- [Redis `SELECT` and logical databases](https://redis.io/docs/latest/commands/select/)
- [Redis `SCAN` iteration guarantees](https://redis.io/docs/latest/commands/scan/)
- [Redis keyspace guidance and `KEYS` warning](https://redis.io/docs/latest/develop/using-commands/keyspace/)
- [Redis Go client](https://github.com/redis/go-redis)
