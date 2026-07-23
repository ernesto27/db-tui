## 1. Navigator search state

- [x] 1.1 Extend the navigator model with the complete loaded table list, derived visible results, filter query input, and search-active state.
- [x] 1.2 Implement normalized, case-insensitive substring filtering that preserves database order and restores all tables for an empty query.
- [x] 1.3 Update selection, scroll bounds, range calculation, and mouse hit testing to operate safely on the visible result set, including zero matches.

## 2. Keyboard interaction and rendering

- [x] 2.1 Add the `Ctrl+F` key binding and route active-search keystrokes to the navigator filter input without triggering global shortcuts.
- [x] 2.2 Implement `Enter` to finish text entry while retaining the filter, and `Esc` to clear and exit search mode.
- [x] 2.3 Render the inline query, filtered count, no-match state, and revised footer help; account for the search row in navigator layout calculations.
- [x] 2.4 Keep table selection and data loading synchronized with filtered keyboard and mouse selection, without issuing row loads for no-match results.

## 3. Verification

- [x] 3.1 Run `gofmt`, `go vet ./...`, and `go build ./...`.
