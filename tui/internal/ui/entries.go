package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	"github.com/phekno/inkwell/tui/internal/api"
)

type entriesMode int

const (
	modeList entriesMode = iota
	modeView
	modeCompose
	modeMove
)

type entriesModel struct {
	api    *api.Client
	mode   entriesMode
	width  int
	height int

	list   []api.EntryMeta
	path   string // current folder in the browser ("" = root)
	cursor int
	loaded bool
	err    string

	current *api.Entry
	body    viewport.Model

	editingID     string // "" means composing a new entry
	saving        bool   // a save is in flight; blocks duplicate submits
	loading       bool   // an entry fetch is in flight
	confirmDelete string // entry id awaiting delete confirmation ("" = none)
	titleInput    textinput.Model
	bodyInput     textarea.Model

	moveID    string
	moveInput textinput.Model
}

func newEntries(c *api.Client) entriesModel {
	ti := textinput.New()
	ti.Placeholder = "Title"
	ti.Prompt = "  "

	ta := textarea.New()
	ta.Placeholder = "Write…"
	ta.ShowLineNumbers = false

	mi := textinput.New()
	mi.Placeholder = "folder/path"
	mi.Prompt = "  "

	vp := viewport.New(80, 20)

	return entriesModel{api: c, titleInput: ti, bodyInput: ta, moveInput: mi, body: vp}
}

func (m entriesModel) Init() tea.Cmd { return m.loadList() }

func (m entriesModel) loadList() tea.Cmd {
	c := m.api
	return func() tea.Msg {
		list, err := c.ListEntries()
		if err != nil {
			return entriesErrMsg{err: err}
		}
		return entriesLoadedMsg{entries: list}
	}
}

// rows is the current browser view: child folders + entries at m.path.
func (m entriesModel) rows() []browseRow { return browse(m.list, m.path) }

func (m entriesModel) Update(msg tea.Msg) (entriesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		cw := m.contentWidth()
		m.body.Width = cw
		m.body.Height = max(5, msg.Height-6)
		// The body sits inside a bordered, padded box, so subtract its frame
		// (border + padding) — otherwise lipgloss re-wraps the textarea lines.
		m.bodyInput.SetWidth(max(10, cw-editorFrameWidth))
		m.bodyInput.SetHeight(max(3, msg.Height-composeChromeHeight))
		// Single-line inputs scroll horizontally; cap their width so a long
		// value can't wrap and push the layout past the terminal height.
		m.titleInput.Width = max(10, cw-3)
		m.moveInput.Width = max(10, cw-3)
		if m.current != nil {
			m.body.SetContent(renderMarkdown(m.current.Body, cw))
		}

	case entriesLoadedMsg:
		m.list = msg.entries
		m.loaded = true
		m.clampCursor()
	case entriesErrMsg:
		m.err = msg.err.Error()
		m.saving = false
		m.loading = false
	case entryOpenedMsg:
		m.current = msg.entry
		m.loading = false
		m.body.SetContent(renderMarkdown(msg.entry.Body, m.contentWidth()))
		m.body.GotoTop()
		m.mode = modeView
	case entryEditLoadedMsg:
		m.current = msg.entry
		m.loading = false
		m.editingID = msg.entry.ID
		m.titleInput.SetValue(msg.entry.Title)
		m.bodyInput.SetValue(msg.entry.Body)
		m.mode = modeCompose
		m.titleInput.Focus()
		return m, textinput.Blink
	case entryCreatedMsg:
		m.list = append([]api.EntryMeta{*msg.meta}, m.list...)
		m.resetForm()
		m.mode = modeList
		m.clampCursor()
	case entryUpdatedMsg:
		for i := range m.list {
			if m.list[i].ID == msg.meta.ID {
				m.list[i].Title = msg.meta.Title
				m.list[i].UpdatedAt = msg.meta.UpdatedAt
			}
		}
		if m.current != nil && m.current.ID == msg.meta.ID {
			m.current.Title = msg.meta.Title
		}
		m.resetForm()
		m.mode = modeList
	case entryMovedMsg:
		for i := range m.list {
			if m.list[i].ID == msg.id {
				m.list[i].Folder = msg.folder
			}
		}
		m.mode = modeList
		m.clampCursor()
	case entryDeletedMsg:
		out := m.list[:0]
		for _, e := range m.list {
			if e.ID != msg.id {
				out = append(out, e)
			}
		}
		m.list = out
		m.current = nil
		m.mode = modeList
		m.clampCursor()
	}

	switch m.mode {
	case modeList:
		return m.updateBrowser(msg)
	case modeView:
		return m.updateView(msg)
	case modeCompose:
		return m.updateCompose(msg)
	case modeMove:
		return m.updateMove(msg)
	}
	return m, nil
}

