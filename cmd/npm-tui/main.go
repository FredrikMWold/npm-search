package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/fredrikmwold/npm-tui/internal/ui"
)

func main() {
	app := ui.New()
	// Do not enable Bubble Tea mouse reporting here because when the program
	// enables mouse reporting the terminal forwards mouse events to the
	// application which in many terminals disables clickable OSC8 hyperlinks.
	// Leaving out mouse reporting lets the terminal handle clicks so sidebar
	// links can be opened by clicking them. Mouse wheel and other mouse
	// interactions will fall back to keyboard navigation (PgUp/PgDn/ arrows).
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
