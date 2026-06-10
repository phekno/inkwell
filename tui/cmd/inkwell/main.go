package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/phekno/inkwell/tui/internal/ui"
)

// Populated at build time by goreleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("inkwell %s (%s, %s)\n", version, commit, date)
		return
	}

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