func (m *entriesModel) clampCursor() {
	if n := len(m.rows()); m.cursor >= n {
		m.cursor = max(0, n-1)
	}
}

func (m *entriesModel) resetForm() {
	m.editingID = ""
	m.saving = false
	m.titleInput.SetValue("")
	m.bodyInput.SetValue("")
	m.titleInput.Blur()
	m.bodyInput.Blur()
}

func (m entriesModel) updateBrowser(msg tea.Msg) (entriesModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	rows := m.rows()

	// When a delete is pending, the next key either confirms or cancels it.
	if m.confirmDelete != "" {
		id := m.confirmDelete
		m.confirmDelete = ""
		if s := key.String(); s == "y" || s == "Y" {
			c := m.api
			return m, func() tea.Msg {
				if err := c.DeleteEntry(id); err != nil {
					return entriesErrMsg{err: err}
				}
				return entryDeletedMsg{id: id}
			}
		}
		return m, nil // any other key cancels
	}

	switch key.String() {
	case "j", "down":
		if m.cursor < len(rows)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "pgdown", "ctrl+d", "ctrl+f":
		if n := len(rows); n > 0 {
			m.cursor = min(n-1, m.cursor+m.listPageSize())
		}
	case "pgup", "ctrl+u", "ctrl+b":
		m.cursor = max(0, m.cursor-m.listPageSize())
	case "end", "G":
		if n := len(rows); n > 0 {
			m.cursor = n - 1
		}
	case "home", "g":
		m.cursor = 0
	case "backspace", "left", "h":
		if m.path != "" {
			m.path = parentPath(m.path)
			m.cursor = 0
		}
	case "enter", "right", "l":
		if len(rows) == 0 {
			return m, nil
		}
		row := rows[m.cursor]
		if row.Folder {
			m.path = row.Path
			m.cursor = 0
			return m, nil
		}
		return m.startOpen(row.Entry.ID, false)
	case "e":
		if row, ok := m.entryAtCursor(rows); ok {
			return m.startOpen(row.Entry.ID, true)
		}
	case "m":
		if row, ok := m.entryAtCursor(rows); ok {
			m.moveID = row.Entry.ID
			m.moveInput.SetValue(row.Entry.Folder)
			m.moveInput.Focus()
			m.mode = modeMove
			return m, textinput.Blink
		}
	case "d":
		if row, ok := m.entryAtCursor(rows); ok {
			m.confirmDelete = row.Entry.ID
		}
	case "n":
		m.resetForm()
		m.mode = modeCompose
		m.titleInput.Focus()
		return m, textinput.Blink
	case "r":
		return m, m.loadList()
	}
	return m, nil
}

func (m entriesModel) entryAtCursor(rows []browseRow) (browseRow, bool) {
	if m.cursor < 0 || m.cursor >= len(rows) || rows[m.cursor].Folder {
		return browseRow{}, false
	}
	return rows[m.cursor], true
}

// startOpen marks a fetch as in flight (so the UI can show "loading…") and
// returns the fetch command.
func (m entriesModel) startOpen(id string, forEdit bool) (entriesModel, tea.Cmd) {
	m.loading = true
	return m, m.openEntry(id, forEdit)
}

// openEntry fetches an entry; forEdit routes it to the edit form, otherwise the
// read-only view.
func (m entriesModel) openEntry(id string, forEdit bool) tea.Cmd {
	c := m.api
	return func() tea.Msg {
		e, err := c.GetEntry(id)
		if err != nil {
			return entriesErrMsg{err: err}
		}
		if forEdit {
			return entryEditLoadedMsg{entry: e}
		}
		return entryOpenedMsg{entry: e}
	}
}

func (m entriesModel) updateView(msg tea.Msg) (entriesModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "q":
			m.mode = modeList
			return m, nil
		case "e":
			if m.current != nil {
				return m.startOpen(m.current.ID, true)
			}
		case "m":
			if m.current != nil {
				m.moveID = m.current.ID
				m.moveInput.SetValue(m.current.Folder)
				m.moveInput.Focus()
				m.mode = modeMove
				return m, textinput.Blink
			}
		case "d":
			if m.current == nil {
				return m, nil
			}
			id := m.current.ID
			c := m.api
			return m, func() tea.Msg {
				if err := c.DeleteEntry(id); err != nil {
					return entriesErrMsg{err: err}
				}
				return entryDeletedMsg{id: id}
			}
		}
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(msg)
	return m, cmd
}

