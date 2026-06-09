package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/phekno/inkwell/tui/internal/ui"
)

func main() {
	app, err := ui.New(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "inkwell:", err)
		os.Exit(1)
	}
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "inkwell:", err)
		os.Exit(1)
	}
}
