package ui

import (
	"fmt"
	"log"
	"math"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-search/internal/commands"
	"github.com/fredrikmwold/npm-search/internal/ui/components"
	clist "github.com/fredrikmwold/npm-search/internal/ui/components/list"
	"github.com/fredrikmwold/npm-search/internal/ui/theme"
)

// Model is the root UI model.
type Model struct {
	width  int
	height int

	input *components.Input
	list  *clist.Model
	side  *components.DetailsModel
	// whether the sidebar is currently open
	sideOpen bool
	focus    focusTarget

	// loading spinner for async searches
	spinner spinner.Model
	loading bool
	// per-row install spinner state
	installing map[string]bool
	// per-row install success state
	installed map[string]bool
}

type focusTarget int

const (
	focusInput focusTarget = iota
	focusResults
)

func New() *Model {
	// configure spinner
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(theme.Mauve)

	return &Model{
		input:      components.NewInput(),
		list:       clist.New(),
		side:       components.NewDetails(),
		focus:      focusInput,
		spinner:    sp,
		installing: map[string]bool{},
		installed:  map[string]bool{},
	}
}

func (m *Model) Init() tea.Cmd {
	// start spinner ticking (we'll render it only when loading)
	return tea.Batch(m.input.Init(), m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		// keep spinner running; update title/placeholder if loading
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			m.list.SetTitle(fmt.Sprintf("Searching npm %s", m.spinner.View()))
			m.list.SetPlaceholder(fmt.Sprintf("Searching npm %s", m.spinner.View()))
		}
		// Also update row spinner frame for installing packages
		m.list.SetRowSpinner(m.spinner.View())
		return m, cmd
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// Update component sizes.
		m.input.SetWidth(m.width)
		// Reserve space for the results box below.
		inputHeight := m.input.Height()
		remaining := int(math.Max(0, float64(m.height-inputHeight)))
		// Split remaining into list and sidebar (70/30 when open and wide enough)
		listW, sideW := computeSplit(m.width, m.sideOpen)
		m.list.SetSize(listW, remaining)
		m.side.SetSize(sideW, remaining)
		// Update focus styles on resize as well.
		m.applyFocus()
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			// Always bring focus back to input on Escape and close sidebar
			m.focus = focusInput
			m.sideOpen = false
			m.applyFocus()
			// Recompute sizes after closing sidebar
			inputHeight := m.input.Height()
			remaining := int(math.Max(0, float64(m.height-inputHeight)))
			listW, sideW := computeSplit(m.width, m.sideOpen)
			m.list.SetSize(listW, remaining)
			m.side.SetSize(sideW, remaining)
			return m, nil
		case tea.KeyRunes:
			// noop, handled below
		case tea.KeyEnter:
			if m.focus == focusInput {
				// Trigger search and move focus to results
				q := m.input.Value()
				m.focus = focusResults
				m.applyFocus()
				m.loading = true
				m.list.SetTitle(fmt.Sprintf("Searching npm %s", m.spinner.View()))
				m.list.SetPlaceholder(fmt.Sprintf("Searching npm %s", m.spinner.View()))
				return m, tea.Batch(commands.SearchNPM(q))
			} else if m.focus == focusResults {
				// Toggle the sidebar when pressing Enter on results
				if m.sideOpen {
					m.sideOpen = false
					// Recompute sizes for closed state
					inputHeight := m.input.Height()
					remaining := int(math.Max(0, float64(m.height-inputHeight)))
					listW, sideW := computeSplit(m.width, m.sideOpen)
					m.list.SetSize(listW, remaining)
					m.side.SetSize(sideW, remaining)
					return m, nil
				}
				// Open the sidebar with details for the selected item
				m.sideOpen = true
				if det, ok := m.list.SelectedDetails(); ok {
					m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
					m.side.SetStats(det.StatsLine)
				}
				// Recompute sizes for open state
				inputHeight := m.input.Height()
				remaining := int(math.Max(0, float64(m.height-inputHeight)))
				listW, sideW := computeSplit(m.width, m.sideOpen)
				m.list.SetSize(listW, remaining)
				m.side.SetSize(sideW, remaining)
				return m, nil
			}
		}
		// Rune key handling (lowercase/uppercase i)
		if r := msg.Runes; len(r) == 1 {
			switch r[0] {
			case 'i':
				if m.focus == focusResults {
					if name, ok := m.list.SelectedName(); ok {
						// mark installing and kick off command
						if m.installing == nil {
							m.installing = map[string]bool{}
						}
						m.installing[name] = true
						m.list.SetInstalling(m.installing)
						return m, commands.InstallNPM(name, false)
					}
				}
			case 'I':
				if m.focus == focusResults {
					if name, ok := m.list.SelectedName(); ok {
						if m.installing == nil {
							m.installing = map[string]bool{}
						}
						m.installing[name] = true
						m.list.SetInstalling(m.installing)
						return m, commands.InstallNPM(name, true)
					}
				}
			}
		}
	case commands.NpmSearchMsg:
		if msg.Err != nil {
			log.Printf("npm search error for %q: %v", msg.Query, msg.Err)
			// stop loading state on error as well
			m.loading = false
			m.list.SetTitle("Results")
			m.list.SetPlaceholder("Type and press Enter to search.")
			return m, nil
		}
		// Map results into list items, include weekly downloads and author
		items := make([]clist.ItemWithMeta, 0, len(msg.Result.Objects))
		// Use Blue for Version to avoid clashing with the selected row color
		verLabel := lipgloss.NewStyle().Foreground(theme.Blue).Bold(true).Render("Version:")
		dlLabel := lipgloss.NewStyle().Foreground(theme.Green).Bold(true).Render("Download:")
		licLabel := lipgloss.NewStyle().Foreground(theme.Peach).Bold(true).Render("License:")
		autLabel := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render("Author:")
		for _, o := range msg.Result.Objects {
			title := o.Package.Name
			line := fmt.Sprintf("%s %s  %s %s  %s %s  %s %s", verLabel, o.Package.Version, dlLabel, fmtInt(o.Package.DownloadsLastWeek), licLabel, nonEmpty(o.Package.License), autLabel, nonEmpty(o.Package.Author))
			full := o.Package.Description
			home := o.Package.Links.Homepage
			repo := o.Package.Links.Repository
			npm := o.Package.Links.NPM
			items = append(items, clist.ItemWithMeta{Title: title, LineDesc: line, FullDesc: full, Homepage: home, Repository: repo, NPMLink: npm})
		}
		m.loading = false
		// send items with metadata for sidebar
		// convert to the specialized setter to preserve extra fields
		m.list.SetItemsWithMeta("Results", items)
		m.list.SetTitle("Results")
		m.list.SetPlaceholder("Type and press Enter to search.")
		// close sidebar by default after a new search
		m.sideOpen = false
		m.side.SetContent("", "", "", "", "")
		m.side.SetStats("")
		// ensure sizing is recomputed on new data
		inputHeight := m.input.Height()
		remaining := int(math.Max(0, float64(m.height-inputHeight)))
		listW, sideW := computeSplit(m.width, m.sideOpen)
		m.list.SetSize(listW, remaining)
		m.side.SetSize(sideW, remaining)
		// initialize sidebar with first selection, if any
		if det, ok := m.list.SelectedDetails(); ok {
			m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
			m.side.SetStats(det.StatsLine)
		} else {
			m.side.SetContent("", "", "", "", "")
			m.side.SetStats("")
		}
		return m, nil
	case commands.NpmInstallMsg:
		// clear installing flag for the package
		if msg.Package != "" && m.installing != nil {
			delete(m.installing, msg.Package)
			m.list.SetInstalling(m.installing)
		}
		// mark success (no error) to show checkmark
		if msg.Package != "" && msg.Err == nil {
			if m.installed == nil {
				m.installed = map[string]bool{}
			}
			m.installed[msg.Package] = true
			m.list.SetInstalled(m.installed)
		}
		return m, nil
	}

	// Let the focused component handle the message.
	var cmds []tea.Cmd
	if m.focus == focusInput {
		if cmd := m.input.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else {
		if cmd := m.list.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Update sidebar live on selection change
		if det, ok := m.list.SelectedDetails(); ok {
			m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
			m.side.SetStats(det.StatsLine)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "" // wait for initial size
	}
	// Render input and results. The input renders its own inline label.
	m.input.SetLabel("npm search:", lipgloss.NewStyle().Foreground(theme.Subtext0))
	inputView := m.input.View()
	// Two-column layout: list + sidebar
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), m.side.View())
	return lipgloss.JoinVertical(lipgloss.Left, inputView, body)
}

