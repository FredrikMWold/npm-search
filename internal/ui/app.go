package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-tui/internal/commands"
	"github.com/fredrikmwold/npm-tui/internal/ui/components"
	clist "github.com/fredrikmwold/npm-tui/internal/ui/components/list"
	"github.com/fredrikmwold/npm-tui/internal/ui/theme"
)

// Model is the root UI model.
type Model struct {
	width  int
	height int

	input  *components.Input
	list   *clist.Model
	side   *components.DetailsModel
	readme *components.MarkdownViewer
	// sequence for README requests to ignore stale responses
	readmeReq int
	// whether the README is currently loading
	readmeLoading bool
	// whether the sidebar is currently open
	sideOpen bool
	// whether the fullscreen README viewer is open
	readmeOpen bool
	focus      focusTarget

	// loading spinner for async searches
	spinner spinner.Model
	loading bool
	// per-row install spinner state
	installing map[string]bool
	// per-row install success state
	installed map[string]bool
	// timestamp of last mouse wheel event to disambiguate from Up/Down key events
	lastWheel time.Time
}

type focusTarget int

const (
	focusInput focusTarget = iota
	focusResults
	focusSide
)

func New() *Model {
	// configure spinner
	sp := spinner.New()
	// Use a line spinner everywhere
	sp.Spinner = spinner.Meter
	sp.Style = lipgloss.NewStyle().Foreground(theme.Mauve)

	return &Model{
		input:      components.NewInput(),
		list:       clist.New(),
		side:       components.NewDetails(),
		readme:     components.NewMarkdownViewer(),
		focus:      focusInput,
		spinner:    sp,
		installing: map[string]bool{},
		installed:  map[string]bool{},
	}
}

