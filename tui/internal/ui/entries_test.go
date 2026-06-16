package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
