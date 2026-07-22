# Raw Query Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a right-pane raw SQL editor that executes PostgreSQL statements and displays a bounded result or command status.

**Architecture:** `db.Database` will expose a driver-neutral `Execute` operation returning a `QueryResult`. The pgx adapter will execute submitted SQL through the simple protocol, collect at most 100 rows, and return its command tag. The Bubble Tea model will hold a separate query-panel model, route query-mode input to a multiline textarea, and ignore stale completion messages by session and request number.

**Tech Stack:** Go 1.26, Bubble Tea v2, Bubbles textarea v2, Lip Gloss v2, pgx v5, testify.

## Global Constraints

- `app` depends only on `internal/db`; PostgreSQL adapters must not import `app`.
- Execute I/O exclusively in `tea.Cmd` functions that return typed messages; keep `Update` free of I/O.
- Raw SQL accepts reads, writes, and DDL; do not attempt to filter or rewrite submitted SQL.
- Display no more than 100 result rows.
- Log every submitted SQL statement through the existing synchronized query logger.
- Preserve table-data state while the raw-query panel is active.
- Use `Ctrl+R` to open raw queries, `Ctrl+T` to return to table data, and `Ctrl+P` to execute; plain Enter inserts a newline.
- Do not create commits: the user will commit the completed work.

## File Structure

- Modify `internal/db/db.go`: define the bounded raw-result type and add `Execute(context.Context, string)` to `Database`.
- Modify `internal/db/postgres/postgres.go`: implement PostgreSQL raw execution, row collection, command tags, and logging.
- Create `internal/db/postgres/query_test.go`: integration coverage for row results, command results, limits, errors, and logging.
- Create `internal/app/query_panel.go`: own textarea state, query-result state, rendering, and resize behavior.
- Modify `internal/app/model.go`: retain the active right-pane mode and query-panel state.
- Modify `internal/app/keymap.go`: add `Ctrl+R`, `Ctrl+T`, and `Ctrl+P` bindings.
- Modify `internal/app/commands.go`: define raw-query completion messages and asynchronous execution command.
- Modify `internal/app/update.go`: route query-panel messages, start execution, process only current results, reset state on connection changes, and preserve existing table controls.
- Modify `internal/app/view.go`: render the selected right panel and expose panel-specific footer help.

---

### Task 1: Define and implement raw execution at the database boundary

**Files:**
- Modify: `internal/db/db.go`
- Modify: `internal/db/postgres/postgres.go`
- Create: `internal/db/postgres/query_test.go`

**Interfaces:**
- Consumes: `context.Context`, pgx `Rows`, and the existing `logger.Logger`.
- Produces:

  ```go
  const MaxQueryResultRows = 100

  type QueryResult struct {
      Columns    []string
      Rows       [][]any
      CommandTag string
  }

  type Database interface {
      Name() string
      ListTables(context.Context) ([]Table, error)
      GetRows(context.Context, Table, PageRequest) (RowPage, error)
      Execute(context.Context, string) (QueryResult, error)
      Close()
  }
  ```

- [ ] **Step 1: Write failing integration tests for row and command results**

  Create `internal/db/postgres/query_test.go` in package `postgres` so the test can replace the connected adapter's logger with an in-memory logger. Define `const rawQueryDSN = "postgres://db_tui@127.0.0.1:5433/chinook?sslmode=disable"` in this file because `chinookDSN` belongs to the external `postgres_test` package. Include these tests:

  ```go
  func TestExecuteReturnsRowsAndLogsSQL(t *testing.T) {
      database := connectTestDatabase(t)
      postgresDatabase := database.(*postgresql)
      var logOutput bytes.Buffer
      postgresDatabase.logger = logger.New(&logOutput)

      result, err := database.Execute(context.Background(), "SELECT 7 AS number")

      require.NoError(t, err)
      assert.Equal(t, []string{"number"}, result.Columns)
      assert.Equal(t, [][]any{{int32(7)}}, result.Rows)
      assert.Equal(t, "SELECT 1", result.CommandTag)
      assert.Contains(t, logOutput.String(), "SQL: SELECT 7 AS number")
  }

  func TestExecuteReturnsCommandTag(t *testing.T) {
      database := connectTestDatabase(t)

      result, err := database.Execute(context.Background(), "CREATE TEMPORARY TABLE raw_query_test (id integer)")

      require.NoError(t, err)
      assert.Empty(t, result.Columns)
      assert.Empty(t, result.Rows)
      assert.Equal(t, "CREATE TABLE", result.CommandTag)
  }
  ```

  Add the helper with a 10-second context and cleanup:

  ```go
  func connectTestDatabase(t *testing.T) db.Database {
      t.Helper()
      ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
      t.Cleanup(cancel)
      database, err := Connect(ctx, rawQueryDSN)
      require.NoError(t, err)
      t.Cleanup(database.Close)
      return database
  }
  ```