func (m entriesModel) updateCompose(msg tea.Msg) (entriesModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.resetForm()
			m.mode = modeList
			return m, nil
		case "ctrl+s":
			if m.saving {
				return m, nil // a save is already in flight
			}
			title := strings.TrimSpace(m.titleInput.Value())
			body := m.bodyInput.Value()
			if title == "" {
				m.err = "title required"
				return m, nil
			}
			m.saving = true
			m.err = ""
			c := m.api
			if id := m.editingID; id != "" {
				return m, func() tea.Msg {
					meta, err := c.UpdateEntry(id, title, body)
					if err != nil {
						return entriesErrMsg{err: err}
					}
					return entryUpdatedMsg{meta: meta}
				}
			}
			folder := m.path
			return m, func() tea.Msg {
				meta, err := c.CreateEntry(title, body, folder)
				if err != nil {
					return entriesErrMsg{err: err}
				}
				return entryCreatedMsg{meta: meta}
			}
		case "tab":
			if m.titleInput.Focused() {
				m.titleInput.Blur()
				m.bodyInput.Focus()
			} else {
				m.bodyInput.Blur()
				m.titleInput.Focus()
			}
			return m, nil
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	if m.titleInput.Focused() {
		m.titleInput, cmd = m.titleInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.bodyInput.Focused() {
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m entriesModel) updateMove(msg tea.Msg) (entriesModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.moveInput.Blur()
			m.mode = modeList
			return m, nil
		case "enter":
			id := m.moveID
			dest := m.moveInput.Value()
			c := m.api
			m.moveInput.Blur()
			return m, func() tea.Msg {
				meta, err := c.MoveEntry(id, dest)
				if err != nil {
					return entriesErrMsg{err: err}
				}
				return entryMovedMsg{id: id, folder: meta.Folder}
			}
		}
	}
	var cmd tea.Cmd
	m.moveInput, cmd = m.moveInput.Update(msg)
	return m, cmd
}

const (
	listPaneWidth = 34
	// editorFrameWidth is the columns the editor box (border + padding) eats.
	editorFrameWidth = 4
	// composeChromeHeight is the rows the compose view spends on everything but
	// the body textarea: heading, the title label+field, the body label, the
	// box border, the hint line, plus a little slack for an error line.
	composeChromeHeight = 12
)

// rightWidth is the rendered width of the detail/edit pane (incl. its padding).
func (m entriesModel) rightWidth() int {
	return max(m.width-listPaneWidth-3, 20) // account for border + padding
}

// contentWidth is the usable width inside the right pane's 1-col padding.
func (m entriesModel) contentWidth() int {
	return max(m.rightWidth()-2, 18)
}

// listPageSize is how many entry rows fit in the browser, used both for the
// scroll window and for page-at-a-time jumps. The remaining rows go to the
// header and the (up to three-line) hint footer.
func (m entriesModel) listPageSize() int {
	return max(1, m.height-7)
}

// glamourStyle is the markdown theme, detected once at startup. We must NOT use
// glamour's WithAutoStyle() during the event loop: it queries the terminal for
// its background colour, but Bubble Tea owns stdin, so the reply never arrives
// and the render blocks for ~15s on every entry open.
var glamourStyle = "dark"

// renderMarkdown styles Markdown for the terminal via glamour, falling back to
// plain word-wrapped text if the renderer is unavailable.
func renderMarkdown(src string, width int) string {
	if width <= 0 {
		return src
	}
	if r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(glamourStyle), glamour.WithWordWrap(width)); err == nil {
		if out, rerr := r.Render(src); rerr == nil {
			return strings.TrimRight(out, "\n")
		}
	}
	return wrapText(src, width)
}

