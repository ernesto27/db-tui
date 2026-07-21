# Plan: Refactor the Bubble Tea application architecture

- **Date:** 2026-07-20
- **Domain(s):** frontend/TUI architecture, application state management
- **Author:** plan-from-spec (reviewed with user)
- **Status:** Draft
- **Source of truth:** [`research/2026-07-20-scaling-bubble-tea-app-architecture.md`](../research/2026-07-20-scaling-bubble-tea-app-architecture.md)

## 1. Summary

Refactor `internal/app/model.go` from one 876-line mixed-responsibility file into a centralized Bubble Tea coordinator plus cohesive navigator, data-panel, layout, command, keymap, spinner, rendering, and text units. `Model` remains the only application-level `tea.Model`; navigator and data-panel types are ordinary stateful components whose operations are called by the root. The work preserves current behavior, dependencies, asynchronous session guards, bounded row paging, custom data-grid rendering, connection modal behavior, and package boundaries. No automated tests are included in this phase at the user's request; compilation, static checks, and a detailed manual verification matrix replace them temporarily.

## 2. Scope

### In scope

- Split root lifecycle, update routing, commands/messages, shell rendering, layout, keys, spinner, navigator, data state, data grid, and text utilities into focused files within `internal/app`.
- Keep one root `tea.Model` that owns database/session coordination, top-level focus, terminal lifecycle, global messages, and connection replacement.
- Introduce ordinary `navigatorModel` and `dataModel` value types that own their local invariants.
- Introduce one `appLayout` value as the source of truth for pane sizes and mouse hit-testing.
- Replace hard-coded root key comparisons with semantic Bubbles `key.Binding` values while preserving the exact current keys.
- Make the event-routing order explicit: lifecycle/result messages, connection modal, global keys, then focused pane input.
- Keep database I/O inside `tea.Cmd` functions and state mutation inside `Update` or synchronous component methods.
- Preserve session/attempt generation checks and cleanup of stale database connections.
- Preserve current UI text, focus behavior, mouse behavior, wheel debounce, paging, wrapping, sanitation, and terminal options.
- Run `gofmt`, `go vet ./...`, and `go build ./...`; then perform the manual verification checklist in section 8.

### Out of scope / non-goals

- Automated unit, integration, snapshot, race, or Compose-backed tests in this phase.
- A generic overlay interface or modal stack. The concrete connection modal remains until a second overlay creates a real abstraction boundary.
- A screen router or screen interface. Add it with the first genuinely different full-screen workflow.
- Replacing the custom data grid with `bubbles/table`, changing paging behavior, or adding data-grid caching.
- New panels, modals, query editing, schema browsing, connection features, or visible UI redesign.
- New dependencies or changes outside `internal/app`, except this plan document.
- Performance optimization without a measured rendering problem.

## 3. Resolved decisions

| # | Question | Decision |
|---|---|---|
| 1 | Should this be only a file split or the full recommended architecture? | Use the full recommended architecture from the referenced research. |
| 2 | How many application-level Bubble Tea models should exist? | Exactly one: root `app.Model`. |
| 3 | How should panels be represented? | Ordinary feature-owned value types with targeted methods; nested `tea.Model` implementations remain reserved for real reusable Bubbles. |
| 4 | Who owns shared database and async state? | Root `Model` owns the active database, session generation, connection attempt generation, table-load status, and command coordination. |
| 5 | Who owns table and row navigation invariants? | `navigatorModel` owns tables/selection/offset; `dataModel` owns page/row/column/loading/error state. |
| 6 | How are keys and focus handled? | Root owns semantic focus and dispatches semantic `key.Binding` matches to the active pane. |
| 7 | How is layout handled? | A single immutable-per-resize `appLayout` contains normalized dimensions and hit-test geometry. |
| 8 | Should the connection modal become a generic overlay stack now? | No. Keep it concrete and isolate modal routing; extract a stack/interface only after another overlay exists. |
| 9 | Should a screen router be introduced for future features? | No. Introduce a two-level screen/focus model only with the next full-screen workflow. |
| 10 | Should the data grid migrate to `bubbles/table`? | No; preserve server paging, horizontal column windows, wrapping, and variable rendered row heights. |
| 11 | Are behavior or dependencies allowed to change? | No visible behavior changes and no new dependencies. |
| 12 | What verification is required? | No automated tests for now; user will test manually. Implementation still runs formatting, vetting, compilation, and the explicit manual matrix. |
| 13 | What kind of code preview belongs in the plan? | Concrete target types, routing functions, method contracts, moves, and representative exact diffs in the repository's Go style. |