- [ ] **Step 2: Run the new tests to verify they fail**

  Run: `go test ./internal/db/postgres -run 'TestExecute(ReturnsRowsAndLogsSQL|ReturnsCommandTag)' -count=1`

  Expected: compilation failure because `Database.Execute`, `QueryResult`, and `postgresql.Execute` do not exist.

- [ ] **Step 3: Add the driver-neutral result contract**

  In `internal/db/db.go`, directly below `MaxPageSize`, add:

  ```go
  // MaxQueryResultRows is the largest number of rows returned for a raw query.
  const MaxQueryResultRows = 100

  // QueryResult contains the first rows returned by a SQL statement and its command status.
  // Columns is empty when the statement does not return rows.
  type QueryResult struct {
      Columns    []string
      Rows       [][]any
      CommandTag string
  }
  ```

  Add the documented `Execute(ctx context.Context, sql string) (QueryResult, error)` method between `GetRows` and `Close` in `Database`.

- [ ] **Step 4: Implement PostgreSQL execution**

  Add this method to `internal/db/postgres/postgres.go` after `GetRows`:

  ```go
  // Execute runs arbitrary SQL and returns its first 100 rows and command status.
  func (p *postgresql) Execute(ctx context.Context, sql string) (db.QueryResult, error) {
      p.logger.Log(sql)
      rows, err := p.pool.Query(ctx, sql, pgx.QueryExecModeSimpleProtocol)
      if err != nil {
          return db.QueryResult{}, fmt.Errorf("execute PostgreSQL query: %w", err)
      }
      defer rows.Close()

      result := db.QueryResult{CommandTag: ""}
      for index, description := range rows.FieldDescriptions() {
          if index == 0 {
              result.Columns = make([]string, 0, len(rows.FieldDescriptions()))
          }
          result.Columns = append(result.Columns, description.Name)
      }
      for rows.Next() {
          if len(result.Rows) == db.MaxQueryResultRows {
              break
          }
          values, err := rows.Values()
          if err != nil {
              return db.QueryResult{}, fmt.Errorf("read PostgreSQL query row: %w", err)
          }
          result.Rows = append(result.Rows, values)
      }
      rows.Close()
      if err := rows.Err(); err != nil {
          return db.QueryResult{}, fmt.Errorf("iterate PostgreSQL query rows: %w", err)
      }
      result.CommandTag = rows.CommandTag().String()
      return result, nil
  }
  ```

  Import `pgx` is already present; retain it for the simple-protocol option. Replace the field-description loop with a single allocation if preferred, but preserve the exact result semantics. `rows.Close()` before checking `Err` is required because it makes the pool connection reusable and makes the command tag available.

- [ ] **Step 5: Add limit and error regression tests**

  Add these tests to `query_test.go`:

  ```go
  func TestExecuteLimitsRows(t *testing.T) {
      database := connectTestDatabase(t)

      result, err := database.Execute(context.Background(), "SELECT generate_series(1, 101) AS number")

      require.NoError(t, err)
      assert.Len(t, result.Rows, db.MaxQueryResultRows)
      assert.Equal(t, int32(1), result.Rows[0][0])
      assert.Equal(t, int32(100), result.Rows[99][0])
  }

  func TestExecuteWrapsDatabaseErrors(t *testing.T) {
      database := connectTestDatabase(t)

      _, err := database.Execute(context.Background(), "SELECT * FROM raw_query_table_that_does_not_exist")

      assert.Error(t, err)
      assert.ErrorContains(t, err, "execute PostgreSQL query")
  }
  ```

- [ ] **Step 6: Run database-package tests**

  Run: `go test ./internal/db/postgres -count=1`

  Expected: PASS, including existing table paging tests and all four raw-execution tests. Ensure the local Compose PostgreSQL service is running first with `docker compose up -d`.

### Task 2: Build an isolated query-panel model and renderer

**Files:**
- Create: `internal/app/query_panel.go`
- Modify: `internal/app/layout.go`

