# Plan: TUI database connection modal

- **Date:** 2026-07-20
- **Domain(s):** frontend/TUI, backend connection lifecycle, configuration
- **Author:** plan-from-spec (reviewed with user)
- **Status:** Draft

## 1. Summary

Add a `Ctrl+L` connection modal to `db-tui`. It will let a user select the only currently supported engine (PostgreSQL), enter host, database name, port, username, password, or an optional DSN. A successful asynchronous connection test persists one PostgreSQL DSN to the existing private configuration file and immediately replaces the active database session.

## 2. Scope

### In scope

- A keyboard-modal PostgreSQL connection form, rendered over the current TUI view.
- Host, database name, port, username, password, optional DSN, and a one-option PostgreSQL engine display.
- Tab navigation, Shift+Tab reverse navigation, Enter submit, and Escape cancellation before submission.
- Individual-field validation and safe PostgreSQL DSN construction.
- DSN precedence: a non-blank DSN is connected and saved exactly as entered (after trimming); individual fields are ignored.
- Asynchronous test connection, inline failure feedback, persistence only after a successful connection, immediate session replacement, and old-session cleanup.
- Reading and writing one `postgresql.dsn` in `$HOME/.config/db-tui/config.json` with its existing `0600` file mode.
- Unit and Compose-backed integration coverage plus README updates.

### Out of scope / non-goals

- Engines other than PostgreSQL, multiple profiles/connections, connection history, TLS/SSL controls, SSH tunnels, keychain storage, and encrypted credentials.
- Persisting individual form values. Only the effective DSN is saved.
- Mouse interaction in the connection modal.
- Cancelling an in-progress connection attempt. Escape is disabled while it is being checked.

## 3. Resolved decisions

| # | Question | Decision |
|---|---|---|
| 1 | Which engine and connection slots are supported? | PostgreSQL only, with one `postgresql` configuration slot. |
| 2 | Which individual fields are required? | Host, database name, port, and username; password is optional. |
| 3 | Is host part of the form? | Yes. |
| 4 | Does the form accept a DSN? | Yes; the DSN is optional. A non-empty DSN overrides every individual field. |
| 5 | How is configuration stored? | Continue storing only `postgresql.dsn`; do not persist the individual form fields. |
| 6 | When is the replacement performed? | Connect and verify first, save the DSN second, then close the old session and adopt the new one. |
| 7 | What happens if validation, connection, or persistence fails? | Keep the modal open, preserve values, and show the error inline. |
| 8 | How does the form behave? | `Tab`/`Shift+Tab` change fields, `Enter` submits, `Esc` cancels before submission. |
| 9 | What happens during the connection check? | The modal remains open, shows progress, and disables Escape and form input until completion. |
| 10 | How does startup behave? | It auto-connects from the saved DSN, as it does today. |
| 11 | How are credentials protected for this phase? | Passwords remain plaintext inside the existing user-only (`0600`) config file. |

## 4. Design

`cmd/db-tui` remains the composition root: it loads the configured DSN, supplies `postgres.Connect` and `config.SavePostgreSQLDSN` callbacks to `app.New`, and closes the final app model on exit. `internal/app` never imports either the config or PostgreSQL adapter package.

The modal is an `internal/app` value object composed from six `bubbles/v2/textinput` models. The engine is rendered as a disabled one-choice selector (`Engine: PostgreSQL`) rather than an editable database driver control. The password input uses `textinput.EchoPassword`. It produces either a trimmed DSN or a URL-escaped `postgres://` DSN assembled with `net.JoinHostPort` and `net/url`.

The app creates the connection in a `tea.Cmd` with the existing five-second timeout. That command first calls the injected connector, then saves the DSN. If saving fails, it closes the newly created database and returns an error. On success, `Model` closes the old database, resets every table/row state field, increments a session generation, and starts table loading for the replacement. Load-result messages carry that generation, so stale work from a prior session cannot repaint the replacement session.