## 4. Design

### 4.1 Target ownership

```text
tea runtime
    |
    v
app.Model (only tea.Model)
    |-- database/session/connection coordination
    |-- lifecycle and typed-result routing
    |-- focus + semantic key dispatch
    |-- appLayout
    |-- connectionModal (concrete overlay)
    |-- navigatorModel
    |     `-- tables, selected index, visible offset
    `-- dataModel
          `-- row page, remote offset, row viewport/selection,
              column offset, row loading/error
```

The root is a coordinator, not a generic store. Each invariant has one owner. Cross-feature effects remain explicit: when navigator selection changes, the root asks `dataModel` to begin a load and returns the database command.

### 4.2 Target files

```text
internal/app/
  model.go                root state, New, Init, Close
  update.go               routing pipeline, global and focused input
  commands.go             table/row commands and result messages
  layout.go               normalized geometry and hit-testing values
  keymap.go               semantic key bindings
  view.go                 shell, header, footer, modal composition, panel style
  navigator.go            navigator state, invariants, hit-testing, rendering
  data_panel.go           row/column state and navigation requests
  data_grid.go            grid measurement and rendering
  spinner.go              shared spinner state and tick command
  text.go                 cell formatting, sanitation, ANSI truncation
  connection.go           unchanged connection settings/command concern
  connection_modal.go     concrete modal inputs/update/view
```

There is deliberately no `overlay.go`, `screen.go`, or new subpackage in this phase.

### 4.3 Routing pipeline

```text
incoming tea.Msg
    |
    +--> lifecycle/result? --> update root/feature state --> return command
    |
    +--> modal open? -------> modal routing only ---------> return command
    |
    +--> global key? -------> quit/open modal/focus switch
    |
    `--> focused pane/mouse -> navigator or data operation -> optional row load
```

`tea.WindowSizeMsg`, spinner ticks, table results, and row results are processed before the modal gate, preserving background sizing and loading. Connection completion stays in modal routing because a connection attempt cannot be cancelled while it is running under current behavior.

### 4.4 State contracts

```go
type navigatorModel struct {
	tables   []db.Table
	selected int
	offset   int
}

type dataModel struct {
	page         db.RowPage
	offset       int
	viewport     int
	selected     int
	columnOffset int
	loading      bool
	err          error
}

type rowLoadRequest struct {
	offset      int
	selectedRow int
}
```

Pane navigation returns a `rowLoadRequest` when it reaches a remote page boundary. It never calls the database. The root converts that request into `loadRows(...)` and starts the shared spinner.

## 5. Interfaces & contracts

### Root Bubble Tea contract

```go
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() tea.View
func (m Model) Close()
```

No other app-defined type implements `tea.Model`.

### Navigator contract

```go
func (m *navigatorModel) setTables(tables []db.Table)
func (m *navigatorModel) reset()
func (m *navigatorModel) move(delta, visibleRows int) bool
func (m *navigatorModel) selectIndex(index, visibleRows int) bool
func (m *navigatorModel) ensureVisible(visibleRows int)
func (m navigatorModel) selectedTable() (db.Table, bool)
func (m navigatorModel) tableAtMouse(msg tea.MouseClickMsg, layout appLayout) (int, bool)
func (m navigatorModel) view(status navigatorStatus, layout appLayout, focused bool) string
```

### Data-panel contract

```go
func (m *dataModel) reset()
func (m *dataModel) beginLoad(offset int)
func (m *dataModel) finishLoad(page db.RowPage, selectedRow int, err error, layout appLayout)
func (m *dataModel) moveUp(layout appLayout) (rowLoadRequest, bool)
func (m *dataModel) moveDown(layout appLayout) (rowLoadRequest, bool)
func (m *dataModel) scrollColumns(delta int, layout appLayout)
func (m *dataModel) ensureSelectedVisible(layout appLayout)
func (m dataModel) view(status dataStatus, layout appLayout, focused bool) string
```

### Command/result contract

