# ADR 0010: Require explicit navigator activation before loading rows

## Status

Accepted on 2026-08-07.

## Context

The navigator currently loads rows whenever its highlighted item changes. This couples inexpensive navigation to a database query: moving with the keyboard, mouse, or filter can issue row requests for relations the user only passes over.

The navigator contains three row-browsable relation types: base tables, ordinary views, and materialized views. Although their available actions differ, they share the pageable data panel and row-loading path.

This decision supersedes the implicit row-loading behavior described by ADRs 0006, 0007, and 0009. Their relation discovery, grouping, and capability decisions remain unchanged.

## Decision

Changing the highlighted navigator item will not load rows. A user must explicitly activate the highlighted relation with `Enter` or a double-click before db-tui loads its first row page.

This rule applies consistently to tables, ordinary views, and materialized views.

After connection and relation discovery, db-tui may highlight the first available relation, but it will leave the data panel unloaded. Initial discovery does not implicitly activate a relation; the first row query requires explicit activation.

After activation, the data panel remains owned by the active relation while the user navigates elsewhere. Moving the highlight does not clear or relabel the displayed data. Activating another relation transfers data-panel ownership to that relation and starts its first-page load.

The data-panel title names the active relation. The navigator retains its existing `>` highlight cursor without a separate active-relation marker, avoiding additional visual decoration while browsing.

Explicit activation governs row browsing only. Table-specific actions such as export, DDL, column inspection, and index inspection continue to target the highlighted table. They do not require that table to be active or its rows to have been loaded.

While navigator search is open, typing and filter-result changes only move the highlight. Pressing `Enter` finishes the search and activates the highlighted result in one step; it does not require a second `Enter`.

A double-click is two left clicks on the same highlighted relation within 500 milliseconds while the data panel is visible. The first click only changes the highlight; the second activates the relation and loads its first page. Double-clicks in raw-query mode do not load rows.

Activation is idempotent. Pressing `Enter` when the highlighted relation is already active does nothing: it does not reload rows, return to the first page, or alter the data-panel state.

When a different highlighted relation is activated, it becomes active immediately and the previous relation's page is cleared before the new request completes. The panel shows the new active relation's loading state and, if loading fails, its error rather than restoring the previous relation.

Asynchronous row results are validated against the active relation, connection session, and requested page. Moving only the highlight does not invalidate an in-flight result for the active relation. Activating another relation makes the prior relation's result stale.

`Enter` activates a relation only while the navigator has focus or while navigator search is active. Pressing `Enter` while the data panel has focus is a no-op, even if the navigator still has a highlighted relation.

Before the first activation, the data panel shows an instructional empty state telling the user to press `Enter` to load the highlighted relation. It does not describe the unloaded relation as an empty row page. The footer includes the `Enter` activation hint while the navigator is focused.

Connection changes clear both active-relation state and row data. Paging an already active relation remains unchanged: data-panel row navigation and page keys may load adjacent bounded pages without reactivation.

### Interaction summary

| Event | Highlight may change | Active relation changes | Starts a row query |
| --- | --- | --- | --- |
| Initial relation discovery | Yes | No | No |
| Arrow, `j`/`k`, Page Up/Down, Home/End in navigator | Yes | No | No |
| Single navigator mouse click or wheel | Yes | No | No |
| Section switch | Yes | No | No |
| Search/filter edit or cancel | Yes | No | No |
| `Enter` in navigator or search on a different relation | Maybe | Yes | Yes, first page |
| Double-click on a relation | Yes | Yes | Yes, first page |
| `Enter` on the active relation | No | No | No |
| `Enter` in data panel | No | No | No |
| Row/page navigation in data panel | No | No | When an adjacent page is needed |
| Connection change | Reset | Cleared | No row query during discovery |

### Implementation constraints

- Store active relation identity independently from navigator cursor state.
- Use the active relation for data-panel status, paging commands, and row-result acceptance.
- Use the highlighted relation for table-specific actions.
- Give row loads a monotonically increasing request ID. This prevents an older request from being accepted after the user activates another relation and then quickly returns to the first relation at the same offset.
- Preserve the existing bounded page size and asynchronous `tea.Cmd` execution model.
- Add focused update tests for every interaction category in the table, including in-flight activation races and same-relation `Enter` idempotence.

## Consequences

- Browsing the navigator will not issue a row query for every highlighted relation.
- Highlighting and activating a relation become distinct domain concepts.
- Connecting to a database and discovering its relations will not query relation rows.
- The data-panel label and asynchronous row-result validation must use the active relation rather than the highlighted relation.
- The data-panel title communicates which relation owns its rows while the navigator independently shows the current highlight.
- Relation metadata and export actions remain available without issuing a row query first.
- Search and ordinary navigator interaction share one explicit activation gesture.
- Explicit activation is not a row-refresh command.
- Loading and error feedback always identify the newly active relation rather than showing stale rows from its predecessor.
- Activation requires navigator context and cannot occur accidentally from the data panel.
- Existing bounded pagination remains available after activation.

## Alternatives considered

### Apply explicit activation only to tables

Rejected because all three relation types share the same row-browsing workflow. Giving views and materialized views different activation semantics would make navigation inconsistent.

### Automatically load the initially highlighted relation

Rejected because an implicit initial query would violate the explicit-activation rule and make `Enter` optional for the first relation only.

### Clear the data panel when the navigator highlight moves

Rejected because browsing would destructively remove useful context even though the user has not activated a different relation. Keeping the active relation visible lets users inspect the navigator without changing the data panel.

### Make table actions target only the active relation

Rejected because loading rows should not be a prerequisite for unrelated operations such as inspecting DDL, columns, or indexes. Table actions continue to follow the navigator highlight.

### Require a second Enter after finishing search

Rejected because the first `Enter` already expresses selection intent. Finishing search and activating its highlighted result in one step keeps activation consistent without issuing queries while the filter is merely being edited.

### Reload the active relation when Enter is pressed again

Rejected because `Enter` activates a different highlighted relation; it is not a refresh command. Repeated activation of the current relation remains a no-op and preserves the current page.

### Keep the previous relation visible until loading succeeds

Rejected because the active relation, panel title, and visible rows would temporarily describe different relations. Activation transfers ownership immediately, and any failure is displayed for the relation the user requested.

### Let Enter activate from the data panel

Rejected because the data panel can display one active relation while the navigator highlights another. Activation is a navigator action and requires navigator focus.
