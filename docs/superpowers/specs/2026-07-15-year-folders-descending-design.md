# Year Folders Sort Descending — Design

**Date:** 2026-07-15
**Scope:** web (`web/src/lib/tree.ts`) + TUI (`tui/internal/ui/browser.go`). No API changes.

## Problem

After the journal-by-year migration, `Journal/` contains `2014`–`2026`
subfolders. Both clients sort folders ascending alphabetically, so the oldest
year lands on top while entries inside sort newest-first — the user has to
scroll to reach the current year.

## Change (approved)

Sibling folders whose names are **purely numeric** sort **descending
numerically** (2026, 2025, … 2014). Word-named folders keep ascending
alphabetical order (Journal, Projects, Work). When numeric and word names mix
at one level, numeric names come first (matching the current
localeCompare-puts-digits-first behavior), then words ascending.

Applies to the web tree (`sortNode` in `web/src/lib/tree.ts`) and the TUI
browser rows (`browse` in `tui/internal/ui/browser.go`). Entry ordering is
unchanged (web: newest-first; TUI: as given).

## Testing

TDD in both codebases: `web/src/lib/tree.test.ts` and
`tui/internal/ui/browser_test.go` gain cases for numeric-descending,
word-ascending, and mixed levels.