`loadTables` and `loadRows` keep the existing five-second timeout. Result messages keep `session`; row results also keep table name, offset, and requested selected row. A result can mutate state only when all identity fields match the current root session and current navigator/data state.

### Modal contract for this phase

The concrete `*connectionModal` remains on `Model`. `Model.updateModal` remains the single interception point. Modal input does not reach global or pane routing. The existing private `submitConnectionMsg`, `cancelConnectionMsg`, and `connectionFinishedMsg` remain typed actions between modal/command/root.

## 6. Behavior & states

### Focus

```text
navigator --Right----------------------> data
data ------Left at first visible column-> navigator
data ------Left with columnOffset > 0---> previous column (focus unchanged)
```

### Table selection

```text
navigator move/select
    -> selected index changes?
       -> data.reset/beginLoad(offset=0)
       -> loadRows(selected table, offset=0, session)
```

### Data paging

```text
Up at first row and offset > 0
    -> request previous page, select its final row

Down at final loaded row and HasMore
    -> request next page, select its first row

PgUp/PgDown in data
    -> request previous/next remote page directly
```

### Modal interception

```text
background lifecycle/results -> always handled
mouse while modal open ------> ignored
all other input -------------> connection modal
connection success ----------> close old DB, adopt new DB, reset panes,
                                increment session, load tables
```

All current loading, empty, error, row status, footer, and spinner states remain textually and visually unchanged.

## 7. Implementation tasks

### Task 1 — Perform a behavior-preserving mechanical split

- **Why:** Reduce navigation cost before changing state ownership, so later diffs are reviewable.
- **Files & changes:**
  - `internal/app/model.go` (edit): retain package documentation, root constants that truly describe the app, `focusPane`, `Model`, `New`, `Init`, `Close`, and the `tea.Model` assertion. Remove functions only after moving them unchanged.
  - `internal/app/commands.go` (new): move `tablesLoadedMsg`, `rowsLoadedMsg`, `loadTables`, and `loadRows` unchanged. Move `tableLoadTimeout` and `rowPageSize` with them.
  - `internal/app/update.go` (new): move `Update`, `updateModal`, `startRowLoad`, and `acceptWheel` unchanged for this task.
  - `internal/app/view.go` (new): move `View`, `baseView`, `renderModalOverlay`, `footerText`, `panelStyle`, and `navigatorWidth` unchanged.
  - `internal/app/navigator.go` (new): initially move `renderNavigator`, `moveSelection`, `ensureSelectionVisible`, `maxNavigatorOffset`, `visibleNavigatorRows`, and `selectedTableName` as methods on `Model`; Task 3 changes their receiver.
  - `internal/app/data_panel.go` (new): initially move `resetRows`, `moveDataUp`, `moveDataDown`, `ensureSelectedRowVisible`, `scrollColumns`, `maxColumnOffset`, `dataPaneWidth`, and `dataPaneHeight` as methods on `Model`; Task 4 changes their receiver.
  - `internal/app/data_grid.go` (new): move `visibleColumnRange`, `visibleDataEnd`, `dataGrid`, `dataColumnWidths`, `totalTableWidth`, and `tableWidth` unchanged.
  - `internal/app/spinner.go` (new): move spinner frames, interval, message, tick/start/render methods.
  - `internal/app/text.go` (new): move `formatCell`, `sanitizeText`, and `truncateLabel` unchanged.

  The intermediate `model.go` preview is:

  ```go
  // Package app contains the root Bubble Tea application model.
  package app

  import (
  	"time"

  	tea "charm.land/bubbletea/v2"

  	"github.com/ernestoponce27/db-tui/internal/db"
  )

  const (
  	defaultWidth  = 100
  	defaultHeight = 24
  	wheelDebounce = 50 * time.Millisecond
  )

  type focusPane uint8

  const (
  	focusNavigator focusPane = iota
  	focusData
  )

  // Model is the root Bubble Tea application model.
  type Model struct {
  	// Existing fields remain unchanged during the mechanical split.
  }

  // New, Init, Close, and the tea.Model assertion remain here.
  ```

  No function logic or identifier changes in this task; imports are adjusted and all new files remain in package `app`.
- **Manual checkpoint:** Run `gofmt -w internal/app`, `go vet ./...`, and `go build ./...`. Launch the app and smoke-check startup, focus switching, one table selection, and opening/cancelling the modal.
- **Depends on:** —