**Interfaces:**
- Consumes: `db.QueryResult`, `textarea.Model`, `appLayout`, `panelStyle`, `formatCell`, and the existing data-grid sizing helpers.
- Produces:

  ```go
  type queryModel struct {
      editor   textarea.Model
      result   db.QueryResult
      loading  bool
      err      error
      request  uint64
  }

  func newQueryModel(layout appLayout) queryModel
  func (m *queryModel) reset(layout appLayout)
  func (m *queryModel) resize(layout appLayout)
  func (m *queryModel) beginExecute() uint64
  func (m *queryModel) finishExecute(result db.QueryResult, err error)
  func (m queryModel) view(layout appLayout, focused bool, connected bool, spinner string) string
  ```

- [ ] **Step 1: Implement `queryModel` and deterministic layout sizing**

  Create `query_panel.go` with a `textarea.Model` configured as follows:

  ```go
  func newQueryModel(layout appLayout) queryModel {
      editor := textarea.New()
      editor.Placeholder = "Write SQL…"
      editor.Prompt = ""
      editor.ShowLineNumbers = false
      styles := editor.Styles()
      styles.Focused.Text = styles.Focused.Text.Foreground(lipgloss.Color("252"))
      styles.Blurred.Text = styles.Blurred.Text.Foreground(lipgloss.Color("250"))
      styles.Focused.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
      styles.Blurred.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
      editor.SetStyles(styles)
      editor.MaxHeight = 0
      editor.MaxContentHeight = 0
      editor.SetHeight(queryEditorViewportHeight(layout))
      editor.SetWidth(max(1, layout.data.width-4))
      return queryModel{editor: editor}
  }

  func (m *queryModel) resize(layout appLayout) {
      m.editor.SetWidth(max(1, layout.data.width-4))
      m.editor.SetHeight(queryEditorViewportHeight(layout))
  }

  func querySectionHeight(layout appLayout) int {
      return max(3, layout.data.height/4)
  }

  func queryEditorViewportHeight(layout appLayout) int {
      return max(1, querySectionHeight(layout)-2)
  }

  func (m queryModel) resultHeight(layout appLayout) int {
      return max(1, layout.data.height-querySectionHeight(layout))
  }

  func (m *queryModel) reset(layout appLayout) {
      *m = newQueryModel(layout)
  }

  func (m *queryModel) beginExecute() uint64 {
      m.loading = true
      m.err = nil
      m.result = db.QueryResult{}
      m.request++
      return m.request
  }

  func (m *queryModel) finishExecute(result db.QueryResult, err error) {
      m.loading = false
      m.result = result
      m.err = err
  }
  ```

  Implement `view` as one right-side `panelStyle` containing a bold `RAW QUERY` heading, the textarea view, and then exactly one result state: disconnected help, spinner text, sanitized error, command tag, an empty result notice, or a grid. To render row results, instantiate `dataModel{page: db.RowPage{Columns: m.result.Columns, Rows: m.result.Rows}}`, use its `visibleColumnRange`, `visibleDataEnd`, and `dataGrid` methods with the result area's width and remaining height, then render `grid.String()`. This reuses the existing cell formatting, wrapping, width calculation, and table styling without coupling raw execution to table paging.

### Task 3: Route root-model input and asynchronous query execution

**Files:**
- Modify: `internal/app/model.go`
- Modify: `internal/app/keymap.go`
- Modify: `internal/app/commands.go`
- Modify: `internal/app/update.go`

**Interfaces:**
- Consumes: `db.Database.Execute(ctx, sql)`, `queryModel`, and `textarea.Model.Update`.
- Produces:

  ```go
  type rightPanel uint8
  const (
      panelData rightPanel = iota
      panelQuery
  )

  type queryFinishedMsg struct {
      result  db.QueryResult
      session uint64
      request uint64
      err     error
  }

  func executeQuery(database db.Database, sql string, session, request uint64) tea.Cmd
  func (m *Model) startQuery() tea.Cmd
  ```

- [ ] **Step 1: Add panel and key state**

  In `model.go`, define `rightPanel` beside `focusPane`, then add these fields to `Model`:

  ```go
  panel rightPanel
  query queryModel
  ```

  Initialize `query: newQueryModel(newAppLayout(defaultWidth, defaultHeight))` in `New`. In `keymap.go`, add `query`, `tableData`, and `executeQuery` fields to `keyMap` and bind them to `ctrl+r`, `ctrl+t`, and `ctrl+p` respectively. `Ctrl+R` must set `panelQuery`, set focus to the right pane, resize the editor, and return `m.query.editor.Focus()`. `Ctrl+T` must set `panelData` and retain all existing data state.

