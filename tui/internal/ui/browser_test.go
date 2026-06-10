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
