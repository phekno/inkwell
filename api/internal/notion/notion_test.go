package notion

import (
	"testing"
	"time"
)

func TestParseDocWithProperties(t *testing.T) {
	md := "# April 11th, 2022\n" +
		"\n" +
		"Created: April 11, 2022 10:01 AM\n" +
		"Tags: Personal\n" +
		"Updated: April 12, 2022 9:00 AM\n" +
		"\n" +
		"Not a bad weekend.\n" +
		"\n" +
		"Second paragraph.\n"

	doc, err := ParseDoc(md)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "April 11th, 2022" {
		t.Errorf("title = %q", doc.Title)
	}
	wantCreated := time.Date(2022, 4, 11, 10, 1, 0, 0, time.Local)
	if !doc.Created.Equal(wantCreated) {
		t.Errorf("created = %v, want %v", doc.Created, wantCreated)
	}
	if doc.Updated.IsZero() || !doc.Updated.Equal(time.Date(2022, 4, 12, 9, 0, 0, 0, time.Local)) {
		t.Errorf("updated = %v", doc.Updated)
	}
	if doc.Body != "Not a bad weekend.\n\nSecond paragraph." {
		t.Errorf("body = %q", doc.Body)
	}
}

func TestParseDocDropsTagsKeepsBodyColons(t *testing.T) {
	// A body line containing a colon must survive (it's after the blank line).
	md := "# Note\n" +
		"Created: January 2, 2024 3:04 PM\n" +
		"Tags: Work, Personal\n" +
		"\n" +
		"Ratio: 2:1 is fine.\n"

	doc, err := ParseDoc(md)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Body != "Ratio: 2:1 is fine." {
		t.Errorf("body = %q", doc.Body)
	}
}

func TestParseDocNoProperties(t *testing.T) {
	md := "# Just a title\n\nStraight into the body.\n"
	doc, err := ParseDoc(md)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "Just a title" {
		t.Errorf("title = %q", doc.Title)
	}
	if !doc.Created.IsZero() {
		t.Errorf("expected zero Created, got %v", doc.Created)
	}
	if doc.Body != "Straight into the body." {
		t.Errorf("body = %q", doc.Body)
	}
}

func TestParseDocUpdatedFallsBackToCreatedIsCallerJob(t *testing.T) {
	// ParseDoc reports zero Updated when absent; the fallback is the caller's.
	md := "# T\nCreated: March 3, 2025 8:00 AM\n\nBody.\n"
	doc, err := ParseDoc(md)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Updated.IsZero() {
		t.Errorf("expected zero Updated, got %v", doc.Updated)
	}
}

func TestFolderFromPath(t *testing.T) {
	root := "/exp"
	cases := map[string]string{
		"/exp/Journal/April 11th, 2022 abc.md":        "Journal",
		"/exp/Work/OE and Debt/Update - Sept 2023.md": "Work/OE and Debt",
		"/exp/Home 97ee.md":                           "",
	}
	for full, want := range cases {
		if got := FolderFromPath(root, full); got != want {
			t.Errorf("FolderFromPath(%q) = %q, want %q", full, got, want)
		}
	}
}

func TestDateFromTitle(t *testing.T) {
	cases := []struct {
		title string
		ok    bool
		want  time.Time
	}{
		{"Update - September 2023", true, time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local)},
		{"J2 - 1st check - February 17th, 2023", true, time.Date(2023, 2, 17, 0, 0, 0, 0, time.Local)},
		{"Beginning - February 2023", true, time.Date(2023, 2, 1, 0, 0, 0, 0, time.Local)},
		{"Work", false, time.Time{}},
		{"Joshua Fechner", false, time.Time{}},
	}
	for _, c := range cases {
		got, ok := DateFromTitle(c.title)
		if ok != c.ok {
			t.Errorf("DateFromTitle(%q) ok = %v, want %v", c.title, ok, c.ok)
			continue
		}
		if ok && !got.Equal(c.want) {
			t.Errorf("DateFromTitle(%q) = %v, want %v", c.title, got, c.want)
		}
	}
}