- [ ] **Step 2: Add the asynchronous command and stale-message guard**

  In `commands.go`, add `queryFinishedMsg` and:

  ```go
  func executeQuery(database db.Database, sql string, session, request uint64) tea.Cmd {
      return func() tea.Msg {
          ctx, cancel := context.WithTimeout(context.Background(), tableLoadTimeout)
          defer cancel()
          result, err := database.Execute(ctx, sql)
          return queryFinishedMsg{result: result, session: session, request: request, err: err}
      }
  }
  ```

  In `update.go`, add the lifecycle branch before `rowsLoadedMsg`:

  ```go
  case queryFinishedMsg:
      if msg.session != m.session || msg.request != m.query.request {
          return nil, true
      }
      m.query.finishExecute(msg.result, msg.err)
      return nil, true
  ```

  Add `startQuery` that returns nil when there is no database or the editor has only whitespace; otherwise calls `m.query.beginExecute()` and batches `executeQuery(m.database, m.query.editor.Value(), m.session, request)` with `m.startSpinner()`.

- [ ] **Step 3: Route query-mode input without breaking table controls**

  At the start of `updateKey`, retain global connection and quit keys. Then process query-panel commands in this order:

  ```go
  case key.Matches(msg, m.keys.query):
      m.panel = panelQuery
      return m.query.editor.Focus()
  case key.Matches(msg, m.keys.tableData):
      m.panel = panelData
      return nil
  case m.panel == panelQuery && key.Matches(msg, m.keys.executeQuery):
      return m.startQuery()
  case m.panel == panelQuery:
      editor, cmd := m.query.editor.Update(msg)
      m.query.editor = editor
      return cmd
  ```

  Keep the existing navigator and data-page key handling after that block, so it runs only in `panelData`. Update mouse-wheel and click handling to skip data-grid navigation when `m.panel == panelQuery`; navigator interaction remains available. On `tea.WindowSizeMsg`, call `m.query.resize(m.layout)`. On successful connection replacement and disconnect, call `m.query.reset(m.layout)` beside `m.data.reset()`.

### Task 4: Integrate the panel into the right-side view and validate the application

**Files:**
- Modify: `internal/app/view.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `Model.panel`, `queryModel.view`, and `dataModel.view`.
- Produces: panel-aware base view and keyboard documentation.

- [ ] **Step 1: Render the active panel and accurate footer help**

  In `baseView`, select the right content before joining the body:

  ```go
  rightPanel := m.data.view(m.dataStatus(), m.layout, m.focus == focusData)
  if m.panel == panelQuery {
      rightPanel = m.query.view(m.layout, m.focus == focusData, m.database != nil, m.spinner())
  }
  ```

  Use `rightPanel` in the `lipgloss.JoinHorizontal` call. Update `footerText` so a connected table-data view includes `Ctrl+R raw query`; raw-query mode includes `Ctrl+P execute` and `Ctrl+T table data`. Keep connection and quit shortcuts visible. Do not report table row paging status while query mode is active.

  Add a short `README.md` paragraph after the connection-modal instructions:

  ```markdown
  Press `Ctrl+R` to open the raw SQL panel. Enter any PostgreSQL statement and press `Ctrl+P` to run it; plain Enter adds a new line. Query results display at most 100 rows. Press `Ctrl+T` to return to the selected table's data view.
  ```

- [ ] **Step 2: Run repository validation without committing**

  Run: `scripts/validate.sh`

  Expected: PASS: no `gofmt` output, `go vet ./...`, normal tests, and race tests all exit zero. If the PostgreSQL integration service is unavailable, start it with `docker compose up -d` and rerun the validation command.

## Plan self-review

- Spec coverage: Task 1 covers the driver-neutral boundary, arbitrary PostgreSQL SQL, logging, errors, command statuses, and the 100-row cap. Task 2 provides independent query state and rendering. Task 3 provides shortcuts, async execution, stale-message protection, lifecycle resets, and resize/input routing. Task 4 integrates the right-pane replacement, footer, docs, and full validation. Per user direction, no new TUI/model/view tests are added.
- Placeholder scan: no unresolved requirements, placeholder markers, or unspecified test commands remain.
- Type consistency: every app task uses `db.QueryResult`, `Database.Execute(context.Context, string)`, `queryFinishedMsg`, and the same session/request stale-result identifiers defined in Task 1 or Task 3.
