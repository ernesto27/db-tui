# Configurable page size from the settings modal

## Problem Statement

db-tui stores `maxPageSize` in its configuration file, but users currently
have to edit that file outside the application to change the number of rows
shown in a relation page. The existing value is silently clamped to 100, which
prevents users from choosing the page size they need.

Users need an in-app settings surface where they can set a valid page size,
save it for future sessions, and continue browsing without an unsolicited
reload or a pagination gap when the value changes.

## Solution

`Ctrl+S` opens a centered Settings modal containing the configured page-size
field. The field is prefilled with the currently configured value. A page size
is valid when it is a positive whole number, with no application-defined upper
bound.

`Enter` validates and saves the setting. Invalid input remains in the modal
with a clear validation error and is never persisted. `Esc` closes the modal
without changing the active configuration. A successful save persists the
value for later application launches but does not reload rows already on
screen. The next relation-page request uses the new size.

When a user moves forward from a page loaded with an earlier page size, the
new request begins after the last displayed row and uses the new size. This
preserves a continuous sequence of rows without repeats or gaps.

## User Stories

1. As a db-tui user, I want to open Settings with `Ctrl+S`, so that I can change application preferences without editing JSON manually.
2. As a db-tui user, I want Settings available while disconnected, so that I can prepare my preferred page size before connecting.
3. As a db-tui user, I want Settings available from both the relation browser and raw-query view, so that the preference is globally discoverable.
4. As a db-tui user, I want the modal to show my current configured page size, so that I can make a deliberate edit from the existing value.
5. As a db-tui user, I want a clearly labelled page-size field, so that I understand what the value controls.
6. As a db-tui user, I want to enter any positive whole-number page size, including values greater than 100, so that db-tui does not impose an arbitrary browsing limit.
7. As a db-tui user, I want zero rejected, so that every page request has a meaningful row count.
8. As a db-tui user, I want negative values rejected, so that the application never constructs an invalid row-page request.
9. As a db-tui user, I want non-numeric and non-integral input rejected, so that saved configuration is unambiguous.
10. As a db-tui user, I want invalid text to remain visible with an explanation, so that I can correct it without re-entering the whole value.
11. As a db-tui user, I want `Enter` to save a valid setting, so that the interaction matches other db-tui modals.
12. As a db-tui user, I want `Esc` to discard my unsaved edit, so that I can safely cancel.
13. As a db-tui user, I want a save failure reported in the modal, so that I do not mistake an in-memory change for a persisted preference.
14. As a db-tui user, I want the saved value to survive a restart, so that I do not need to configure it repeatedly.
15. As a db-tui user, I want the currently displayed relation page to stay intact after saving, so that changing a preference does not issue an unexpected query.
16. As a db-tui user, I want the next relation-page request to use my new page size, so that the preference takes effect at the next natural paging boundary.
17. As a db-tui user, I want forward paging after a size change to start immediately after the visible rows, so that I neither see duplicate rows nor skip rows.
18. As a db-tui user, I want Settings to block underlying keyboard and mouse interactions while open, so that editing a value cannot also alter my relation, query, or selection.
19. As a db-tui user, I want the footer to advertise `Ctrl+S` when no modal is open, so that Settings is discoverable.
20. As a db-tui user, I want the initial Settings modal to remain focused on page-size configuration, so that a simple preference does not become a crowded administration screen.

## Implementation Decisions

- Settings is an application-level modal with a dedicated modal state, following the existing modal precedence, overlay, focus, and keyboard-routing model.
- `Ctrl+S` is a global shortcut whenever no modal is already open. The modal consumes all normal input until it is saved or cancelled.
- The initial Settings modal contains one focused text field labelled `Page size`, prepopulated from the active configuration.
- `Enter` validates and persists the field; `Esc` closes the modal without changing application or persisted configuration. Tab navigation is unnecessary until Settings contains multiple editable controls.
- Validation accepts base-10 positive integers representable by the application’s page-request type. Empty, zero, negative, decimal, and non-numeric values are rejected in place.
- The persisted configuration keeps its existing `maxPageSize` field. Its normalization retains the configured positive value rather than clamping it to 100.
- Relation-page requests accept every positive limit representable by the page-request type. The fixed 100-row maximum is removed from the relation-page contract and each database adapter's relation-page validation.
- The default configured page size remains 100 for users without an existing preference.
- Raw-query result truncation is unchanged. This feature governs relation-page browsing only and does not turn raw-query results into a paged or unbounded result set.
- Saving does not start a row load, reset the selected row, change the active relation, or redraw the existing page with a different number of rows.
- The next requested relation page uses the saved page size. Forward paging computes its next offset from the current page's offset plus the number of rows actually displayed, preserving continuity across a page-size change.
- Backward paging retains its existing boundary semantics while using the newly configured limit for the newly requested page.
- Saving configuration remains synchronous only at the configuration boundary; Bubble Tea update handlers initiate it through a command that returns a typed success or failure message.
- The Settings modal reports persistence failures without applying the candidate page size to the root model.
- ADR 0013 defines the configured-page-size domain rule. This specification resolves its previously open modal interaction details for implementation.

## Testing Decisions

- The primary application seam is the root Bubble Tea model: tests should send keyboard and persistence-result messages through the existing update helper, then assert observable modal state, configuration state, returned commands, and rendered feedback.
- Root-model tests will verify opening Settings from disconnected, relation-browser, and raw-query states; prefilled current value; input capture; valid save; invalid input; cancel; save failure; and footer discoverability.
- Tests will assert behavior rather than private helper structure: the user-visible modal, persisted candidate, absence of unintended row loads, and the limits/offsets passed to the next row-load command are the contract.
- Configuration-package tests will verify that positive values above 100 survive loading and saving unchanged, while absent, zero, and negative persisted values normalize to the default page size.
- Adapter-level regression tests will verify that a positive relation-page limit above 100 is accepted by PostgreSQL, MySQL, SQLite, and Oracle. Existing tests for zero and negative limits remain required.
- Data-panel and root-model paging tests will cover a page-size change between requests: a page beginning at offset 0 with 100 displayed rows followed by a page size of 25 must request offset 100 and limit 25.
- Existing update-key, update-lifecycle, data-panel, configuration, and adapter test suites are prior art. Tests must use fake databases or local fixtures and must not require remote credentials.
- The full repository test suite, race tests, formatting, and vet checks must pass before the feature is considered complete.

## Out of Scope

- Changing the raw-query result limit or adding raw-query pagination.
- Adding more application settings beyond page size.
- Per-connection or per-database page-size preferences.
- Applying a changed page size by reloading, resizing, or otherwise modifying the currently displayed page.
- Adding mouse controls, a settings list, or multiple-field navigation to the initial modal.
- Adding an explicit relation refresh action.
- Imposing a separate configurable safety maximum for relation-page sizes.
- Altering database row ordering or the user's SQL.

## Further Notes

- The project glossary defines **configured page size** as the persisted row count for future relation-page requests.
- The term `maxPageSize` is retained for configuration compatibility, although the value is the requested page size rather than an upper bound.
- Very large configured page sizes can increase query time, transfer volume, memory use, and terminal rendering work; accepting that tradeoff is intentional.
- Per repository policy, this specification is local documentation and must not be published to GitHub or an external issue tracker unless explicitly requested.
