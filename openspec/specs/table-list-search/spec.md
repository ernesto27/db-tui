## Purpose

Define keyboard-driven filtering of loaded database tables in the navigator.

## Requirements

### Requirement: Activate table-list search
The application SHALL activate table-list search when the user presses `Ctrl+F` while no modal is open. Activating search SHALL focus an inline query input in the navigator and SHALL route text-entry keys to that input.

#### Scenario: Open search from the data pane
- **WHEN** the user presses `Ctrl+F` while the data pane has focus
- **THEN** the navigator search input is focused and the navigator displays the current filtered result set

#### Scenario: Search input receives text
- **WHEN** table-list search is active and the user enters printable text or deletes text
- **THEN** the query input updates without invoking global navigator shortcuts

### Requirement: Filter loaded tables by query
The application SHALL filter the currently loaded public tables as the query changes. A table SHALL match when its name contains the trimmed query using case-insensitive comparison. An empty query SHALL display all loaded tables in their original order and SHALL not trigger database introspection.

#### Scenario: Match table names regardless of case
- **WHEN** the loaded table names include `InvoiceLine` and the user enters `line`
- **THEN** `InvoiceLine` is included in the visible result set

#### Scenario: Restore complete table list
- **WHEN** the user clears a non-empty query
- **THEN** all loaded tables are visible in their original order

### Requirement: Navigate and select filtered results
The application SHALL apply navigator selection, scrolling, pagination, and mouse hit testing to the visible filtered result set. It SHALL normalize the selected index and scroll offset whenever the result set changes, and a valid selected table SHALL continue to drive the existing table-data load behavior.

#### Scenario: Navigate a filtered result
- **WHEN** a query produces multiple results and the user moves down in the navigator
- **THEN** the highlight moves to the next visible matching table and the table-data pane loads that table

#### Scenario: Filter removes the selected table
- **WHEN** a query removes the currently selected table but leaves one or more matching tables
- **THEN** the navigator highlights a valid visible matching table and the data pane updates for that selection

### Requirement: Communicate search state and no-match results
The application SHALL render the active search query and the count of visible matching tables in the navigator or footer. When no tables match a non-empty query, it SHALL display a no-match state, expose no selectable table, and SHALL not request row data.

#### Scenario: Show no matching tables
- **WHEN** the query matches no loaded table names
- **THEN** the navigator displays a no-matching-tables message and has no selected table

#### Scenario: Show filtered count
- **WHEN** a query reduces the visible table list
- **THEN** the interface indicates the matching-table count relative to the loaded-table count

### Requirement: Complete or cancel table-list search
The application SHALL retain the active filter and return navigator key handling when the user presses `Enter` in the search input. It SHALL clear the query and restore the full table list when the user presses `Esc` in the search input.

#### Scenario: Keep filter after completing text entry
- **WHEN** the user presses `Enter` with a non-empty search query
- **THEN** the search input stops consuming text keys and navigation operates on the matching tables

#### Scenario: Cancel search
- **WHEN** the user presses `Esc` while editing a search query
- **THEN** the query is cleared, the full table list is restored, and navigator key handling resumes
