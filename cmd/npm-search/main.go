package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"npm-search/internal/ui"
)

func main() {
	// Setup logging to a file

	app := ui.New()
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
