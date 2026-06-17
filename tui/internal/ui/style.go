package ui

import "github.com/charmbracelet/lipgloss"

// AdaptiveColor picks the variant matching the terminal background, so dark
// mode "just works" on dark terminals.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f5f5f5"}).
			Padding(0, 1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
			Italic(true)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#a02020", Dark: "#ff8080"})

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#aaaaaa"})

	// editorBoxStyle frames the compose/edit text area so its bounds are visible,
	// with a little breathing room padded inside the border.
	editorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#d0d0d0", Dark: "#444444"}).
			Padding(0, 1)
)
