package ui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/phekno/inkwell/tui/internal/cognito"
)

type loginModel struct {
	email, password textinput.Model
	focused         int // 0 = email, 1 = password
	cognito         *cognito.Client
	submitting      bool
	err             string
}

func newLogin(c *cognito.Client) loginModel {
	e := textinput.New()
	e.Placeholder = "you@example.com"
	e.Focus()
	e.Prompt = "  "

	p := textinput.New()
	p.Placeholder = "password"
	p.EchoMode = textinput.EchoPassword
	p.EchoCharacter = '•'
	p.Prompt = "  "

	return loginModel{email: e, password: p, cognito: c}
}

func (m loginModel) Init() tea.Cmd { return textinput.Blink }

func (m loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case signInErrMsg:
		m.submitting = false
		m.err = msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.toggleFocus()
			return m, nil
		case "shift+tab", "up":
			m.toggleFocus()
			return m, nil
		case "enter":
			if m.submitting {
				return m, nil
			}
			if m.focused == 0 && strings.TrimSpace(m.email.Value()) != "" {
				m.toggleFocus()
				return m, nil
			}
			email := strings.TrimSpace(strings.ToLower(m.email.Value()))
			pwd := m.password.Value()
			if email == "" || pwd == "" {
				m.err = "email and password required"
				return m, nil
			}
			m.submitting = true
			m.err = ""
			c := m.cognito
			return m, func() tea.Msg {
				s, err := c.SignIn(context.Background(), email, pwd)
				if err != nil {
					return signInErrMsg{err: err}
				}
				_ = cognito.SaveSession(s)
				return signedInMsg{session: s}
			}
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	if m.focused == 0 {
		m.email, cmd = m.email.Update(msg)
	} else {
		m.password, cmd = m.password.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *loginModel) toggleFocus() {
	if m.focused == 0 {
		m.email.Blur()
		m.password.Focus()
		m.focused = 1
	} else {
		m.password.Blur()
		m.email.Focus()
		m.focused = 0
	}
}

func (m loginModel) View() string {
	box := lipgloss.NewStyle().Padding(2, 4).Width(46)

	var b strings.Builder
	b.WriteString(titleStyle.Render("inkwell") + "\n\n")
	b.WriteString(labelStyle.Render("email") + "\n" + m.email.View() + "\n\n")
	b.WriteString(labelStyle.Render("password") + "\n" + m.password.View() + "\n\n")
	if m.err != "" {
		b.WriteString(errStyle.Render(m.err) + "\n\n")
	}
	if m.submitting {
		b.WriteString(hintStyle.Render("signing in…"))
	} else {
		b.WriteString(hintStyle.Render("tab to switch · enter to submit · ctrl+c to quit"))
	}
	return box.Render(b.String())
}