func TestPlanInlineCreated(t *testing.T) {
	content := "# April 11th, 2022\nCreated: April 11, 2022 10:01 AM\nTags: Personal\n\nBody here.\n"
	p, ok, reason := Plan("/exp", "/exp/Journal/April 11th, 2022 abc.md", content)
	if !ok {
		t.Fatalf("expected import, got skip: %s", reason)
	}
	if p.Folder != "Journal" || p.Title != "April 11th, 2022" {
		t.Errorf("plan = %+v", p)
	}
	if p.DateSource != "created" {
		t.Errorf("date source = %q", p.DateSource)
	}
	if !p.Created.Equal(time.Date(2022, 4, 11, 10, 1, 0, 0, time.Local)) {
		t.Errorf("created = %v", p.Created)
	}
	// Updated absent -> falls back to Created.
	if !p.Updated.Equal(p.Created) {
		t.Errorf("updated = %v, want = created", p.Updated)
	}
	if p.ID == "" || p.Body != "Body here." {
		t.Errorf("plan = %+v", p)
	}
}

func TestPlanDateFromTitle(t *testing.T) {
	content := "# Update - September 2023\n\nNo created property here.\n"
	p, ok, _ := Plan("/exp", "/exp/Work/OE and Debt/Update - September 2023 x.md", content)
	if !ok {
		t.Fatal("expected import via title date")
	}
	if p.DateSource != "title" {
		t.Errorf("date source = %q", p.DateSource)
	}
	if !p.Created.Equal(time.Date(2023, 9, 1, 0, 0, 0, 0, time.Local)) {
		t.Errorf("created = %v", p.Created)
	}
	if p.Folder != "Work/OE and Debt" {
		t.Errorf("folder = %q", p.Folder)
	}
}

func TestPlanSkipsUndated(t *testing.T) {
	content := "# Work\n\n[OE and Debt](Work/OE%20and%20Debt.md)\n"
	_, ok, reason := Plan("/exp", "/exp/Work fe12.md", content)
	if ok {
		t.Fatal("expected skip for undated structural page")
	}
	if reason == "" {
		t.Error("expected a skip reason")
	}
}

func TestPlanForcedUsesFallbackWhenUndated(t *testing.T) {
	fallback := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	content := "# Salary History\n\n- Started at $46k\n"
	p, err := PlanForced("/exp", "/exp/Salary History abc.md", content, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if p.DateSource != "fallback" || !p.Created.Equal(fallback) {
		t.Errorf("plan = %+v", p)
	}
	if p.Folder != "" || p.Title != "Salary History" || p.Body != "- Started at $46k" {
		t.Errorf("plan = %+v", p)
	}
}

func TestPlanForcedPrefersRealDate(t *testing.T) {
	fallback := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	content := "# T\nCreated: June 5, 2024 1:00 PM\n\nBody.\n"
	p, err := PlanForced("/exp", "/exp/T.md", content, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if p.DateSource != "created" {
		t.Errorf("should prefer the inline Created date, got %q", p.DateSource)
	}
}

func TestDeterministicIDStableAndChronological(t *testing.T) {
	t1 := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	a := DeterministicID(t1, "Journal", "Entry A")
	again := DeterministicID(t1, "Journal", "Entry A")
	if a != again {
		t.Fatalf("not deterministic: %q vs %q", a, again)
	}

	b := DeterministicID(t1, "Journal", "Entry B")
	if a == b {
		t.Fatalf("different titles should yield different ids")
	}

	later := DeterministicID(t2, "Journal", "Entry A")
	// ULIDs are lexicographically sortable by timestamp; later time sorts higher.
	if later <= a {
		t.Fatalf("expected chronological ordering: %q should sort after %q", later, a)
	}
}
