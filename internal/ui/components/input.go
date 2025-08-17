package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-search/internal/ui/theme"
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
	// Colorful cursor and placeholder
	// Set cursor color using the new API
	c := ti.Cursor
	c.Style = lipgloss.NewStyle().Foreground(theme.Mauve)
	ti.Cursor = c
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Surface2).Italic(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderFocused).
		Padding(0, 0)

	return &Input{ti: ti, style: style, height: 3, focus: true}
}

func (i *Input) Init() tea.Cmd { return textinput.Blink }

func (i *Input) SetWidth(w int) {
	if w < 2 {
		w = 2
	}
	i.width = w
	// account for border width (2), no internal padding
	inner := max(1, w-2)
	i.ti.Width = inner
}

func (i *Input) Height() int { return i.height }

// Value returns the current text typed in the input.
func (i *Input) Value() string { return i.ti.Value() }

// SetLabel renders a label inside the input by using the textinput prompt.
func (i *Input) SetLabel(text string, style lipgloss.Style) {
	// Build a colorful prompt inside the border. Keep single color (npm red)
	trimmed := strings.TrimSpace(text)
	var b strings.Builder
	// icon at far left inside border
	icon := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render("❯")
	b.WriteString(icon)
	// space after icon, then red label (including any colon)
	b.WriteString(" ")
	label := lipgloss.NewStyle().Foreground(theme.Red).Render(trimmed)
	b.WriteString(label)
	// right padding after label
	b.WriteString(" ")
	i.ti.Prompt = b.String()
	// Keep prompt style minimal; colors are embedded already
	i.ti.PromptStyle = lipgloss.NewStyle()
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
	innerWidth := intMax(0, i.width-2)
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

// Clear resets the input value and cursor position.
func (i *Input) Clear() {
	i.ti.SetValue("")
	// Move cursor to start to avoid any residual position
	i.ti.SetCursor(0)
}

//
