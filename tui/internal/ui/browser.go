package ui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/phekno/inkwell/tui/internal/api"
)

// browseRow is a single line in the folder browser: either a child folder or
// an entry that lives directly in the current folder.
type browseRow struct {
	Folder bool
	Name   string        // folder segment, or entry title
	Path   string        // full child-folder path (folders only)
	Entry  api.EntryMeta // populated for entries only
}

// folderNameLess orders sibling folder names: purely numeric names (year
// folders like 2026) sort descending so the newest year is on top, numeric
// names come before words, and words sort ascending. Mirrors the web client's
// compareFolderNames in web/src/lib/tree.ts.
func folderNameLess(a, b string) bool {
	an, aNum := digitsValue(a)
	bn, bNum := digitsValue(b)
	switch {
	case aNum && bNum:
		return an > bn // years newest-first
	case aNum != bNum:
		return aNum // numeric before words
	default:
		return a < b
	}
}

// digitsValue parses a purely-digit string ("2026", not "+5" or "v2").
func digitsValue(s string) (int, bool) {
	if s == "" || strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' }) {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// browse computes the rows shown at folder path current: immediate child
// folders (sorted, de-duplicated) first, then entries that live directly in
// current, in the order given.
func browse(entries []api.EntryMeta, current string) []browseRow {
	seen := map[string]bool{}
	var folders []string
	var items []browseRow

	for _, e := range entries {
		rel := relSegments(e.Folder, current)
		if len(rel) == 0 {
			if e.Folder == current {
				items = append(items, browseRow{Name: e.Title, Entry: e})
			}
			continue
		}
		if name := rel[0]; !seen[name] {
			seen[name] = true
			folders = append(folders, name)
		}
	}
	sort.Slice(folders, func(i, j int) bool { return folderNameLess(folders[i], folders[j]) })

	rows := make([]browseRow, 0, len(folders)+len(items))
	for _, name := range folders {
		path := name
		if current != "" {
			path = current + "/" + name
		}
		rows = append(rows, browseRow{Folder: true, Name: name, Path: path})
	}
	return append(rows, items...)
}

// wrapText word-wraps each line of s to width, preserving blank lines so
// paragraphs survive. The read-only viewport doesn't soft-wrap, so long Notion
// paragraphs would otherwise render truncated at the right edge.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if len(cur)+1+len(w) > width {
				out = append(out, cur)
				cur = w
			} else {
				cur += " " + w
			}
		}
		out = append(out, cur)
	}
	return strings.Join(out, "\n")
}

// windowSlice returns the [start, end) range of rows to render so that a list
// taller than the viewport scrolls to keep the cursor visible. It's stateless:
// the cursor sits at the bottom edge once you scroll past the first page.
func windowSlice(rowCount, cursor, height int) (int, int) {
	if height <= 0 || rowCount <= height {
		return 0, rowCount
	}
	start := 0
	if cursor >= height {
		start = cursor - height + 1
	}
	end := start + height
	if end > rowCount {
		end = rowCount
		start = end - height
	}
	return start, end
}

// parentPath returns the folder one level up. Root ("") has no parent.
func parentPath(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i]
	}
	return ""
}

// relSegments returns folder's path relative to current, or nil if folder is
// not at or below current.
func relSegments(folder, current string) []string {
	if current == "" {
		if folder == "" {
			return nil
		}
		return strings.Split(folder, "/")
	}
	if folder == current {
		return nil
	}
	if rest, ok := strings.CutPrefix(folder, current+"/"); ok {
		return strings.Split(rest, "/")
	}
	return nil
}