// Helpers
var _ tea.Model = (*Model)(nil)

func (m *Model) applyFocus() {
	m.input.SetFocused(m.focus == focusInput)
	m.list.SetFocused(m.focus == focusResults)
	m.side.SetFocused(m.focus == focusResults)
}

// (search command/types moved to internal/commands)

// fmtInt formats an int with thin thousand separators for readability.
func fmtInt(n int) string {
	s := fmt.Sprintf("%d", n)
	// insert separators from the right
	out := make([]byte, 0, len(s)+len(s)/3)
	cnt := 0
	for i := len(s) - 1; i >= 0; i-- {
		out = append(out, s[i])
		cnt++
		if cnt%3 == 0 && i != 0 {
			out = append(out, ',')
		}
	}
	// reverse
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func nonEmpty(s string) string {
	if s == "" {
		return "n/a"
	}
	return s
}

// computeSplit returns widths for list and sidebar based on total width.
func computeSplit(total int, open bool) (listW, sideW int) {
	if !open {
		return total, 0
	}
	if total <= 48 {
		// not enough width, hide sidebar
		return total, 0
	}
	// keep at least 22 cols for sidebar
	side := int(math.Max(22, math.Round(float64(total)*0.32)))
	// ensure list has room
	if side > total-20 {
		side = total - 20
	}
	if side < 0 {
		side = 0
	}
	return total - side, side
}
