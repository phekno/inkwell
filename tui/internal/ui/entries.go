package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	editingID  string // "" means composing a new entry
	titleInput textinput.Model
	bodyInput  textarea.Model

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
		m.body.Width = max(40, msg.Width-30)
		m.body.Height = max(5, msg.Height-6)
		m.bodyInput.SetWidth(max(40, msg.Width-6))
		m.bodyInput.SetHeight(max(5, msg.Height-8))

	case entriesLoadedMsg:
		m.list = msg.entries
		m.loaded = true
		m.clampCursor()
	case entriesErrMsg:
		m.err = msg.err.Error()
	case entryOpenedMsg:
		m.current = msg.entry
		m.body.SetContent(msg.entry.Body)
		m.body.GotoTop()
		m.mode = modeView
	case entryEditLoadedMsg:
		m.current = msg.entry
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
	switch key.String() {
	case "j", "down":
		if m.cursor < len(rows)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
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
		return m, m.openEntry(row.Entry.ID, false)
	case "e":
		if row, ok := m.entryAtCursor(rows); ok {
			return m, m.openEntry(row.Entry.ID, true)
		}
	case "m":
		if row, ok := m.entryAtCursor(rows); ok {
			m.moveID = row.Entry.ID
			m.moveInput.SetValue(row.Entry.Folder)
			m.moveInput.Focus()
			m.mode = modeMove
			return m, textinput.Blink
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
				return m, m.openEntry(m.current.ID, true)
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
			title := strings.TrimSpace(m.titleInput.Value())
			body := m.bodyInput.Value()
			if title == "" {
				m.err = "title required"
				return m, nil
			}
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

func (m entriesModel) View() string {
	switch m.mode {
	case modeView:
		return m.renderView()
	case modeCompose:
		return m.renderCompose()
	case modeMove:
		return m.renderMove()
	}
	return m.renderBrowser()
}

func (m entriesModel) renderBrowser() string {
	if !m.loaded {
		return titleStyle.Render("inkwell") + "\n\n" + hintStyle.Render("loading…")
	}

	rowStyle := lipgloss.NewStyle().Padding(0, 2)
	selStyle := rowStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0a0a0a"}).
		Background(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f5f5f5"})

	header := "inkwell — entries"
	if m.path != "" {
		header = "inkwell — " + m.path + "/"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(header) + "\n\n")

	rows := m.rows()
	if len(rows) == 0 {
		b.WriteString(hintStyle.Render("empty · press n to compose · q to quit") + "\n")
	} else {
		for i, row := range rows {
			var line string
			if row.Folder {
				line = "📁 " + row.Name + "/"
			} else {
				line = "📝 " + row.Entry.Title + "  " +
					labelStyle.Render(row.Entry.CreatedAt.Local().Format("2006-01-02 15:04"))
			}
			if i == m.cursor {
				b.WriteString(selStyle.Render("> "+line) + "\n")
			} else {
				b.WriteString(rowStyle.Render("  "+line) + "\n")
			}
		}
		hint := "enter open · n new · e edit · m move · r refresh · q quit"
		if m.path != "" {
			hint = "enter open · ⌫ up · n new · e edit · m move · q quit"
		}
		b.WriteString("\n" + hintStyle.Render(hint))
	}
	if m.err != "" {
		b.WriteString("\n\n" + errStyle.Render(m.err))
	}
	return b.String()
}

func (m entriesModel) renderView() string {
	if m.current == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.current.Title) + "\n")
	b.WriteString(labelStyle.Render(m.current.CreatedAt.Local().Format("2006-01-02 15:04")) + "\n\n")
	b.WriteString(m.body.View() + "\n\n")
	b.WriteString(hintStyle.Render("esc back · e edit · m move · d delete · ↑/↓ scroll"))
	return b.String()
}

func (m entriesModel) renderCompose() string {
	heading := "new entry"
	if m.editingID != "" {
		heading = "edit entry"
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(heading) + "\n\n")
	b.WriteString(labelStyle.Render("title") + "\n" + m.titleInput.View() + "\n\n")
	b.WriteString(labelStyle.Render("body") + "\n" + m.bodyInput.View() + "\n\n")
	b.WriteString(hintStyle.Render("ctrl+s save · tab switch field · esc cancel"))
	if m.err != "" {
		b.WriteString("\n\n" + errStyle.Render(m.err))
	}
	return b.String()
}

func (m entriesModel) renderMove() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("move entry") + "\n\n")
	b.WriteString(labelStyle.Render("destination folder (blank = root)") + "\n" + m.moveInput.View() + "\n\n")
	b.WriteString(hintStyle.Render("enter move · esc cancel"))
	if m.err != "" {
		b.WriteString("\n\n" + errStyle.Render(m.err))
	}
	return b.String()
}
