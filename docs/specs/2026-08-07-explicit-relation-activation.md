# Explicit relation activation for row loading

## Problem Statement

Navigating db-tui's relation list currently loads rows whenever the highlighted relation changes. Moving through tables, ordinary views, or materialized views with the keyboard, mouse, section tabs, or search can therefore issue database queries for relations the user only passes over. Initial relation discovery also loads the first available relation automatically.

Users need to browse the navigator without causing database work. Loading a relation's rows should be an explicit action performed with `Enter` or a double-click, while ordinary navigation remains fast, predictable, and non-destructive.

## Solution

Separate the highlighted relation from the active relation. Navigation changes only the highlight. Pressing `Enter` with the navigator focused or double-clicking a highlighted row-browsable relation activates it and loads its first bounded page.

The active relation owns the data panel until another relation is explicitly activated. Its rows and title remain visible while the user browses elsewhere. The navigator retains its existing highlight cursor without a separate active-relation marker.

This behavior applies consistently to base tables, ordinary views, and materialized views. Existing pagination continues to load pages for the active relation, while table-specific actions continue to target the highlighted table without requiring a row load.

## User Stories

1. As a database user, I want to move through base tables without loading their rows, so that browsing the navigator does not issue unwanted queries.
2. As a database user, I want to move through ordinary views without loading their rows, so that view navigation has the same predictable behavior as table navigation.
3. As a database user, I want to move through materialized views without loading their rows, so that all row-browsable relation types share one interaction model.
4. As a database user, I want initial relation discovery to avoid loading rows, so that connecting to a database performs only object discovery until I request data.
5. As a database user, I want the first available relation to remain highlighted after discovery, so that I have an obvious activation target.
6. As a database user, I want to press `Enter` in the navigator to activate the highlighted relation, so that row loading reflects deliberate intent.
7. As a database user, I want to double-click a relation to activate it, so that mouse navigation offers a direct way to load its rows.
8. As a database user, I want activation to load the relation's first bounded page, so that a newly activated relation begins from a predictable position.
9. As a database user, I want arrow keys and `j`/`k` to change only the highlight, so that ordinary keyboard browsing never loads rows.
10. As a database user, I want Page Up, Page Down, Home, and End in the navigator to change only the highlight, so that large navigation jumps do not trigger queries.
11. As a database user, I want a single mouse click and mouse-wheel navigation in the navigator to change only the highlight, so that pointer-based browsing does not trigger queries.
12. As a database user, I want switching between relation sections to avoid loading rows, so that comparing available object types is inexpensive.
13. As a database user, I want editing or cancelling a relation filter to avoid loading rows, so that each intermediate search result does not cause a query.
14. As a database user, I want `Enter` in an open navigator search to finish the search and activate its highlighted result in one step, so that filtered activation does not require a second confirmation.
15. As a database user, I want `Enter` in the data panel to do nothing, so that activation cannot occur when my input focus is outside the navigator.
16. As a database user, I want pressing `Enter` on the already active relation to do nothing, so that activation does not unexpectedly refresh data or reset my current page.
17. As a database user, I want the currently active relation's data to remain visible while I highlight other relations, so that browsing does not destroy useful context.
18. As a database user, I want the data-panel title to name the active relation rather than the highlighted relation, so that displayed rows are never mislabeled.
19. As a database user, I want the data-panel title to identify the active relation without extra navigator decoration, so that relation ownership remains clear while the navigator stays visually simple.
20. As a database user, I want activating a different relation to switch ownership immediately, so that the active relation, panel title, loading state, and request all describe the same relation.
21. As a database user, I want the previous relation's rows cleared when a new relation is activated, so that stale rows are not displayed under the new relation's title.
22. As a database user, I want a row-load failure to remain associated with the newly active relation, so that the error identifies the operation I requested.
23. As a database user, I want navigation away from a loading active relation to leave its request valid, so that merely moving the highlight does not discard requested data.
24. As a database user, I want results from superseded row requests ignored, so that slow responses cannot overwrite the currently active relation or page.
25. As a database user, I want quickly activating relation A, relation B, and relation A again to accept only the newest A request, so that same-name and same-offset races cannot display obsolete data.
26. As a database user, I want table export, DDL, column inspection, and index inspection to target the highlighted table, so that metadata and export operations do not require loading rows first.
27. As a database user, I want row and page navigation in the data panel to continue operating on the active relation, so that explicit activation does not disrupt bounded pagination.
28. As a database user, I want changing database connections to clear the active relation and row data, so that state from one connection cannot appear under another.
29. As a database user, I want an instructional empty state before the first activation, so that I know to press `Enter` rather than mistaking unloaded data for an empty table.
30. As a database user, I want the footer to advertise `Enter` activation while the navigator is focused, so that the explicit-loading interaction is discoverable.

## Implementation Decisions