The base view and modal are composed with Lip Gloss layers: the current view remains visible underneath a centered, bordered modal. While a modal is open, normal navigation and mouse handling are bypassed; only modal key handling runs.

## 5. Interfaces & contracts

```go
// internal/config/config.go
// SavePostgreSQLDSN writes the only saved PostgreSQL connection.
func SavePostgreSQLDSN(dsn string) error

// internal/app/connection.go
type ConnectFunc func(context.Context, string) (db.Database, error)
type SaveDSNFunc func(string) error

// app.New receives concrete behavior from cmd, preserving app -> db dependency direction.
func New(
	database db.Database,
	databaseName string,
	startupErr error,
	savedDSN string,
	connect ConnectFunc,
	saveDSN SaveDSNFunc,
) Model
```

`SavePostgreSQLDSN` rejects a blank DSN and writes this exact shape:

```json
{
  "postgresql": {
    "dsn": "postgres://user:password@host:5432/database"
  }
}
```

The connection command returns a private message containing the `db.Database`, the effective DSN, and an error. On error, its `database` field is nil because the command owns and closes a database that cannot be persisted.

## 6. Behavior & states

```text
normal --Ctrl+L--> editing modal --Enter + valid--> checking
editing modal --Esc--> normal
checking --success--> save DSN -> close old session -> reload tables -> normal
checking --connect/save failure--> editing modal + inline error
checking --Esc/typing--> checking (ignored)
```

- `q` and `Ctrl+C` retain their application-wide quit behavior only when the modal is not open.
- A blank DSN triggers individual validation: non-empty trimmed host/database/username; decimal port in `1..65535`; password is unconstrained.
- A non-blank DSN is trimmed and passed through unchanged. The PostgreSQL connector is the definitive parser/validator for it.
- Opening the modal copies the currently saved DSN to the DSN input; discrete inputs start blank because they are not persisted.
- Repeated `Ctrl+L` while open does not reset user input. Modal mouse events do nothing.
- Connection commands carry a monotonically increasing attempt ID; a result that no longer matches the open modal is ignored and any supplied database is closed.

## 7. Implementation tasks

### Task 1 — Add explicit DSN persistence

- **Why:** The TUI needs a configuration writer that preserves the existing one-DSN schema and permissions.
- **Files & changes:**
  - `internal/config/config.go` (edit, after `Load`): factor JSON/file creation into `writeConfig`, reuse it in `createEmptyConfig`, and add:

    ```go
    // SavePostgreSQLDSN writes dsn as the saved PostgreSQL connection.
    func SavePostgreSQLDSN(dsn string) error {
    	if strings.TrimSpace(dsn) == "" {
    		return errors.New(`config field "postgresql.dsn" is required`)
    	}

    	path, err := configPath()
    	if err != nil {
    		return err
    	}
    	if err := os.MkdirAll(filepath.Dir(path), configDirectoryMode); err != nil {
    		return fmt.Errorf("create config directory: %w", err)
    	}
    	if err := writeConfig(path, Config{PostgreSQL: &PostgreSQLConfig{DSN: dsn}}); err != nil {
    		return err
    	}
    	return nil
    }

    func createEmptyConfig(path string) ([]byte, error) {
    	config := Config{PostgreSQL: &PostgreSQLConfig{}}
    	data, err := encodeConfig(config)
    	if err != nil {
    		return nil, err
    	}
    	if err := os.WriteFile(path, data, configFileMode); err != nil {
    		return nil, fmt.Errorf("create empty config: %w", err)
    	}
    	return data, nil
    }

    func writeConfig(path string, config Config) error {
    	data, err := encodeConfig(config)
    	if err != nil {
    		return err
    	}
    	if err := os.WriteFile(path, data, configFileMode); err != nil {
    		return fmt.Errorf("write config: %w", err)
    	}
    	return nil
    }

    func encodeConfig(config Config) ([]byte, error) {
    	data, err := json.MarshalIndent(config, "", "  ")
    	if err != nil {
    		return nil, fmt.Errorf("encode config: %w", err)
    	}
    	return append(data, '\n'), nil
    }
    ```

  - `internal/config/config_test.go` (edit): add `TestSavePostgreSQLDSN`, using `useTemporaryHome`, that saves a URL, checks `Load().PostgreSQL.DSN`, checks the exact JSON has no separate fields, confirms mode `0600` on non-Windows, and verifies blank input returns the existing required-field error.
