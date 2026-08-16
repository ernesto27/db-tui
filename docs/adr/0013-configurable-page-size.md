# ADR 0013: Configure row-page size in the TUI

## Status

Proposed on 2026-08-15.

## Context

db-tui persists a `maxPageSize` value in its application configuration, but
users can change it only by editing the JSON configuration file outside the
application. Despite its name, the value is the exact number of rows requested
for a browsed relation page. The application currently clamps it to the
database package's hard limit of 100, and every database adapter rejects larger
row-page requests.

Application settings need an in-app editing surface that can grow beyond the
initial page-size setting. Changing the page size while a page is displayed
also needs defined pagination behavior so the transition does not repeat or
skip rows.

## Decision

`Ctrl+S` opens a settings modal. The initial modal contains one setting,
presented as the configured page size, and is structured as the application's
settings surface rather than as a one-off page-size prompt.

The configured page size is valid when it is a positive whole number. It has
no application-defined upper bound. The TUI validates the entered value and
does not persist invalid input.

A valid saved value is written to `~/.config/db-tui/config.json` and remains in
effect across application restarts.

Saving a new page size does not reload or otherwise replace the currently
displayed page. The next row-page request uses the new size. When moving
forward after changing the size, the next request begins after the final row
of the currently displayed page, rather than advancing by the newly configured
size from the old page's starting offset. This avoids duplicated or skipped
rows when the old and new sizes differ.

The modal's save, cancellation, error-feedback, and focus behavior remain to be
settled before this ADR can be accepted.

## Consequences

- Users can configure pagination without leaving db-tui or manually editing
  JSON.
- The configured page size becomes distinct from any internal safety bound
  retained for other result-producing operations.
- Relation paging can request more than 100 rows, so the database-neutral page
  request contract and every adapter's validation must support positive limits
  without the current upper bound.
- Paging must derive the next forward offset from the displayed page's actual
  row count, not solely from the current configured page size.
- Very large page sizes can increase database work, transfer volume, memory
  use, and rendering cost; the application accepts that tradeoff rather than
  imposing an upper limit.
- Configuration-save failures must remain visible to the user and must not be
  reported as successful settings changes.

## Alternatives considered

### Keep the hard maximum of 100

Rejected. Users must be able to enter and save any positive whole-number page
size, including values above 100.

### Clamp oversized values silently

Rejected. Silent clamping would make the saved value and observed page size
disagree with the user's explicit input.

### Reload the active relation immediately after saving

Rejected. The new page size takes effect on the next row-page request and does
not cause an unsolicited query.

### Advance by the new page size after a mid-page change

Rejected. If the displayed page was loaded using a different size, advancing
from its starting offset by the new size could repeat or skip rows. Forward
navigation continues immediately after the displayed page instead.