func (m *Model) Init() tea.Cmd {
	// start spinner ticking, scan local deps, and load project packages initially
	m.loading = true
	m.list.SetTitle("Loading project packages…")
	m.list.SetPlaceholder("Loading project packages…")
	return tea.Batch(m.input.Init(), m.spinner.Tick, commands.ScanInstalledDeps(), commands.LoadProjectPackages())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		// keep spinner running; update title/placeholder if loading
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			// Render the prefix with lavender background and padding, but use a
			// plain outer title style so the spinner remains unstyled.
			m.list.SetTitle(m.list.RenderPrefixedTitle("Searching npm", m.spinner.View()))
			m.list.UsePlainTitleStyle()
			m.list.SetPlaceholder(fmt.Sprintf("Searching npm %s", m.spinner.View()))
		}
		// If README is loading, update its spinner label as well
		if m.readmeOpen && m.readmeLoading {
			m.readme.SetLoading("Loading readme", m.spinner.View())
		}
		// Also update row spinner frame for installing packages
		m.list.SetRowSpinner(m.spinner.View())
		return m, cmd
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.recomputeLayout()
		// Update focus styles on resize as well.
		m.applyFocus()
		// Recenter loading state if README is loading
		if m.readmeOpen && m.readmeLoading {
			m.readme.SetLoading("Loading readme", m.spinner.View())
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			// If README is open, close it quickly without resetting state
			if m.readmeOpen {
				m.readmeOpen = false
				m.readmeLoading = false
				m.recomputeLayout()
				return m, nil
			}
			// Otherwise: Clear the input, return focus to it, close sidebar, and reload local packages
			m.input.Clear()
			m.focus = focusInput
			m.sideOpen = false
			m.readmeOpen = false
			m.readmeLoading = false
			m.applyFocus()
			// Trigger reload of project packages
			m.loading = true
			m.list.SetTitle("Loading project packages…")
			m.list.SetPlaceholder("Loading project packages…")
			// Recompute sizes after closing sidebar
			m.recomputeLayout()
			return m, commands.LoadProjectPackages()
		case tea.KeyTab:
			// cycle focus; when sidebar is open, include it in the cycle
			if m.sideOpen {
				switch m.focus {
				case focusInput:
					m.focus = focusResults
				case focusResults:
					m.focus = focusSide
				default:
					m.focus = focusInput
				}
			} else {
				if m.focus == focusInput {
					m.focus = focusResults
				} else {
					m.focus = focusInput
				}
			}
			m.applyFocus()
			return m, nil
		case tea.KeyRunes:
			// handled below
		case tea.KeyEnter:
			if m.focus == focusInput {
				// Move focus to results; only trigger search if query is non-empty
				q := m.input.Value()
				m.focus = focusResults
				m.applyFocus()
				if strings.TrimSpace(q) == "" {
					// Keep current items (e.g., project packages) and do not enter loading state
					return m, nil
				}
				m.loading = true
				m.list.SetTitle(m.list.RenderPrefixedTitle("Searching npm", m.spinner.View()))
				m.list.UsePlainTitleStyle()
				m.list.SetPlaceholder(fmt.Sprintf("Searching npm %s", m.spinner.View()))
				return m, tea.Batch(commands.SearchNPM(q))
			} else if m.focus == focusResults {
				// Toggle the sidebar when pressing Enter on results
				if m.sideOpen {
					m.sideOpen = false
					m.readmeOpen = false
					m.readmeLoading = false
					m.list.SetShowReadmeHotkey(false)
					// return focus to results when closing
					m.focus = focusResults
					m.applyFocus()
					// Recompute sizes for closed state
					m.recomputeLayout()
					return m, nil
				}
				// Open the sidebar with details for the selected item
				m.sideOpen = true
				// keep focus on results so arrows and wheel continue to work
				m.focus = focusResults
				m.applyFocus()
				m.list.SetShowReadmeHotkey(true)
				if det, ok := m.list.SelectedDetails(); ok {
					m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
					m.side.SetStats(det.StatsLine)
				}
				// Recompute sizes for open state
				m.recomputeLayout()
				// Kick off downloads range fetch for sparkline (last 365 days)
				if name, ok := m.list.SelectedName(); ok {
					return m, commands.FetchDownloadsRange(name, 365)
				}
				return m, nil
			}
		}
		// Rune key handling
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
			case 'u':
				if m.focus == focusResults {
					if name, ok := m.list.SelectedName(); ok {
						if m.installing == nil {
							m.installing = map[string]bool{}
						}
						m.installing[name] = true
						m.list.SetInstalling(m.installing)
						// Reuse install command which performs update when already installed
						return m, commands.InstallNPM(name, false)
					}
				}
			case 'r', 'R':
				// Only handle README toggle when results are focused and sidebar is open
				if m.focus != focusResults || !m.sideOpen {
					break
				}
				if m.readmeOpen {
					m.readmeOpen = false
					m.readmeLoading = false
					// When closing README, keep sidebar/results focus cycle intact
					if m.sideOpen {
						if m.focus == focusSide {
							// keep focus on sidebar after README closes
							m.focus = focusSide
						} else {
							m.focus = focusResults
						}
					} else {
						m.focus = focusResults
					}
					m.applyFocus()
					m.recomputeLayout()
					return m, nil
				}
				if det, ok := m.list.SelectedDetails(); ok {
					m.readmeOpen = true
					m.recomputeLayout()
					if det.Repository == "" {
						m.readme.SetPlain("No GitHub repository URL found for this package.")
						return m, nil
					}
					// Show spinner while loading the README
					m.readmeLoading = true
					m.readme.SetLoading("Loading readme", m.spinner.View())
					m.readmeReq++
					return m, commands.FetchGitHubReadmeWithReq(det.Repository, m.readmeReq)
				}
				return m, nil
			}
		}
	case commands.NpmSearchMsg:
		if msg.Err != nil {
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
		dlLabel := lipgloss.NewStyle().Foreground(theme.Sky).Bold(true).Render("Weekly Downloads:")
		licLabel := lipgloss.NewStyle().Foreground(theme.Yellow).Bold(true).Render("License:")
		autLabel := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true).Render("Author:")
		for _, o := range msg.Result.Objects {
			title := o.Package.Name
			line := fmt.Sprintf("%s %s  %s %s  %s %s  %s %s", verLabel, o.Package.Version, dlLabel, fmtInt(o.Package.DownloadsLastWeek), licLabel, nonEmpty(o.Package.License), autLabel, nonEmpty(o.Package.Author))
			full := o.Package.Description
			home := o.Package.Links.Homepage
			repo := o.Package.Links.Repository
			npm := o.Package.Links.NPM
			items = append(items, clist.ItemWithMeta{Title: title, LineDesc: line, FullDesc: full, Homepage: home, Repository: repo, NPMLink: npm, Latest: o.Package.Version})
		}
		m.loading = false
		// restore default title style (with lavender background) for regular titles
		m.list.UseDefaultTitleStyle()
		// send items with metadata for sidebar
		// convert to the specialized setter to preserve extra fields
		if msg.Query == "" {
			m.list.SetItemsWithMeta("Project packages", items)
			if len(items) == 0 {
				m.list.SetTitle("Project packages")
				m.list.SetPlaceholder("No packages found in package.json. Type and press Enter to search.")
			} else {
				m.list.SetTitle("Project packages")
				m.list.SetPlaceholder("Press Enter for details, Tab to toggle focus. Type and Enter to search.")
			}
		} else {
			m.list.SetItemsWithMeta("Results", items)
			m.list.SetTitle("Results")
			m.list.SetPlaceholder("Type and press Enter to search.")
		}
		// close sidebar and README by default after a new search
		m.sideOpen = false
		m.readmeOpen = false
		m.readmeLoading = false
		m.list.SetShowReadmeHotkey(false)
		m.side.SetContent("", "", "", "", "")
		m.side.SetStats("")
		m.side.SetDownloadsValues(nil)
		// ensure sizing is recomputed on new data
		m.recomputeLayout()
		// initialize sidebar with first selection, if any
		if det, ok := m.list.SelectedDetails(); ok {
			m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
			m.side.SetStats(det.StatsLine)
		} else {
			m.side.SetContent("", "", "", "", "")
			m.side.SetStats("")
		}
		// refresh installed marks against current package.json
		return m, commands.ScanInstalledDeps()
	case commands.ScanDepsMsg:
		if msg.Installed != nil {
			// merge known installed from scans with runtime installs
			if m.installed == nil {
				m.installed = map[string]bool{}
			}
			for k, v := range msg.Installed {
				if v {
					m.installed[k] = true
				}
			}
			m.list.SetInstalled(m.installed)
			// provide wanted (manifest) versions for update detection
			m.list.SetWantedVersions(msg.Wanted)
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
			// rescan package.json to refresh installed and wanted versions
			return m, commands.ScanInstalledDeps()
		}
		return m, nil
	case commands.GitHubReadmeMsg:
		// Render markdown asynchronously for responsiveness
		if msg.Err != nil {
			m.readmeLoading = false
			if m.readmeOpen {
				m.readme.SetPlain("Could not load README:\n\n" + msg.Err.Error())
				// Ensure viewport processes the new content immediately
				if cmd := m.readme.Update(nil); cmd != nil {
					return m, cmd
				}
			}
			return m, nil
		}
		// Ignore stale responses
		if msg.Req != m.readmeReq {
			return m, nil
		}
		// Render synchronously so the formatted output shows immediately
		if m.readmeOpen {
			m.readmeLoading = false
			m.readme.SetMarkdown(msg.Content)
			// Ensure viewport processes the new content immediately
			if cmd := m.readme.Update(nil); cmd != nil {
				return m, cmd
			}
		}
		return m, nil
	case components.MarkdownRenderedMsg:
		// Ignore stale renders not matching the current request
		if msg.Seq != m.readmeReq {
			return m, nil
		}
		// Ensure we handle the async rendered content promptly
		if cmd := m.readme.Update(msg); cmd != nil {
			return m, cmd
		}
		return m, nil
	}

	// Let the focused component handle the message.
	var cmds []tea.Cmd
	// Input routing to ensure correct scrolling behavior
	if !m.readmeOpen {
		switch t := msg.(type) {
		case tea.MouseMsg:
			if m.sideOpen {
				// Mouse wheel should scroll the sidebar only; ignore other mouse events here
				mm := t
				if mm.Type == tea.MouseWheelUp || mm.Type == tea.MouseWheelDown {
					m.lastWheel = time.Now()
					if cmd := m.side.Update(msg); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
				return m, tea.Batch(cmds...)
			}
			// Sidebar closed: route only wheel events to the list (ignore motion/click noise)
			mm := t
			if mm.Type == tea.MouseWheelUp || mm.Type == tea.MouseWheelDown {
				if cmd := m.list.Update(msg); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			// Update sidebar content if selection moved
			if det, ok := m.list.SelectedDetails(); ok {
				m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
				m.side.SetStats(det.StatsLine)
				if m.sideOpen {
					if name, ok2 := m.list.SelectedName(); ok2 {
						cmds = append(cmds, commands.FetchDownloadsRange(name, 365))
					}
				}
			}
			return m, tea.Batch(cmds...)
		case tea.KeyMsg:
			switch t.Type {
			case tea.KeyUp, tea.KeyDown:
				// If a wheel event just occurred, ignore synthetic Up/Down to prevent flicker
				if m.sideOpen && time.Since(m.lastWheel) < 200*time.Millisecond {
					return m, nil
				}
				// Arrow keys scroll the list
				if cmd := m.list.Update(msg); cmd != nil {
					cmds = append(cmds, cmd)
				}
				// Update sidebar content as selection changes
				if det, ok := m.list.SelectedDetails(); ok {
					m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
					m.side.SetStats(det.StatsLine)
					if m.sideOpen {
						if name, ok2 := m.list.SelectedName(); ok2 {
							cmds = append(cmds, commands.FetchDownloadsRange(name, 365))
						}
					}
				}
				return m, tea.Batch(cmds...)
			}
		}
	}
	if m.readmeOpen {
		if cmd := m.readme.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else if m.focus == focusInput {
		if cmd := m.input.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else if m.focus == focusResults {
		// If arrow keys were handled earlier, they already updated the list and sidebar
		// This branch handles other messages destined for the list (non-arrow keys/misc)
		if cmd := m.list.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		if det, ok := m.list.SelectedDetails(); ok {
			m.side.SetContent(det.Name, det.Description, det.Homepage, det.Repository, det.NPMLink)
			m.side.SetStats(det.StatsLine)
			if m.sideOpen {
				if name, ok2 := m.list.SelectedName(); ok2 {
					cmds = append(cmds, commands.FetchDownloadsRange(name, 365))
				}
			}
		}
	} else if m.focus == focusSide {
		if m.sideOpen {
			if cmd := m.side.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	// Handle async downloads-range results outside the large switch for clarity
	switch msg := msg.(type) {
	case commands.NpmDownloadsRangeMsg:
		if msg.Err == nil && msg.Package != "" {
			m.side.SetDownloadsValues(msg.Values)
			if len(msg.Points) > 0 {
				ts := make([]time.Time, 0, len(msg.Points))
				for _, p := range msg.Points {
					ts = append(ts, p.Time)
				}
				m.side.SetDownloadsPoints(ts)
			} else {
				m.side.SetDownloadsPoints(nil)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "" // wait for initial size
	}
	// Render input and results. The input renders its own inline label.
	m.input.SetLabel("npm-search:", lipgloss.NewStyle().Foreground(theme.Subtext0))
	inputView := m.input.View()
	// When README is open, use the full area below the input
	var body string
	if m.readmeOpen {
		body = m.readme.View()
	} else {
		// Two-column layout: list + sidebar
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), m.side.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, inputView, body)
}

// Helpers
var _ tea.Model = (*Model)(nil)

func (m *Model) applyFocus() {
	m.input.SetFocused(m.focus == focusInput)
	m.list.SetFocused(m.focus == focusResults)
	m.side.SetFocused(m.focus == focusSide)
}

//

// recomputeLayout updates child sizes based on current width/height/sidebar state.
func (m *Model) recomputeLayout() {
	m.input.SetWidth(m.width)
	// Height remaining for list/sidebar
	remaining := m.height - m.input.Height()
	if remaining < 0 {
		remaining = 0
	}
	if m.readmeOpen {
		// Full width for README viewer
		m.readme.SetSize(m.width, remaining)
		// Still keep list/side sizes updated (hidden) to avoid surprises on return
		listW, sideW := computeSplit(m.width, m.sideOpen)
		m.list.SetSize(listW, remaining)
		m.side.SetSize(sideW, remaining)
		return
	}
	listW, sideW := computeSplit(m.width, m.sideOpen)
	m.list.SetSize(listW, remaining)
	m.side.SetSize(sideW, remaining)
}

// fmtInt formats an int with thousand separators; moved to helpers.go