- **Depends on:** —

### Task 2 — Build the modal model and DSN rules

- **Why:** Keep field behavior, focus management, validation, and DSN precedence isolated from the root TUI model.
- **Files & changes:**
  - `internal/app/connection_modal.go` (new): add a private `connectionModal` and fields for `host`, `databaseName`, `port`, `username`, `password`, and `dsn`. Configure its inputs with `bubbles/v2/textinput`; use `EchoPassword` for the password and a width calculated from the modal width. Implement these complete core methods:

    ```go
    func (m connectionModal) effectiveDSN() (string, error) {
    	if dsn := strings.TrimSpace(m.inputs[dsnInput].Value()); dsn != "" {
    		return dsn, nil
    	}

    	host := strings.TrimSpace(m.inputs[hostInput].Value())
    	databaseName := strings.TrimSpace(m.inputs[databaseNameInput].Value())
    	username := strings.TrimSpace(m.inputs[usernameInput].Value())
    	if host == "" {
    		return "", errors.New("host is required")
    	}
    	if databaseName == "" {
    		return "", errors.New("database name is required")
    	}
    	if username == "" {
    		return "", errors.New("username is required")
    	}

    	port, err := strconv.Atoi(strings.TrimSpace(m.inputs[portInput].Value()))
    	if err != nil || port < 1 || port > 65535 {
    		return "", errors.New("port must be between 1 and 65535")
    	}

    	user := url.User(username)
    	if password := m.inputs[passwordInput].Value(); password != "" {
    		user = url.UserPassword(username, password)
    	}

    	return (&url.URL{
    		Scheme: "postgres",
    		User:   user,
    		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
    		Path:   "/" + databaseName,
    	}).String(), nil
    }
    ```

    Implement `newConnectionModal(savedDSN string)`, `update(tea.Msg) (connectionModal, tea.Cmd)`, `focus(delta int) tea.Cmd`, and `view(width int) string`. `update` handles `tab`, `shift+tab`, `enter`, and `esc` before forwarding ordinary input to the focused `textinput.Model`; it returns private `submitConnectionMsg` or `cancelConnectionMsg` messages. When `connecting` is true it accepts no keys. `view` renders the non-focusable `Engine  [ PostgreSQL ]` selector, all labels and inputs, an inline `errorText`, and `Tab/Shift+Tab move  •  Enter connect  •  Esc cancel` (or `Checking connection…`) in a rounded border.

  - `internal/app/connection_modal_test.go` (new): table-test `effectiveDSN` for a DSN overriding deliberately invalid discrete values; each missing required field; invalid low, high, and non-numeric ports; blank password; URL escaping of user/password/database; and an IPv6 host. Add focus tests for Tab/Shift+Tab wraparound, and key tests that confirm Escape emits cancel while editing but is ignored while connecting.
- **Depends on:** —

### Task 3 — Inject connection operations and make session replacement safe

