package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// AdaptiveColor picks the variant matching the terminal background, so
	// dark mode "just works" for users on dark terminals.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f5f5f5"}).
			Padding(0, 1)

	bodyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#3a3a3a", Dark: "#cccccc"}).
			Padding(1, 2)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
			Italic(true).
			Padding(0, 2)
)

type Model struct {
	width, height int
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	return titleStyle.Render("inkwell") + "\n" +
		bodyStyle.Render("Your journal, encrypted in the cloud.") + "\n" +
		hintStyle.Render("press q to quit")
}