### Task 2 — Introduce semantic key bindings and shared layout geometry

- **Why:** Stop duplicating key strings and width/coordinate formulas before extracting pane state.
- **Files & changes:**
  - `internal/app/keymap.go` (new): add the complete key map while retaining exact bindings:

    ```go
    package app

    import "charm.land/bubbles/v2/key"

    type keyMap struct {
    	connect    key.Binding
    	quit       key.Binding
    	focusLeft  key.Binding
    	focusRight key.Binding
    	up         key.Binding
    	down       key.Binding
    	pageUp     key.Binding
    	pageDown   key.Binding
    	home       key.Binding
    	end        key.Binding
    }

    func defaultKeyMap() keyMap {
    	return keyMap{
    		connect:    key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "connect")),
    		quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
    		focusLeft:  key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left/focus tables")),
    		focusRight: key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right/focus data")),
    		up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
    		down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
    		pageUp:     key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "previous page")),
    		pageDown:   key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "next page")),
    		home:       key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "first table")),
    		end:        key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "last table")),
    	}
    }
    ```

  - `internal/app/layout.go` (new): make all view and hit-test geometry derive from one value:

    ```go
    package app

    const (
    	bodyStartRow          = 1
    	navigatorListStartRow = 4
    )

    type paneRect struct {
    	x      int
    	y      int
    	width  int
    	height int
    }

    type appLayout struct {
    	width             int
    	height            int
    	bodyHeight        int
    	navigator         paneRect
    	data              paneRect
    	navigatorListY    int
    	navigatorListRows int
    }

    func newAppLayout(width, height int) appLayout {
    	width = max(width, 64)
    	height = max(height, 16)
    	navigatorWidth := 26
    	if width < 80 {
    		navigatorWidth = 20
    	}

    	bodyHeight := height - 4
    	return appLayout{
    		width:      width,
    		height:     height,
    		bodyHeight: bodyHeight,
    		navigator: paneRect{x: 0, y: bodyStartRow, width: navigatorWidth, height: bodyHeight},
    		data: paneRect{
    			x: navigatorWidth + 1, y: bodyStartRow,
    			width: width - navigatorWidth - 1, height: bodyHeight,
    		},
    		navigatorListY:    bodyStartRow + navigatorListStartRow,
    		navigatorListRows: max(1, height-7),
    	}
    }

    func (l appLayout) mouseInNavigator(x int) bool {
    	return x >= 0 && x < l.navigator.width
    }

    func (l appLayout) clickableNavigatorX(x int) bool {
    	return x > l.navigator.x && x < l.navigator.width-1
    }
    ```

  - `internal/app/model.go` (edit, `Model` and `New`): replace `width`/`height` with `layout appLayout`, add `keys keyMap`, and initialize both:

    ```diff
    - width  int
    - height int
    + layout appLayout
    + keys   keyMap
    ```

    ```go
    layout: newAppLayout(defaultWidth, defaultHeight),
    keys:   defaultKeyMap(),
    ```

  - `internal/app/update.go` and `view.go` (edit): replace raw `m.width`, `m.height`, and `navigatorWidth(...)` formulas with `m.layout`; on resize assign `m.layout = newAppLayout(msg.Width, msg.Height)`. Replace root `msg.String()` comparisons with `key.Matches(msg, m.keys.<binding>)`. Do not change modal-internal Tab/Enter/Escape strings because those bindings are private to one form.
- **Manual checkpoint:** Resize across widths 79/80 and heights below/above 16; verify the current minimum layout, navigator width switch, click target, wheel target, and footer remain unchanged.
- **Depends on:** Task 1.

### Task 3 — Extract navigator state and invariants