func (m entriesModel) View() string {
	if !m.loaded {
		status := hintStyle.Render("loading…")
		if m.err != "" {
			status = errStyle.Render(m.err)
		}
		return titleStyle.Render("inkwell") + "\n\n" + status
	}

	h := max(1, m.height)
	left := lipgloss.NewStyle().
		Width(listPaneWidth).Height(h).Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#d0d0d0", Dark: "#333333"}).
		Render(m.listPane())
	right := lipgloss.NewStyle().
		Width(m.rightWidth()).Height(h).Padding(0, 1).
		Render(m.rightPane())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// listPane renders the always-visible folder browser on the left.
func (m entriesModel) listPane() string {
	selStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0a0a0a"}).
		Background(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f5f5f5"})

	header := "inkwell"
	if m.path != "" {
		header = m.path + "/"
	}
	inner := listPaneWidth - 2

	var b strings.Builder
	b.WriteString(titleStyle.Render(truncate.StringWithTail(header, uint(inner), "…")) + "\n\n")

	rows := m.rows()
	if len(rows) == 0 {
		b.WriteString(hintStyle.Render("empty · n to compose") + "\n")
	} else {
		start, end := windowSlice(len(rows), m.cursor, m.listPageSize())
		if start > 0 {
			b.WriteString(hintStyle.Render(fmt.Sprintf("↑ %d more", start)) + "\n")
		}
		for i := start; i < end; i++ {
			row := rows[i]
			label := "📝 " + row.Entry.Title
			if row.Folder {
				label = "📁 " + row.Name + "/"
			}
			label = truncate.StringWithTail(label, uint(inner-2), "…")
			if i == m.cursor {
				b.WriteString(selStyle.Width(inner).Render("> "+label) + "\n")
			} else {
				b.WriteString("  " + label + "\n")
			}
		}
		if end < len(rows) {
			b.WriteString(hintStyle.Render(fmt.Sprintf("↓ %d more", len(rows)-end)) + "\n")
		}
	}

	hint := "enter open · n new · e edit\nm move · d del · r refresh · q quit\n⇞/⇟ page · g/G top/end"
	if m.path != "" {
		hint = "enter open · ⌫ up · n new\ne edit · m move · d del · q quit\n⇞/⇟ page · g/G top/end"
	}
	if m.confirmDelete != "" {
		hint = "delete? y = yes · other = cancel"
	}
	b.WriteString("\n" + hintStyle.Render(hint))
	if m.err != "" {
		b.WriteString("\n\n" + errStyle.Render(m.err))
	}
	return b.String()
}

// rightPane renders the detail/edit/compose content beside the browser.
func (m entriesModel) rightPane() string {
	if m.loading {
		return titleStyle.Render("⏳ Loading…")
	}
	switch m.mode {
	case modeCompose:
		return m.composeContent()
	case modeMove:
		return m.moveContent()
	default: // modeView, or modeList showing the last-opened entry as a preview
		if m.current == nil {
			return hintStyle.Render("Select an entry (enter), or press n for a new one.")
		}
		var b strings.Builder
		b.WriteString(titleStyle.Render(m.current.Title) + "\n")
		b.WriteString(labelStyle.Render(m.current.CreatedAt.Local().Format("2006-01-02 15:04")) + "\n\n")
		b.WriteString(m.body.View())
		if m.mode == modeView {
			b.WriteString("\n\n" + hintStyle.Render("esc list · e edit · m move · d delete · ↑/↓ scroll"))
		}
		return b.String()
	}
}

func (m entriesModel) composeContent() string {
	heading := "new entry"
	if m.editingID != "" {
		heading = "edit entry"
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(heading) + "\n\n")
	b.WriteString(labelStyle.Render("title") + "\n" + m.titleInput.View() + "\n\n")
	b.WriteString(labelStyle.Render("body") + "\n")
	// lipgloss adds the border outside Width(), so size the box to contentWidth-2
	// (border) — its inner padding then leaves the textarea's contentWidth-4.
	b.WriteString(editorBoxStyle.Width(m.contentWidth()-2).Render(m.bodyInput.View()) + "\n\n")
	if m.saving {
		b.WriteString(titleStyle.Render("⏳ Saving…"))
	} else {
		b.WriteString(hintStyle.Render("ctrl+s save · tab switch field · esc cancel"))
	}
	if m.err != "" {
		b.WriteString("\n\n" + errStyle.Render(m.err))
	}
	return b.String()
}

func (m entriesModel) moveContent() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("move entry") + "\n\n")
	b.WriteString(labelStyle.Render("destination folder (blank = root)") + "\n" + m.moveInput.View() + "\n\n")
	b.WriteString(hintStyle.Render("enter move · esc cancel"))
	if m.err != "" {
		b.WriteString("\n\n" + errStyle.Render(m.err))
	}
	return b.String()
}