- The application model will store active relation identity independently from navigator cursor state.
- A relation identity consists of its relation type and name, allowing tables, ordinary views, and materialized views to share activation behavior without losing their domain distinction.
- Navigator selection remains the highlighted relation. Navigation handlers update this state without creating a row-loading command.
- Relation discovery may initialize the highlight but must not initialize the active relation or start a row query.
- A single navigator activation path will handle `Enter` from normal navigator focus and from navigator search.
- A double-click is two left clicks on the same relation within 500 milliseconds while the data panel is visible. The first click preserves single-click navigation; the second uses the same activation path as `Enter`. Raw-query mode does not activate navigator relations.
- Search `Enter` will finish search and attempt activation in the same update. If the highlighted relation is already active, the activation remains a no-op.
- Activating a different relation will set it active, reset the data-panel page state, and start its first-page load immediately.
- The row-loading path will receive or resolve the active relation rather than reading the current navigator highlight.
- Each row request will carry a monotonically increasing request ID in addition to connection session, relation identity, and page offset.
- A row result will be accepted only when its request ID, session, relation identity, and offset match the active request.
- Moving only the highlight will not change the active request and will not make its result stale.
- Activating another relation will supersede every earlier row request, including an earlier request for the same relation and offset.
- The data-panel status and title will derive from the active relation. Before activation, the panel will render an instructional state instead of an empty-page state.
- The navigator will retain its existing highlight cursor without adding an active-relation marker or section indicator.
- The footer will expose the `Enter` activation control when navigator activation is available.
- Table-specific actions will continue to derive their target from the highlighted table.
- Data-panel pagination will continue to derive its row source from the active relation and may load adjacent bounded pages without another activation.
- Connection replacement or removal will clear the active relation, outstanding row-request identity, and data-panel state.
- Database adapters and the driver-neutral database interface require no behavioral expansion; this is an application state and interaction change.
- Existing asynchronous command execution, cancellation boundaries, and maximum row-page size remain unchanged.
- This specification follows ADR 0010 and supersedes only the implicit row-loading portions of the earlier view and materialized-view navigation decisions.

## Testing Decisions

- The primary seam is the root Bubble Tea model's update and view behavior. Tests will send user and lifecycle messages through the existing model-update helper, execute returned commands only when necessary, and assert externally observable state, command creation, database calls, and rendered guidance.
- Tests should describe behavior rather than internal helper structure. They should verify which events do or do not issue row commands, which relation owns the data panel, what users see, and which asynchronous result wins.
- Table-driven root-update tests will cover keyboard navigation, section switching, search edits and cancellation, search activation, single and double mouse clicks, mouse-wheel navigation, initial discovery, normal activation, repeated activation, data-panel `Enter`, pagination, and connection reset.
- Root-view assertions will cover the pre-activation instruction, active data-panel title, the unchanged navigator highlight presentation, and footer activation help.
- Lifecycle tests will cover a result that remains valid after highlight-only navigation, a result made stale by activating a different relation, a session change, an offset mismatch, and the A-to-B-to-A same-offset race requiring request IDs.
- Tests will cover tables, ordinary views, and materialized views through the same activation scenarios, using table-driven cases where their expected behavior is identical.
- Tests will verify that table export and metadata actions still use the highlighted table when another relation is active.
- Tests will verify that navigation preserves the active page, offset, selected row, column offset, loading state, and error state until another relation is activated.
- Tests will verify that activating a different relation clears the prior page immediately and attributes loading or failure to the new active relation.
- Existing navigator unit tests remain appropriate for cursor clamping, filtering, per-section positions, and mouse hit testing. New behavior should prefer the root model seam; lower-level navigator tests are reserved for identity comparisons that cannot be observed clearly at the root seam.
- Existing update lifecycle, update key, navigator, and data-panel tests provide the closest prior art. No remote database or credentials are required; fake database implementations will provide deterministic row responses and call observations.
- The full repository test suite and validation script must pass, including race tests, because active requests and stale asynchronous results are central to the behavior change.

## Out of Scope

- Adding an explicit refresh command for the active relation.
- Treating repeated `Enter` as refresh or resetting the active relation to its first page.
- Activating a relation by a single mouse click, wheel input, section switch, or filter changes.
- Changing the behavior or availability of table export, DDL, column inspection, or index inspection.
- Adding view-specific or materialized-view-specific actions.
- Changing database discovery queries, relation grouping, or engine capability rules.
- Changing row ordering, page size, pagination boundaries, or database adapter behavior.
- Prefetching, caching, or persisting active relation data across connections or application restarts.
- Adding a new global keyboard shortcut for row refresh.

## Further Notes

- Domain terms in this specification—row-browsable relation, highlighted relation, active relation, relation identity, and stale row result—are defined in the project glossary.
- The accepted interaction and architecture decision is documented in ADR 0010.
- The navigator deliberately retains only its existing `>` highlight cursor; the data-panel title identifies the active relation.
- No issue-tracker publication is required for this specification.
