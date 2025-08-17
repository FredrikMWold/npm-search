package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"npm-search/internal/ui/theme"
)

// Input is a wrapper component around bubbles textinput with rounded border.
type Input struct {
	ti     textinput.Model
	width  int
	style  lipgloss.Style
	height int
	focus  bool
}

func NewInput() *Input {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = ""
	ti.Focus()

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderFocused).
		Padding(0, 1) // small horizontal padding inside the border

	return &Input{ti: ti, style: style, height: 3, focus: true}
}

func (i *Input) Init() tea.Cmd { return textinput.Blink }

func (i *Input) SetWidth(w int) {
	if w < 2 {
		w = 2
	}
	i.width = w
	inner := max(1, w-2*1-2) // account for border width (2) + padding (1 left + 1 right)
	i.ti.Width = inner
}

func (i *Input) Height() int { return i.height }

// Value returns the current text typed in the input.
func (i *Input) Value() string { return i.ti.Value() }

// SetLabel renders a label inside the input by using the textinput prompt.
func (i *Input) SetLabel(text string, style lipgloss.Style) {
	i.ti.Prompt = text + " "
	i.ti.PromptStyle = style
}

func (i *Input) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	i.ti, cmd = i.ti.Update(msg)
	return cmd
}

func (i *Input) View() string {
	// Render text input within the styled box ensuring total width (including
	// border and padding) equals i.width. Rounded border adds 1 col per side
	// and we configured horizontal padding of 1 per side => subtract 2.
	innerWidth := max(0, i.width-2)
	box := i.style.Width(innerWidth).Render(i.ti.View())
	return box
}

func (i *Input) SetFocused(f bool) {
	i.focus = f
	if f {
		i.ti.Focus()
		i.style = i.style.BorderForeground(theme.BorderFocused)
	} else {
		i.ti.Blur()
		i.style = i.style.BorderForeground(theme.BorderUnfocused)
	}
}

// (Results removed; replaced by list component in a separate package.)

// max helper
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
