// Package notion parses a Notion "Markdown & CSV" export into inkwell entries.
package notion

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// Doc is a parsed Notion markdown page.
type Doc struct {
	Title   string
	Created time.Time // zero if the page had no Created: property
	Updated time.Time // zero if absent
	Body    string
}

const notionTimeLayout = "January 2, 2006 3:04 PM"

// Planned is a page resolved to an importable inkwell entry.
type Planned struct {
	Path       string
	Folder     string
	Title      string
	Created    time.Time
	Updated    time.Time
	Body       string
	ID         string
	DateSource string // "created" (inline property) or "title" (parsed fallback)
}

// Plan turns one markdown file into an import plan. ok is false (with a reason)
// when the page has no usable date — an inline Created: nor a date in its title
// — which is how structural/index stubs get skipped.
func Plan(root, fullPath, content string) (Planned, bool, string) {
	doc, err := ParseDoc(content)
	if err != nil {
		return Planned{}, false, err.Error()
	}
	if !doc.Created.IsZero() {
		return buildPlan(root, fullPath, doc, doc.Created, "created"), true, ""
	}
	if t, ok := DateFromTitle(doc.Title); ok {
		return buildPlan(root, fullPath, doc, t, "title"), true, ""
	}
	return Planned{}, false, "no Created: property and no date in title"
}

// PlanForced is like Plan but never skips: a page with no inline Created and no
// title date falls back to the given date. Use it for undated pages you want to
// import anyway. It errors only when the markdown has no title.
func PlanForced(root, fullPath, content string, fallback time.Time) (Planned, error) {
	doc, err := ParseDoc(content)
	if err != nil {
		return Planned{}, err
	}
	created, source := doc.Created, "created"
	if created.IsZero() {
		if t, ok := DateFromTitle(doc.Title); ok {
			created, source = t, "title"
		} else {
			created, source = fallback, "fallback"
		}
	}
	return buildPlan(root, fullPath, doc, created, source), nil
}

func buildPlan(root, fullPath string, doc Doc, created time.Time, source string) Planned {
	updated := doc.Updated
	if updated.IsZero() {
		updated = created
	}
	folder := FolderFromPath(root, fullPath)
	return Planned{
		Path:       fullPath,
		Folder:     folder,
		Title:      doc.Title,
		Created:    created,
		Updated:    updated,
		Body:       doc.Body,
		ID:         DeterministicID(created, folder, doc.Title),
		DateSource: source,
	}
}

var errNoTitle = errors.New("notion: no '# ' title line")

// propLine matches a Notion property line: a capitalized label, then ": value".
var propLine = regexp.MustCompile(`^[A-Z][A-Za-z0-9 ./-]*: `)

// ParseDoc extracts the title, Created/Updated timestamps, and body from a
// Notion-exported markdown page. The property block is the run of "Key: value"
// lines after the title (up to the first blank line); Created/Updated are kept,
// every other property is dropped, and the body is everything after.
func ParseDoc(md string) (Doc, error) {
	md = strings.TrimPrefix(md, "\ufeff") // strip BOM if present
	lines := strings.Split(md, "\n")

	titleIdx := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") {
			titleIdx = i
			break
		}
	}
	if titleIdx < 0 {
		return Doc{}, errNoTitle
	}

	var doc Doc
	doc.Title = strings.TrimSpace(strings.TrimPrefix(lines[titleIdx], "# "))

	rest := lines[titleIdx+1:]
	i := 0
	for i < len(rest) && strings.TrimSpace(rest[i]) == "" { // skip blanks after title
		i++
	}
	for i < len(rest) {
		line := rest[i]
		if strings.TrimSpace(line) == "" { // blank terminates the property block
			i++
			break
		}
		if !propLine.MatchString(line) { // non-property line: body starts here
			break
		}
		key, val, _ := strings.Cut(line, ": ")
		switch key {
		case "Created":
			if t, err := time.ParseInLocation(notionTimeLayout, strings.TrimSpace(val), time.Local); err == nil {
				doc.Created = t
			}
		case "Updated":
			if t, err := time.ParseInLocation(notionTimeLayout, strings.TrimSpace(val), time.Local); err == nil {
				doc.Updated = t
			}
		}
		i++
	}

	doc.Body = strings.TrimSpace(strings.Join(rest[i:], "\n"))
	return doc, nil
}

// FolderFromPath returns the entry folder for a file: its directory relative to
// the export root, with OS separators normalized to "/". Top-level files -> "".
func FolderFromPath(root, full string) string {
	rel := strings.TrimPrefix(filepath.ToSlash(full), strings.TrimRight(filepath.ToSlash(root), "/")+"/")
	dir := path.Dir(rel)
	if dir == "." || dir == "/" {
		return ""
	}
	return dir
}

var (
	reFullDate  = regexp.MustCompile(`([A-Z][a-z]+) (\d{1,2})(?:st|nd|rd|th)?, (\d{4})`)
	reMonthYear = regexp.MustCompile(`([A-Z][a-z]+) (\d{4})`)
)

// DateFromTitle best-effort parses a date from a page title, for pages that
// lack a Created: property. Handles "Month Dayth, YYYY" and "Month YYYY".
func DateFromTitle(title string) (time.Time, bool) {
	if m := reFullDate.FindStringSubmatch(title); m != nil {
		if t, err := time.ParseInLocation("January 2 2006", m[1]+" "+m[2]+" "+m[3], time.Local); err == nil {
			return t, true
		}
	}
	if m := reMonthYear.FindStringSubmatch(title); m != nil {
		if t, err := time.ParseInLocation("January 2006", m[1]+" "+m[2], time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// DeterministicID derives a ULID whose timestamp is created and whose entropy
// is a hash of folder+title: stable across runs and chronologically sortable.
func DeterministicID(created time.Time, folder, title string) string {
	h := sha256.Sum256([]byte(folder + "\x00" + title))
	id, err := ulid.New(ulid.Timestamp(created), bytes.NewReader(h[:]))
	if err != nil {
		panic(err) // entropy reader has 32 bytes; ULID needs 10
	}
	return id.String()
}
