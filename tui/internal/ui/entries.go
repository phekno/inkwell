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
)

type entriesModel struct {
	api    *api.Client
	mode   entriesMode
	width  int
	height int

	list   []api.EntryMeta
	cursor int
	loaded bool
	err    string

	current *api.Entry
	body    viewport.Model

	titleInput textinput.Model
	bodyInput  textarea.Model
}

func newEntries(c *api.Client) entriesModel {
	ti := textinput.New()
	ti.Placeholder = "Title"
	ti.Prompt = "  "

	ta := textarea.New()
	ta.Placeholder = "Write…"
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	return entriesModel{api: c, titleInput: ti, bodyInput: ta, body: vp}
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
		if m.cursor >= len(m.list) {
			m.cursor = 0
		}
	case entriesErrMsg:
		m.err = msg.err.Error()
	case entryOpenedMsg:
		m.current = msg.entry
		m.body.SetContent(msg.entry.Body)
		m.body.GotoTop()
		m.mode = modeView
	case entryCreatedMsg:
		m.list = append([]api.EntryMeta{*msg.meta}, m.list...)
		m.mode = modeList
		m.cursor = 0
		m.titleInput.SetValue("")
		m.bodyInput.SetValue("")
	case entryDeletedMsg:
		out := m.list[:0]
		for _, e := range m.list {
			if e.ID != msg.id {
				out = append(out, e)
			}
		}
		m.list = out
		if m.cursor >= len(m.list) {
			m.cursor = max(0, len(m.list)-1)
		}
		m.current = nil
		m.mode = modeList
	}

	switch m.mode {
	case modeList:
		return m.updateList(msg)
	case modeView:
		return m.updateView(msg)
	case modeCompose:
		return m.updateCompose(msg)
	}
	return m, nil
}

func (m entriesModel) updateList(msg tea.Msg) (entriesModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "j", "down":
		if m.cursor < len(m.list)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(m.list) == 0 {
			return m, nil
		}
		id := m.list[m.cursor].ID
		c := m.api
		return m, func() tea.Msg {
			e, err := c.GetEntry(id)
			if err != nil {
				return entriesErrMsg{err: err}
			}
			return entryOpenedMsg{entry: e}
		}
	case "n":
		m.mode = modeCompose
		m.titleInput.Focus()
		return m, textinput.Blink
	case "r":
		return m, m.loadList()
	}
	return m, nil
}

func (m entriesModel) updateView(msg tea.Msg) (entriesModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "q":
			m.mode = modeList
			return m, nil
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
			m.mode = modeList
			m.titleInput.Blur()
			m.bodyInput.Blur()
			return m, nil
		case "ctrl+s":
			title := strings.TrimSpace(m.titleInput.Value())
			body := m.bodyInput.Value()
			if title == "" {
				m.err = "title required"
				return m, nil
			}
			c := m.api
			return m, func() tea.Msg {
				meta, err := c.CreateEntry(title, body)
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

func (m entriesModel) View() string {
	switch m.mode {
	case modeView:
		return m.renderView()
	case modeCompose:
		return m.renderCompose()
	}
	return m.renderList()
}

func (m entriesModel) renderList() string {
	if !m.loaded {
		return titleStyle.Render("inkwell") + "\n\n" + hintStyle.Render("loading…")
	}

	rowStyle := lipgloss.NewStyle().Padding(0, 2)
	selStyle := rowStyle.Copy().
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0a0a0a"}).
		Background(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f5f5f5"})

	var b strings.Builder
	b.WriteString(titleStyle.Render("inkwell — entries") + "\n\n")
	if len(m.list) == 0 {
		b.WriteString(hintStyle.Render("no entries yet · press n to compose · q to quit") + "\n")
	} else {
		for i, e := range m.list {
			line := e.Title + "  " + labelStyle.Render(e.CreatedAt.Local().Format("2006-01-02 15:04"))
			if i == m.cursor {
				b.WriteString(selStyle.Render("> "+line) + "\n")
			} else {
				b.WriteString(rowStyle.Render("  "+line) + "\n")
			}
		}
		b.WriteString("\n" + hintStyle.Render("enter open · n new · r refresh · q quit"))
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
	b.WriteString(hintStyle.Render("esc back · d delete · ↑/↓ scroll"))
	return b.String()
}

func (m entriesModel) renderCompose() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("new entry") + "\n\n")
	b.WriteString(labelStyle.Render("title") + "\n" + m.titleInput.View() + "\n\n")
	b.WriteString(labelStyle.Render("body") + "\n" + m.bodyInput.View() + "\n\n")
	b.WriteString(hintStyle.Render("ctrl+s save · tab switch field · esc cancel"))
	if m.err != "" {
		b.WriteString("\n\n" + errStyle.Render(m.err))
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