- **Why:** Table list ownership is the first feature boundary and drives data-load requests without owning database I/O.
- **Files & changes:**
  - `internal/app/navigator.go` (edit): change root methods into the complete `navigatorModel` core below; retain the existing rendering body, replacing `m.tables`, `m.selected`, and `m.navigatorOffset` with `m.tables`, `m.selected`, and `m.offset` on the new receiver.

    ```go
    type navigatorModel struct {
    	tables   []db.Table
    	selected int
    	offset   int
    }

    func (m *navigatorModel) reset() {
    	m.tables = nil
    	m.selected = 0
    	m.offset = 0
    }

    func (m *navigatorModel) setTables(tables []db.Table) {
    	m.tables = tables
    	m.selected = 0
    	m.offset = 0
    }

    func (m navigatorModel) selectedTable() (db.Table, bool) {
    	if len(m.tables) == 0 || m.selected < 0 || m.selected >= len(m.tables) {
    		return db.Table{}, false
    	}
    	return m.tables[m.selected], true
    }

    func (m *navigatorModel) move(delta, visibleRows int) bool {
    	return m.selectIndex(m.selected+delta, visibleRows)
    }

    func (m *navigatorModel) selectIndex(index, visibleRows int) bool {
    	if len(m.tables) == 0 {
    		m.selected = 0
    		m.offset = 0
    		return false
    	}
    	previous := m.selected
    	m.selected = min(max(index, 0), len(m.tables)-1)
    	m.ensureVisible(visibleRows)
    	return m.selected != previous
    }

    func (m *navigatorModel) ensureVisible(visibleRows int) {
    	visibleRows = max(1, visibleRows)
    	if len(m.tables) == 0 {
    		m.selected = 0
    		m.offset = 0
    		return
    	}
    	m.selected = min(max(m.selected, 0), len(m.tables)-1)
    	if m.selected < m.offset {
    		m.offset = m.selected
    	}
    	if m.selected >= m.offset+visibleRows {
    		m.offset = m.selected - visibleRows + 1
    	}
    	m.offset = min(max(m.offset, 0), max(0, len(m.tables)-visibleRows))
    }

    func (m navigatorModel) tableAtMouse(msg tea.MouseClickMsg, layout appLayout) (int, bool) {
    	visibleIndex := msg.Y - layout.navigatorListY
    	index := m.offset + visibleIndex
    	valid := msg.Button == tea.MouseLeft &&
    		layout.clickableNavigatorX(msg.X) &&
    		visibleIndex >= 0 && visibleIndex < layout.navigatorListRows &&
    		index >= 0 && index < len(m.tables)
    	return index, valid
    }
    ```

  - `internal/app/model.go` (edit): replace `tables`, `selected`, and `navigatorOffset` with:

    ```go
    navigator navigatorModel
    ```

  - `internal/app/update.go` (edit): change all table access through `m.navigator`. When a selection method returns `true`, call the root `startRowLoad(0, 0)`. On `tablesLoadedMsg`, call `m.navigator.setTables(msg.tables)`. On reset/reconnection, call `m.navigator.reset()`.
  - `internal/app/view.go` (edit): use `m.navigator.tables`, `m.navigator.offset`, and `m.navigator.selectedTable()` for footer counts and selected-table text; delegate navigator rendering to `m.navigator.view(...)` with a small status value containing database name, startup error, loading, and table-load error.
- **Manual checkpoint:** Verify arrow/J/K, PgUp/PgDown, Home/End, clicking, wheel selection, clamping at both ends, selection visibility after resizing, and one row reload per actual selection change.
- **Depends on:** Task 2.

### Task 4 — Extract data-panel state, navigation, and grid rendering

