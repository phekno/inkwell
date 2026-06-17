package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/phekno/inkwell/tui/internal/api"
)

// Regression: hitting ctrl+s twice before the save returned used to create two
// entries because nothing blocked the second submit.
func TestComposeGuardsAgainstDoubleSave(t *testing.T) {
	m := newEntries(api.New("http://example", "tok"))
	m.mode = modeCompose
	m.titleInput.SetValue("hello")

	ctrlS := tea.KeyMsg{Type: tea.KeyCtrlS}

	m, cmd1 := m.updateCompose(ctrlS)
	if cmd1 == nil {
		t.Fatal("first ctrl+s should dispatch a save")
	}
	if !m.saving {
		t.Fatal("first ctrl+s should mark the model as saving")
	}

	_, cmd2 := m.updateCompose(ctrlS)
	if cmd2 != nil {
		t.Fatal("second ctrl+s while still saving must not dispatch another save")
	}
}

func key(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// Regression (issues 2 & 3): the compose view used to size the textarea/title to
// the full right-pane width, ignoring the pane's 1-col padding. lipgloss then
// re-wrapped the over-wide lines, inserting phantom breaks and pushing the
// rendered block past the terminal height, which scrolled the whole screen up.
// The composed view must fit inside the terminal: no line wider than the screen,
// and no taller than the screen.
func TestComposeViewFitsTerminal(t *testing.T) {
	const w, h = 80, 24
	m := newEntries(api.New("http://example", "tok"))
	m.loaded = true
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})

	m.mode = modeCompose
	m.titleInput.SetValue(strings.Repeat("x", 200))          // longer than the pane
	m.bodyInput.SetValue(strings.Repeat("alpha beta ", 400)) // many wrapping lines

	out := m.View()

	if got := lipgloss.Height(out); got > h {
		t.Fatalf("compose view height %d exceeds terminal height %d (screen would scroll)", got, h)
	}
	if got := lipgloss.Width(out); got > w {
		t.Fatalf("compose view width %d exceeds terminal width %d", got, w)
	}
	for i, line := range strings.Split(out, "\n") {
		if lw := lipgloss.Width(line); lw > w {
			t.Fatalf("compose line %d width %d exceeds terminal width %d: %q", i, lw, w, line)
		}
	}
}

// Issue 4: the entries list should jump a page at a time.
func TestBrowserPageScroll(t *testing.T) {
	m := newEntries(api.New("http://example", "tok"))
	m.loaded = true
	m.height = 24
	for i := range 60 {
		m.list = append(m.list, api.EntryMeta{ID: string(rune('a' + i%26)), Title: "t", Folder: ""})
	}
	page := m.listPageSize()
	if page < 1 || page >= len(m.list) {
		t.Fatalf("unexpected page size %d for %d rows", page, len(m.list))
	}

	m, _ = m.updateBrowser(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.cursor != page {
		t.Fatalf("pgdown should advance one page (%d), got %d", page, m.cursor)
	}
	m, _ = m.updateBrowser(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.cursor != 0 {
		t.Fatalf("pgup should go back one page to 0, got %d", m.cursor)
	}
	m, _ = m.updateBrowser(tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != len(m.list)-1 {
		t.Fatalf("end should jump to last row %d, got %d", len(m.list)-1, m.cursor)
	}
	m, _ = m.updateBrowser(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Fatalf("home should jump to first row 0, got %d", m.cursor)
	}
}

func TestBrowserDeleteRequiresConfirm(t *testing.T) {
	m := newEntries(api.New("http://example", "tok"))
	m.loaded = true
	m.list = []api.EntryMeta{{ID: "e1", Title: "t", Folder: ""}}

	// 'd' arms the confirm but must NOT delete yet.
	m, cmd := m.updateBrowser(key("d"))
	if m.confirmDelete != "e1" {
		t.Fatalf("d should arm confirm for the selected entry, got %q", m.confirmDelete)
	}
	if cmd != nil {
		t.Fatal("d alone must not dispatch a delete")
	}

	// any non-confirm key cancels.
	cancelled, cmd := m.updateBrowser(key("n"))
	if cancelled.confirmDelete != "" || cmd != nil {
		t.Fatal("a non-confirm key should cancel without deleting")
	}

	// 'y' confirms and dispatches the delete.
	m.confirmDelete = "e1"
	_, cmd = m.updateBrowser(key("y"))
	if cmd == nil {
		t.Fatal("y should dispatch the delete")
	}
}

// Regression: an error during the initial list load used to be hidden behind a
// permanent "loading…" because the error only rendered after a successful load.
func TestBrowserShowsLoadErrorInsteadOfPerpetualLoading(t *testing.T) {
	m := entriesModel{err: "unauthorized"} // loaded=false, mode=modeList (zero values)

	out := m.View()

	if !strings.Contains(out, "unauthorized") {
		t.Fatalf("load error should be visible, got: %q", out)
	}
	if strings.Contains(out, "loading") {
		t.Fatalf("should not still say loading when there is an error: %q", out)
	}
}
