# ADR 0012: Drag-select rendered text in the data grid

## Status

Proposed.

## Context

The data panel renders a title, status information, a column-header row, and
pageable result rows. Terminal-emulator text selection is not a dependable
interaction while db-tui uses the alternate screen and mouse tracking. The UI
therefore needs an application-owned selection interaction for copying result
text.

## Decision

Only the data grid is selectable: its column-header row and visible result-data
rows. The data-panel title, status text, panel border, navigator, query panel,
and overlays are excluded.

A primary-button drag creates a character-range selection within the rendered
cell where the drag starts. Moving into another row or column clamps the range
to the original cell. A wrapped cell remains one selectable cell across all of
its rendered lines. Releasing the button copies that selection to the system
clipboard.
The clipboard value is plain text containing every selected visible character,
including column padding and inter-column spaces, but never ANSI styling
sequences.

A primary-button click that does not produce a drag retains the existing row
selection behavior and does not write to the clipboard.

The release must be within the selectable grid. Releasing outside it cancels
the drag and leaves the clipboard unchanged. Navigating rows or columns and
loading a new page clears any existing text selection.

## Consequences

- Copying is scoped to tabular result text and cannot accidentally include
  surrounding application chrome.
- The feature must map terminal cells to the grid's rendered text and maintain
  its own highlight and drag state.
- A drag that ends outside the grid is deliberately cancelled rather than
  guessing an edge position.

## Alternatives considered

### Rely on the terminal emulator's selection

Rejected. Alternate-screen and mouse-tracking behavior varies among terminals
and multiplexers, so it cannot provide a portable db-tui interaction.

### Copy whole rows or cells only

Rejected. The required interaction is a character-range selection, equivalent
to terminal-style dragging, and includes the column-header row.