- **Why:** Keep row/column paging invariants together while leaving remote I/O at the root.
- **Files & changes:**
  - `internal/app/data_panel.go` (edit): introduce `dataModel` and use `rowLoadRequest` for remote boundaries:

    ```go
    type rowLoadRequest struct {
    	offset      int
    	selectedRow int
    }

    type dataModel struct {
    	page         db.RowPage
    	offset       int
    	viewport     int
    	selected     int
    	columnOffset int
    	loading      bool
    	err          error
    }

    func (m *dataModel) reset() {
    	*m = dataModel{}
    }

    func (m *dataModel) beginLoad(offset int) {
    	m.page = db.RowPage{}
    	m.offset = offset
    	m.viewport = 0
    	m.selected = 0
    	m.columnOffset = 0
    	m.loading = true
    	m.err = nil
    }

    func (m *dataModel) finishLoad(page db.RowPage, selectedRow int, err error, layout appLayout) {
    	m.loading = false
    	m.err = err
    	if err != nil {
    		return
    	}
    	m.page = page
    	m.viewport = 0
    	m.selected = min(selectedRow, max(0, len(page.Rows)-1))
    	m.columnOffset = 0
    	m.ensureSelectedVisible(layout)
    }

    func (m *dataModel) moveUp(layout appLayout) (rowLoadRequest, bool) {
    	if m.loading {
    		return rowLoadRequest{}, false
    	}
    	if m.selected > 0 {
    		m.selected--
    		m.ensureSelectedVisible(layout)
    		return rowLoadRequest{}, false
    	}
    	if m.offset > 0 {
    		return rowLoadRequest{offset: max(0, m.offset-rowPageSize), selectedRow: rowPageSize - 1}, true
    	}
    	return rowLoadRequest{}, false
    }

    func (m *dataModel) moveDown(layout appLayout) (rowLoadRequest, bool) {
    	if m.loading {
    		return rowLoadRequest{}, false
    	}
    	if m.selected < len(m.page.Rows)-1 {
    		m.selected++
    		m.ensureSelectedVisible(layout)
    		return rowLoadRequest{}, false
    	}
    	if m.page.HasMore {
    		return rowLoadRequest{offset: m.offset + rowPageSize}, true
    	}
    	return rowLoadRequest{}, false
    }

    func (m *dataModel) scrollColumns(delta int, layout appLayout) {
    	m.columnOffset = min(max(m.columnOffset+delta, 0), max(0, len(m.page.Columns)-1))
    	m.ensureSelectedVisible(layout)
    }
    ```

  - `internal/app/data_grid.go` (edit): change `Model` receivers to `dataModel`; replace `m.rowPage`, `m.rowSelected`, and `m.columnOffset` with `m.page`, `m.selected`, and `m.columnOffset`. Pass `layout.data.width` and `layout.data.height` into visibility functions. Keep the existing width allocation, wrapping, alternating row style, selected-row style, and table-border calculations byte-for-byte otherwise.
  - `internal/app/model.go` (edit): replace all row/page fields with:

    ```go
    data dataModel
    ```

  - `internal/app/update.go` (edit): make `startRowLoad` call `m.data.beginLoad(offset)` and turn returned `rowLoadRequest` values into `startRowLoad(request.offset, request.selectedRow)`. Validate row results against `m.data.offset`; call `m.data.finishLoad(...)` on accepted results.
  - `internal/app/view.go` (edit): delegate data rendering to `m.data.view(...)`, passing selected table name, root loading/error status, spinner text, and `m.layout`.
- **Manual checkpoint:** Verify row movement, selected-row visibility with wrapped cells, previous/next remote pages, PgUp/PgDown, column scrolling, loading/error/empty page states, `NULL`, bytes, and control-character sanitation.
- **Depends on:** Task 3.

### Task 5 — Make root routing an explicit coordinator

- **Why:** Future panels and overlays need one obvious input-precedence policy rather than a growing mixed switch.
- **Files & changes:**
  - `internal/app/model.go` (edit): converge on this root shape:

    ```go
    type Model struct {
    	database          db.Database
    	databaseName      string
    	savedConnection   ConnectionSettings
    	connect           ConnectFunc
    	saveConnection    SaveConnectionFunc
    	modal             *connectionModal
    	connectionAttempt uint64
    	session           uint64

    	loading      bool
    	startupErr   error
    	tableLoadErr error
    	navigator    navigatorModel
    	data         dataModel

    	spinnerFrame   int
    	spinnerRunning bool
    	layout          appLayout
    	keys            keyMap
    	focus           focusPane
    	lastWheelAt     time.Time
    	lastWheelButton tea.MouseButton
    }
    ```

  - `internal/app/update.go` (edit): make `Update` visibly enforce the routing order:

    ```go
    func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    	if command, handled := m.updateLifecycle(msg); handled {
    		return m, command
    	}

    	if m.modal != nil {
    		switch msg.(type) {
    		case tea.MouseClickMsg, tea.MouseWheelMsg:
    			return m, nil
    		default:
    			return m.updateModal(msg)
    		}
    	}

    	switch msg := msg.(type) {
    	case tea.KeyPressMsg:
    		return m, m.updateKey(msg)
    	case tea.MouseClickMsg:
    		return m, m.updateMouseClick(msg)
    	case tea.MouseWheelMsg:
    		return m, m.updateMouseWheel(msg)
    	default:
    		return m, nil
    	}
    }
    ```

    `updateLifecycle` handles `spinnerTickMsg`, `tablesLoadedMsg`, `rowsLoadedMsg`, and `tea.WindowSizeMsg`. `updateModal` retains submit/cancel/connection-finished behavior. `updateKey` first handles connect/quit/focus bindings, then delegates navigation by `m.focus`. Mouse helpers use only `m.layout` and pane methods. Commands are returned, never executed in these functions.

    Keep the stale-result guard structurally equivalent to:

    ```go
    table, selected := m.navigator.selectedTable()
    if msg.session != m.session || !selected ||
    	msg.tableName != table.Name || msg.offset != m.data.offset {
    	return nil, true
    }
    ```

  - `internal/app/connection_modal.go` (edit only if necessary for imports): keep the concrete modal and its current typed messages. Do not add an overlay interface.
