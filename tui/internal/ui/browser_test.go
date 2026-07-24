package ui

import (
	"testing"

	"github.com/phekno/inkwell/tui/internal/api"
)

func fixture() []api.EntryMeta {
	return []api.EntryMeta{
		{ID: "1", Title: "root note", Folder: ""},
		{ID: "2", Title: "standup", Folder: "work"},
		{ID: "3", Title: "1:1", Folder: "work/meetings"},
		{ID: "4", Title: "trip", Folder: "personal/trips"},
	}
}

func TestBrowseRoot(t *testing.T) {
	rows := browse(fixture(), "")
	// folders first (sorted), then entries at this level
	if len(rows) != 3 {
		t.Fatalf("got %d rows: %+v", len(rows), rows)
	}
	if !rows[0].Folder || rows[0].Name != "personal" || rows[0].Path != "personal" {
		t.Fatalf("row0 = %+v", rows[0])
	}
	if !rows[1].Folder || rows[1].Name != "work" {
		t.Fatalf("row1 = %+v", rows[1])
	}
	if rows[2].Folder || rows[2].Entry.Title != "root note" {
		t.Fatalf("row2 = %+v", rows[2])
	}
}

func TestBrowseNested(t *testing.T) {
	rows := browse(fixture(), "work")
	// child folder "meetings" (full path work/meetings) + entry "standup"
	if len(rows) != 2 {
		t.Fatalf("got %d rows: %+v", len(rows), rows)
	}
	if !rows[0].Folder || rows[0].Name != "meetings" || rows[0].Path != "work/meetings" {
		t.Fatalf("row0 = %+v", rows[0])
	}
	if rows[1].Folder || rows[1].Entry.Title != "standup" {
		t.Fatalf("row1 = %+v", rows[1])
	}
}

func TestBrowseDeepFolderHasNoDuplicateChildren(t *testing.T) {
	entries := []api.EntryMeta{
		{ID: "a", Title: "x", Folder: "work/meetings"},
		{ID: "b", Title: "y", Folder: "work/meetings"},
		{ID: "c", Title: "z", Folder: "work/journal"},
	}
	rows := browse(entries, "work")
	// two distinct child folders, no entries directly in "work"
	if len(rows) != 2 || rows[0].Name != "journal" || rows[1].Name != "meetings" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestBrowseNumericFoldersSortDescending(t *testing.T) {
	entries := []api.EntryMeta{
		{ID: "a", Title: "x", Folder: "Journal/2014"},
		{ID: "b", Title: "y", Folder: "Journal/2026"},
		{ID: "c", Title: "z", Folder: "Journal/2019"},
	}
	rows := browse(entries, "Journal")
	// year folders newest-first
	if len(rows) != 3 || rows[0].Name != "2026" || rows[1].Name != "2019" || rows[2].Name != "2014" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestBrowseNumericSortIsNumericNotLexical(t *testing.T) {
	entries := []api.EntryMeta{
		{ID: "a", Title: "x", Folder: "9"},
		{ID: "b", Title: "y", Folder: "10"},
		{ID: "c", Title: "z", Folder: "100"},
	}
	rows := browse(entries, "")
	if len(rows) != 3 || rows[0].Name != "100" || rows[1].Name != "10" || rows[2].Name != "9" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestBrowseNumericFoldersBeforeWords(t *testing.T) {
	entries := []api.EntryMeta{
		{ID: "a", Title: "w", Folder: "Archive"},
		{ID: "b", Title: "x", Folder: "2020"},
		{ID: "c", Title: "y", Folder: "2024"},
		{ID: "d", Title: "z", Folder: "Work"},
	}
	rows := browse(entries, "")
	want := []string{"2024", "2020", "Archive", "Work"}
	for i, name := range want {
		if rows[i].Name != name {
			t.Fatalf("row %d = %q, want %q (rows %+v)", i, rows[i].Name, name, rows)
		}
	}
}

func TestWindowSlice(t *testing.T) {
	cases := []struct {
		name                     string
		rowCount, cursor, height int
		wantStart, wantEnd       int
	}{
		{"all fit", 10, 3, 20, 0, 10},
		{"top of long list", 100, 0, 10, 0, 10},
		{"cursor still on first page", 100, 5, 10, 0, 10},
		{"cursor scrolled past fold", 100, 50, 10, 41, 51},
		{"cursor at very end", 100, 99, 10, 90, 100},
		{"empty", 0, 0, 10, 0, 0},
	}
	for _, c := range cases {
		start, end := windowSlice(c.rowCount, c.cursor, c.height)
		if start != c.wantStart || end != c.wantEnd {
			t.Errorf("%s: windowSlice(%d,%d,%d) = (%d,%d), want (%d,%d)",
				c.name, c.rowCount, c.cursor, c.height, start, end, c.wantStart, c.wantEnd)
		}
		// the cursor must always land inside the returned window
		if c.rowCount > 0 && (c.cursor < start || c.cursor >= end) {
			t.Errorf("%s: cursor %d not visible in [%d,%d)", c.name, c.cursor, start, end)
		}
	}
}

func TestWrapText(t *testing.T) {
	if got := wrapText("the quick brown fox", 9); got != "the quick\nbrown fox" {
		t.Errorf("wrap = %q", got)
	}
	// paragraph breaks (blank lines) are preserved
	if got := wrapText("a\n\nb", 10); got != "a\n\nb" {
		t.Errorf("blank lines not preserved: %q", got)
	}
	// width <= 0 is a no-op (don't garble before the viewport is sized)
	if got := wrapText("anything at all", 0); got != "anything at all" {
		t.Errorf("zero width should be a no-op: %q", got)
	}
}

func TestParentPath(t *testing.T) {
	cases := map[string]string{
		"work/meetings": "work",
		"work":          "",
		"":              "",
	}
	for in, want := range cases {
		if got := parentPath(in); got != want {
			t.Errorf("parentPath(%q) = %q, want %q", in, got, want)
		}
	}
}