- **Why:** The app must connect without importing an adapter, preserve Bubble Tea's no-I/O-in-Update rule, and reject stale loads from a closed session.
- **Files & changes:**
  - `internal/app/model.go` (edit): import no concrete driver/config packages; add `connect ConnectFunc`, `saveDSN SaveDSNFunc`, `savedDSN string`, `modal *connectionModal`, `connectionAttempt uint64`, and `session uint64` to `Model`. Change `New` to the six-argument signature in section 5 and set `session: 1`.

    Extend `tablesLoadedMsg` and `rowsLoadedMsg` with `session uint64`; change `loadTables` and `loadRows` to accept and populate it. In their handlers, return early when `msg.session != m.session`. Update every existing call to pass `m.session`.

    At the start of `case tea.KeyPressMsg`, branch to `m.modal.update(msg)` whenever `m.modal != nil`; process its cancel and submit messages there, and do not run global/navigation key handling. Add the normal-mode branch:

    ```go
    case "ctrl+l":
    	modal := newConnectionModal(m.savedDSN)
    	m.modal = &modal
    	command = m.modal.focus(0)
    ```

    On `submitConnectionMsg`, call `m.modal.effectiveDSN()`. Put validation errors in `m.modal.errorText`; otherwise set `connecting`, clear the error, increment `connectionAttempt`, and return `connectAndSave(m.connect, m.saveDSN, dsn, m.connectionAttempt)`. On the result message, ignore mismatched attempts (closing a returned database), or restore editing with an inline error. On success: close a non-nil old `m.database`, assign the returned database, update `databaseName` and `savedDSN`, clear startup/table errors and tables/rows, increment `session`, close the modal, set loading/spinner state, and batch `loadTables(m.database, m.session)` with `m.startSpinner()`.

    Refactor the current `View` body construction into `renderBaseView() tea.View`. `View` calls it, and when `m.modal != nil`, replaces its content with `m.renderModalOverBase(base.Content)`. `renderModalOverBase` creates a `lipgloss.NewLayer(base).X(0).Y(0)` and a centered, higher-Z modal layer, then returns `lipgloss.NewCompositor(baseLayer, modalLayer).Render()`. Retain `AltScreen`, mouse mode, and window title from the base view.

    Add:

    ```go
    // Close releases the current database session, if any.
    func (m Model) Close() {
    	if m.database != nil {
    		m.database.Close()
    	}
    }
    ```

  - `internal/app/connection.go` (new): introduce the injected function types, result message, and the five-second asynchronous operation:

    ```go
    type ConnectFunc func(context.Context, string) (db.Database, error)
    type SaveDSNFunc func(string) error

    type connectionEstablishedMsg struct {
    	database db.Database
    	dsn      string
    	attempt  uint64
    	err      error
    }

    func connectAndSave(connect ConnectFunc, saveDSN SaveDSNFunc, dsn string, attempt uint64) tea.Cmd {
    	return func() tea.Msg {
    		ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
    		defer cancel()

    		database, err := connect(ctx, dsn)
    		if err != nil {
    			return connectionEstablishedMsg{attempt: attempt, err: err}
    		}
    		if err := saveDSN(dsn); err != nil {
    			database.Close()
    			return connectionEstablishedMsg{attempt: attempt, err: fmt.Errorf("save connection: %w", err)}
    		}
    		return connectionEstablishedMsg{database: database, dsn: dsn, attempt: attempt}
    	}
    }
    ```

  - `internal/app/model_connection_test.go` (new): use a small in-memory `db.Database` fake, injected connector, and save function. Cover: `Ctrl+L` opens an overlay with the saved DSN; normal navigation is suppressed; validation remains inline; a connector failure leaves the old session open; a save failure closes the candidate and leaves the old session open; success saves the DSN, closes exactly the old session, adopts the candidate, resets view data, and schedules a load; an old `tablesLoadedMsg`/`rowsLoadedMsg` session cannot overwrite replacement data; and `Model.Close` closes the final session.
- **Depends on:** Tasks 1–2.

### Task 4 — Wire the composition root and final cleanup