- **Manual checkpoint:** Verify modal precedence specifically: open it while tables/rows are loading, resize behind it, confirm loading completes, confirm mouse cannot affect background panes, confirm failed connection stays in the modal, and confirm successful replacement resets both pane models.
- **Depends on:** Tasks 2–4.

### Task 6 — Finish shell/view composition around feature views

- **Why:** Make new panel addition a layout/composition concern rather than a reason to edit data-grid internals.
- **Files & changes:**
  - `internal/app/view.go` (edit): keep the root view declarative and delegate body rendering:

    ```go
    func (m Model) View() tea.View {
    	view := m.baseView()
    	if m.modal != nil {
    		view.Content = m.renderModalOverlay(view.Content)
    	}
    	return view
    }

    func (m Model) baseView() tea.View {
    	header := m.renderHeader()
    	body := lipgloss.JoinHorizontal(
    		lipgloss.Top,
    		m.navigator.view(m.navigatorStatus(), m.layout, m.focus == focusNavigator),
    		" ",
    		m.data.view(m.dataStatus(), m.layout, m.focus == focusData),
    	)
    	footer := lipgloss.NewStyle().
    		Width(m.layout.width).
    		Padding(0, 1).
    		Foreground(lipgloss.Color("245")).
    		Render(m.footerText())

    	view := tea.NewView(strings.Join([]string{header, body, footer}, "\n"))
    	view.AltScreen = true
    	view.MouseMode = tea.MouseModeCellMotion
    	view.WindowTitle = "db-tui"
    	return view
    }
    ```

  - `internal/app/navigator.go` (edit): `view` accepts only `navigatorStatus`, `appLayout`, and focus; it must not read root `Model`.
  - `internal/app/data_panel.go` (edit): `view` accepts only `dataStatus`, `appLayout`, and focus; it must not read root `Model` or database dependencies.
  - `internal/app/text.go` and `data_grid.go` (edit): keep pure formatting/rendering helpers free of root state.

  Status inputs are explicit and read-only:

  ```go
  type navigatorStatus struct {
    	databaseName string
    	startupErr   error
    	loading      bool
    	tableLoadErr error
  }

  type dataStatus struct {
    	tableName     string
    	startupErr    error
    	tablesLoading bool
    	tableLoadErr  error
    	spinner       string
  }
  ```

  Preserve all existing strings and styles; this task changes ownership and call shape only.
- **Manual checkpoint:** Compare normal, loading, startup-error, table-error, no-table, row-loading, row-error, empty-row, populated-grid, and modal-overlay screens against the pre-refactor app.
- **Depends on:** Task 5.

### Task 7 — Remove compatibility leftovers and perform final manual validation

- **Why:** Ensure the result has clear ownership rather than duplicate old and new paths.
- **Files & changes:**
  - `internal/app/model.go`: remove obsolete flat navigator/data/layout fields and unused constants/imports.
  - `internal/app/update.go`: remove old mixed-switch helpers after all call sites use feature components.
  - `internal/app/view.go`: remove `navigatorWidth`, `dataPaneWidth`, and `dataPaneHeight` after `appLayout` owns them.
  - `internal/app/*.go`: add Go doc comments only to exported identifiers; keep internal component names unexported.
  - Run:

    ```sh
    gofmt -w internal/app
    go vet ./...
    go build ./...
    ```

    Do not run `scripts/validate.sh`, `go test ./...`, or `go test -race ./...` in this phase because the user explicitly chose manual testing only.

  - Inspect with:

    ```sh
    rg 'm\.(tables|selected|navigatorOffset|rowPage|rowOffset|rowViewport|rowSelected|columnOffset|rowsLoading|rowsErr|width|height)' internal/app
    rg 'msg\.String\(\)' internal/app
    ```

    The first search must have no obsolete root-state matches. The second may match only modal-local form bindings or an explicitly documented exception.
