package list

import (
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	bblist "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-search/internal/ui/theme"
)

// item implements bblist.Item
type item struct {
	title       string
	description string
	// extra metadata for sidebar
	fullDesc string
	homepage string
	repo     string
	npmLink  string
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
	del         *delegate
}

func New() *Model {
	// Start empty; we'll show a centered placeholder until we have results.
	var items []bblist.Item

	// Create custom delegate (wrap default styles)
	d := newDelegate()
	// Theme normal and selected item styles
	// SelectedTitle includes a left border which acts as the selection cursor;
	// color it with our theme accent.
	d.DefaultDelegate.Styles.SelectedTitle = d.DefaultDelegate.Styles.SelectedTitle.
		Foreground(theme.Mauve).
		BorderForeground(theme.Mauve).
		Bold(true)
	d.DefaultDelegate.Styles.SelectedDesc = d.DefaultDelegate.Styles.SelectedDesc.
		Foreground(theme.Mauve).
		BorderForeground(theme.Mauve)
	d.DefaultDelegate.Styles.NormalTitle = d.DefaultDelegate.Styles.NormalTitle.Foreground(theme.Text)
	d.DefaultDelegate.Styles.NormalDesc = d.DefaultDelegate.Styles.NormalDesc.Foreground(theme.Surface2)

	l := bblist.New(items, d, 0, 0)
	l.Title = "Results"
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	// Disable built-in list filtering; searching is handled by the top input
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Foreground(theme.Subtext0)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(theme.Surface2)
	l.Styles.HelpStyle = lipgloss.NewStyle().Foreground(theme.Surface2)

	// Add custom help keybindings for install actions
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
			key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "install dev")),
		}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
			key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "install dev")),
		}
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderUnfocused).
		Foreground(theme.Text)

	return &Model{style: style, list: l, placeholder: "Type and press Enter to search.", del: d}
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
	// Dynamically toggle help/pagination based on available height AND width to avoid wrapping
	// Help can get quite long with default + custom keys; hide it on narrow widths
	const minHelpWidth = 52
	const minPagWidth = 24
	showHelp := innerH >= 6 && innerW >= minHelpWidth
	showPagination := innerH >= 4 && innerW >= minPagWidth
	m.list.SetShowHelp(showHelp)
	m.list.SetShowPagination(showPagination)
	m.list.SetShowStatusBar(false)

	// Constrain help style width to reduce chance of internal wrapping
	m.list.Styles.HelpStyle = lipgloss.NewStyle().Foreground(theme.Surface2).MaxWidth(innerW)

	// Account for list chrome (title + optional pagination + optional help)
	chrome := 0
	if m.list.Title != "" {
		chrome++
	}
	if showPagination {
		chrome++
	}
	if showHelp {
		chrome++
	}
	viewportH := max(0, innerH-chrome)
	m.list.SetSize(innerW, viewportH)

	// Calibrate viewport to make total rendered height match the inner box.
	// This compensates for any minor off-by-one differences from chrome.
	for i := 0; i < 3; i++ {
		lines := countLines(m.list.View())
		delta := innerH - lines
		if delta == 0 {
			break
		}
		viewportH = max(0, viewportH+delta)
		m.list.SetSize(innerW, viewportH)
	}
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
		// Render list and crop/top-align within the inner box.
		content = lipgloss.Place(innerW, innerH, lipgloss.Left, lipgloss.Top, m.list.View())
	}
	// Ensure the border wraps exactly the inner area and content fills it
	body := lipgloss.Place(innerW, innerH, lipgloss.Left, lipgloss.Top, content)
	return m.style.Width(innerW).Height(innerH).Render(body)
}

// countLines returns the number of lines in s when rendered.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	// Normalize trailing newline so split counts real lines
	if strings.HasSuffix(s, "\n") {
		s = strings.TrimRight(s, "\n")
	}
	return len(strings.Split(s, "\n"))
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

// Details holds the metadata needed by the sidebar for the selected item.
type Details struct {
	Name        string
	Description string
	StatsLine   string
	Homepage    string
	Repository  string
	NPMLink     string
}

// SelectedDetails returns sidebar-ready metadata for the currently selected item.
func (m *Model) SelectedDetails() (Details, bool) {
	if it, ok := m.list.SelectedItem().(item); ok {
		return Details{
			Name:        it.title,
			Description: it.fullDesc,
			StatsLine:   it.description,
			Homepage:    it.homepage,
			Repository:  it.repo,
			NPMLink:     it.npmLink,
		}, true
	}
	return Details{}, false
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

// ItemWithMeta is the input struct for SetItemsWithMeta with sidebar data.
type ItemWithMeta struct {
	Title      string
	LineDesc   string
	FullDesc   string
	Homepage   string
	Repository string
	NPMLink    string
}

// SetItemsWithMeta replaces items and attaches metadata for the sidebar.
func (m *Model) SetItemsWithMeta(title string, items []ItemWithMeta) {
	itms := make([]bblist.Item, 0, len(items))
	for _, it := range items {
		itms = append(itms, item{
			title:       it.Title,
			description: it.LineDesc,
			fullDesc:    it.FullDesc,
			homepage:    it.Homepage,
			repo:        it.Repository,
			npmLink:     it.NPMLink,
		})
	}
	m.list.SetItems(itms)
	if title != "" {
		m.list.Title = title
	}
}

// SetInstalling replaces the set of installing package names.
func (m *Model) SetInstalling(installing map[string]bool) {
	if m.del != nil {
		m.del.installing = installing
	}
}

// SetRowSpinner sets the spinner frame used for installing rows.
func (m *Model) SetRowSpinner(frame string) {
	if m.del != nil {
		m.del.frame = frame
	}
}

// SetInstalled marks which package names were installed successfully to show
// a green checkmark prefix.
func (m *Model) SetInstalled(installed map[string]bool) {
	if m.del != nil {
		m.del.installed = installed
	}
}

// max helper (local copy)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// delegate customizes row rendering to show a spinner for installing items.
type delegate struct {
	bblist.DefaultDelegate
	installing map[string]bool
	installed  map[string]bool
	frame      string
}

func newDelegate() *delegate {
	d := &delegate{
		DefaultDelegate: bblist.NewDefaultDelegate(),
		installing:      map[string]bool{},
	}
	return d
}

// Render prints each list item with optional spinner when installing using the
// DefaultDelegate to preserve correct height/spacing.
func (d *delegate) Render(w io.Writer, m bblist.Model, index int, listItem bblist.Item) {
	it, _ := listItem.(item)
	prefix := ""
	if d.installing != nil && d.installing[it.Name()] {
		prefix = d.frame + " "
	} else if d.installed != nil && d.installed[it.Name()] {
		// green checkmark for successful install
		check := lipgloss.NewStyle().Foreground(theme.Green).Render("✔")
		prefix = check + " "
	}
	// Wrap the item to override Title() with spinner prefix while preserving
	// default height/formatting.
	wi := wrappedItem{item: it, pre: prefix}
	d.DefaultDelegate.Render(w, m, index, wi)
}

// wrappedItem decorates an item with a prefix for the Title while delegating
// other methods to the embedded item.
type wrappedItem struct {
	item
	pre string
}

func (w wrappedItem) Title() string { return w.pre + w.item.Title() }