- **Why:** The executable must supply the actual adapter/config operations and release whichever session is active when Bubble Tea exits.
- **Files & changes:**
  - `cmd/db-tui/main.go` (edit): retain the existing startup `config.Load` and five-second `postgres.Connect`, but keep a `savedDSN` when loading succeeds. Replace the initial-database `defer` and discarded Run result with:

    ```go
    	model := app.New(
    		database,
    		databaseName,
    		startErr,
    		savedDSN,
    		postgres.Connect,
    		config.SavePostgreSQLDSN,
    	)
    	finalModel, err := tea.NewProgram(model).Run()
    	if finalApp, ok := finalModel.(app.Model); ok {
    		finalApp.Close()
    	}
    	if err != nil {
    		_, _ = fmt.Fprintf(os.Stderr, "db-tui: %v\\n", err)
    		os.Exit(1)
    	}
    ```

    Set `savedDSN = config.PostgreSQL.DSN` immediately before the existing initial `postgres.Connect` call.

  - `cmd/db-tui/main_test.go` (new): extract the small startup-construction function from `main` into a testable unexported function (returning `app.Model` is sufficient), then test that a valid loaded DSN is passed to the initial connector and to the form's saved DSN parameter. Do not test `os.Exit`.
- **Depends on:** Tasks 1 and 3.

### Task 5 — Document and exercise the feature end-to-end

- **Why:** Make the persisted-secret behavior and keyboard workflow clear, and verify the real PostgreSQL path against Compose.
- **Files & changes:**
  - `README.md` (edit, replace the current “Local configuration and secrets” example): retain the same JSON schema but explain that `Ctrl+L` opens a PostgreSQL connection modal, input can be a full DSN or individual host/database/port/username/password values, DSN wins when present, successful tests replace the live session and save the DSN, and passwords in this MVP are plaintext in a user-only `0600` file. Add the command to the keyboard workflow/help text in the same section.
  - `internal/app/connection_integration_test.go` (new): guarded by the existing local Compose setup, create an app model with `postgres.Connect` and a temporary-home wrapper around `config.SavePostgreSQLDSN`; populate individual modal fields with `127.0.0.1`, `5433`, `chinook`, `db_tui`, and blank password; execute the returned connection command; deliver its message to the model; then assert `databaseName == "chinook"`, `config.Load` returns a non-blank generated DSN, and `ListTables` can load Chinook tables. Ensure `model.Close()` runs through `t.Cleanup`.
  - `scripts/validate.sh` (no code change): run it after `gofmt` to execute formatting, vet, normal tests, and race tests; start `docker compose up -d` first for the integration tests.
- **Depends on:** Tasks 1–4.

## 8. Testing

- **Unit tests:**
  - Config save/reload, single-field schema, blank DSN rejection, and file permissions.
  - Form validation of each required input, port bounds, blank password, DSN precedence, and URL escaping.
  - Focus cycling, cancellation, disabled keys during checking, and password-masked rendering.
  - App state transitions with fake databases: failed connect/save retention, resource cleanup, success replacement, stale result protection, and final shutdown cleanup.
- **Integration tests:**
  - With `docker compose up -d`, submit the discrete PostgreSQL form to the local Chinook service, persist its generated DSN to an isolated config home, replace the app database, and list tables through the real pgx adapter.
  - Retain all existing `internal/db/postgres` Compose-backed connection/list/row tests; they continue proving that both the startup and modal DSNs use the same adapter contract.

## 9. Acceptance criteria

- Pressing `Ctrl+L` displays a centered modal above any current app state, including a startup-error state.
- The modal visibly has a one-choice PostgreSQL engine row and all six requested inputs; password characters are masked.
- Tab/Shift+Tab, Enter, and Escape work as specified; input and Escape are disabled while connecting.
- A pasted DSN wins over invalid individual values. Without one, missing required fields and invalid ports show inline errors without losing text.
- A failed connection or config write keeps the old database live and the modal open with an inline error.
- A successful test saves exactly one `postgresql.dsn`, closes the old database, immediately loads the new database's tables, and closes the final active database on quit.
- Startup still auto-connects using a valid saved DSN.
- `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` pass with the Compose service running for integration coverage.

## 10. Risks & open items

- Passwords are intentionally plaintext for this MVP. The config file is restricted to the user (`0600`), but it is not encrypted. Keychain/encrypted-secret support is explicitly deferred.
- The discrete form has no SSL/TLS options. Any SSL behavior is supplied only through a pasted DSN, which is deliberately the authoritative advanced-connection path.