- **Manual checkpoint:** Complete every scenario in section 8 and record any difference before accepting the refactor.
- **Depends on:** Tasks 1–6.

## 8. Verification

### Automated unit tests

Omitted for this phase at the user's explicit request. This is not because unit testing is inapplicable; navigator clamping, row paging, variable-height visibility, stale-result rejection, and routing priority are all good future unit-test targets.

### Automated integration tests

Omitted for this phase at the user's explicit request. The existing PostgreSQL adapter tests are unchanged, but no new wired app-model coverage will be added or run as part of this work.

### Required compile/static verification

- `gofmt -w internal/app`
- `go vet ./...`
- `go build ./...`

### Manual verification matrix

1. Start PostgreSQL with `docker compose up -d` and launch with `go run ./cmd/db-tui`.
2. Confirm the initial table list loads, the first table is selected, and its first row page loads.
3. Move navigator selection with Up/Down and J/K; confirm data reloads only when selection changes.
4. Exercise Home, End, navigator PgUp/PgDown, clicks, and wheel input at list boundaries.
5. Switch focus with Left/Right; confirm Left scrolls data columns before returning focus to navigator.
6. Move through rows with Up/Down and J/K; cross both remote page boundaries and use data PgUp/PgDown.
7. Resize below and above 80 columns, and below the 64x16 logical minimum; confirm selection remains visible and mouse hit areas align.
8. Inspect a table containing `NULL`, byte data, long wrapped cells, and enough columns to scroll horizontally.
9. Open the connection modal with Ctrl+L; verify Tab/Shift+Tab, Enter, Escape, password masking, validation errors, and mouse suppression.
10. Open the modal while a load spinner is active and resize; confirm background lifecycle messages still update without accepting background input.
11. Submit an invalid connection; confirm entered values and old session remain.
12. Submit the local Chinook connection; confirm the old session is replaced, navigator/data state resets, and new tables/rows load.
13. Quit with Q and Ctrl+C in normal mode; confirm current behavior while the modal is open also remains unchanged.
14. Exercise startup-error, no-table, table-load-error, row-load-error, and empty-page displays when practical with local/manual fakes or temporary connection changes.

## 9. Acceptance criteria

- `Model` is the only app-defined type satisfying `tea.Model`.
- Root `Model` no longer directly stores navigator selection/offset or data page/row/column fields.
- Navigator and data invariants each have one owning type and no database I/O.
- All database work remains in `tea.Cmd`; result messages retain current generation and identity checks.
- Event routing visibly follows lifecycle/results, modal, global keys, then focused pane input.
- One `appLayout` supplies view dimensions and mouse hit-test geometry.
- Root key handling uses semantic Bubbles key bindings with exactly the current keys.
- The connection modal remains concrete; no speculative overlay stack or screen router is introduced.
- The custom data-grid behavior and current visible UI remain unchanged.
- No new dependencies or package-boundary violations are introduced.
- `gofmt`, `go vet ./...`, and `go build ./...` succeed.
- The user completes and accepts the manual verification matrix.

## 10. Risks & open items

- **Accepted regression risk:** automated tests are intentionally deferred. Manual testing may miss stale-message races, layout edges, and subtle paging regressions. The research report's test backlog should be implemented in a later task.
- **Large structural change:** grouping state and changing receivers touches most `internal/app` call sites. The mechanical split must be committed and manually checked separately before state extraction.
- **Preview details may shift mechanically:** imports and exact status argument placement can change during compilation cleanup, but ownership, routing order, public behavior, and file responsibilities are fixed by this plan.
- **Overlay evolution deferred:** add an `overlay` interface/stack only when a second modal, nesting, or command palette exists.
- **Screen evolution deferred:** add screen state only with a concrete second full-screen workflow.
- **Rendering performance deferred:** repeated grid measurement remains unchanged; benchmark before adding caches.
