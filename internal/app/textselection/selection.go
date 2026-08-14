// Package textselection implements reusable character-range selection for
// rendered terminal text.
package textselection

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Point identifies a terminal cell within rendered text.
type Point struct {
	X int
	Y int
}

// Region is an inclusive rectangular selection boundary.
type Region struct {
	Left   int
	Right  int
	Top    int
	Bottom int
}

// Contains reports whether a point is inside the region.
func (r Region) Contains(point Point) bool {
	return point.X >= r.Left && point.X <= r.Right &&
		point.Y >= r.Top && point.Y <= r.Bottom
}

func (r Region) clamp(point Point) Point {
	return Point{
		X: min(max(point.X, r.Left), r.Right),
		Y: min(max(point.Y, r.Top), r.Bottom),
	}
}

// Selection tracks one drag-constrained character range.
type Selection struct {
	anchor   Point
	head     Point
	region   Region
	dragging bool
	active   bool
}

// Start begins a selection when point is inside region.
func (s *Selection) Start(point Point, region Region) bool {
	if !region.Contains(point) {
		return false
	}
	*s = Selection{anchor: point, head: point, region: region, dragging: true}
	return true
}

// Update moves the selection head, clamped to its starting region.
func (s *Selection) Update(point Point) bool {
	if !s.dragging {
		return false
	}
	s.head = s.region.clamp(point)
	s.active = s.head != s.anchor
	return true
}

// Release completes the drag and reports whether it selected a range.
func (s *Selection) Release(point Point) bool {
	if !s.dragging {
		return false
	}
	s.head = s.region.clamp(point)
	s.dragging = false
	s.active = s.head != s.anchor
	return s.active
}

// Clear removes the current drag and selected range.
func (s *Selection) Clear() {
	*s = Selection{}
}

// Active reports whether the selection contains a character range.
func (s Selection) Active() bool {
	return s.active
}

// Dragging reports whether a selection drag is in progress.
func (s Selection) Dragging() bool {
	return s.dragging
}

// Text returns the selected rendered characters without ANSI styling.
func (s Selection) Text(rendered string) string {
	if !s.active {
		return ""
	}
	lines := strings.Split(ansi.Strip(rendered), "\n")
	start, end := s.normalized()
	selected := make([]string, 0, end.Y-start.Y+1)
	for row := start.Y; row <= end.Y; row++ {
		left, right, ok := s.lineRange(row, start, end)
		if !ok || row >= len(lines) {
			continue
		}
		selected = append(selected, ansi.Cut(lines[row], left, right))
	}
	return strings.Join(selected, "\n")
}

// Render applies style to the selected rendered characters.
func (s Selection) Render(rendered string, style lipgloss.Style) string {
	if !s.active {
		return rendered
	}
	start, end := s.normalized()
	lines := strings.Split(rendered, "\n")
	for row, line := range lines {
		left, right, ok := s.lineRange(row, start, end)
		if !ok {
			continue
		}
		selected := ansi.Strip(ansi.Cut(line, left, right))
		lines[row] = ansi.Cut(line, 0, left) +
			style.Render(selected) +
			ansi.Cut(line, right, ansi.StringWidth(line))
	}
	return strings.Join(lines, "\n")
}

func (s Selection) normalized() (Point, Point) {
	start, end := s.anchor, s.head
	if end.Y < start.Y || (end.Y == start.Y && end.X < start.X) {
		start, end = end, start
	}
	return start, end
}

func (s Selection) lineRange(row int, start, end Point) (int, int, bool) {
	if row < start.Y || row > end.Y {
		return 0, 0, false
	}
	left, right := s.region.Left, s.region.Right+1
	if row == start.Y {
		left = start.X
	}
	if row == end.Y {
		right = end.X + 1
	}
	return left, right, left < right
}
