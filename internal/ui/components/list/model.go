package list

import (
	"github.com/charmbracelet/bubbles/key"
	bblist "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"npm-search/internal/ui/theme"
)

// item implements bblist.Item
type item struct {
	title       string
	description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// Name returns the package name portion (title currently holds it)
func (i item) Name() string { return i.title }

// Model wraps a bubbles list inside a bordered container synced to size
// constraints from the parent.
type Model struct {
	width       int
	height      int
	style       lipgloss.Style
	list        bblist.Model
	focus       bool
	placeholder string
}

func New() *Model {
	// Start empty; we'll show a centered placeholder until we have results.
	var items []bblist.Item

	delegate := bblist.NewDefaultDelegate()
	// Theme normal and selected item styles
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(theme.Mauve).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(theme.Subtext0)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(theme.Text)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(theme.Surface2)
	l := bblist.New(items, delegate, 0, 0)
	l.Title = "Results"
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(theme.Subtext0)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(theme.Surface2)
	l.Styles.HelpStyle = lipgloss.NewStyle().Foreground(theme.Surface2)

	// Add custom help keybindings for install actions
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "npm install")),
			key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "npm install -D")),
		}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "npm install")),
			key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "npm install -D")),
		}
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderUnfocused).
		Foreground(theme.Text)

	return &Model{style: style, list: l, placeholder: "Type and press Enter to search."}
}

// SetSize sets the outer container size; the inner list is sized to fill it
// while accounting for borders.
func (m *Model) SetSize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 0 {
		h = 0
	}
	m.width, m.height = w, h
	innerW := max(0, w-2)
	innerH := max(0, h-2)
	m.list.SetSize(innerW, innerH)
}

func (m *Model) SetFocused(f bool) {
	m.focus = f
	if f {
		m.style = m.style.BorderForeground(theme.BorderFocused)
	} else {
		m.style = m.style.BorderForeground(theme.BorderUnfocused)
	}
}

// SetTitle sets the list's title string.
func (m *Model) SetTitle(title string) { m.list.Title = title }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *Model) View() string {
	innerW := max(0, m.width-2)
	innerH := max(0, m.height-2)
	var content string
	if len(m.list.Items()) == 0 {
		// Center the placeholder text when there are no items
		ph := lipgloss.NewStyle().Foreground(theme.Surface2).Italic(true).Render(m.placeholder)
		content = lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, ph)
	} else {
		content = m.list.View()
	}
	return m.style.
		Width(innerW).
		Height(innerH).
		Render(content)
}

// IsEmpty reports whether the list has any items.
func (m *Model) IsEmpty() bool { return len(m.list.Items()) == 0 }

// SetPlaceholder updates the empty-state text.
func (m *Model) SetPlaceholder(s string) { m.placeholder = s }

// SelectedName returns the currently selected package name, if any.
func (m *Model) SelectedName() (string, bool) {
	if it, ok := m.list.SelectedItem().(item); ok {
		return it.Name(), true
	}
	return "", false
}

// SetItems replaces the list items and optionally sets a title.
func (m *Model) SetItems(title string, items []struct{ Title, Description string }) {
	itms := make([]bblist.Item, 0, len(items))
	for _, it := range items {
		itms = append(itms, item{title: it.Title, description: it.Description})
	}
	m.list.SetItems(itms)
	if title != "" {
		m.list.Title = title
	}
}

// max helper (local copy)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
