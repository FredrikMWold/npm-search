package theme

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette (subset)
var (
	// Core
	Base     = lipgloss.Color("#1e1e2e")
	Mantle   = lipgloss.Color("#181825")
	Crust    = lipgloss.Color("#11111b")
	Text     = lipgloss.Color("#cdd6f4")
	Subtext0 = lipgloss.Color("#a6adc8")
	Surface0 = lipgloss.Color("#313244")
	Surface1 = lipgloss.Color("#45475a")
	Surface2 = lipgloss.Color("#585b70")

	// Accents
	Mauve = lipgloss.Color("#cba6f7")
	Blue  = lipgloss.Color("#89b4fa")
	Green = lipgloss.Color("#a6e3a1")
	Peach = lipgloss.Color("#fab387")
	Red   = lipgloss.Color("#f38ba8")
)

// Convenience
var (
	BorderUnfocused = Surface2
	BorderFocused   = Mauve
)
