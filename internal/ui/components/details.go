package components

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-tui/internal/ui/theme"
)

// DetailsModel renders a sidebar with info about a selected package.
type DetailsModel struct {
	width  int
	height int
	style  lipgloss.Style
	// scrollable content area
	vp viewport.Model

	// content
	title       string
	description string
	stats       string
	homepage    string
	repository  string
	npmLink     string

	// downloads over time series
	dlValues []float64
	dlTimes  []time.Time
	// cached rendered chart string for current width
	dlRendered string
	// cached content string and dirty flag
	content string
	dirty   bool
	// track when we need to reset scroll to top (e.g., on selection change)
	resetTop  bool
	lastTitle string
}

func NewDetails() *DetailsModel {
	st := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderUnfocused).
		Foreground(theme.Text)
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	// viewport is unstyled; outer style draws the border
	return &DetailsModel{style: st, vp: vp, dirty: true}
}

func (d *DetailsModel) Init() tea.Cmd { return nil }

func (d *DetailsModel) SetSize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 0 {
		h = 0
	}
	// Invalidate cached sparkline rendering on any size change
	if w != d.width || h != d.height {
		d.dlRendered = ""
	}
	d.width, d.height = w, h
	// viewport fills the inner area within the border
	innerW := intMax(0, d.width-2)
	innerH := intMax(0, d.height-2)
	d.vp.Width = innerW
	d.vp.Height = innerH
	d.dirty = true
}

func (d *DetailsModel) SetFocused(f bool) {
	if f {
		d.style = d.style.BorderForeground(theme.BorderFocused)
	} else {
		d.style = d.style.BorderForeground(theme.BorderUnfocused)
	}
}

// SetContent updates the sidebar content.
func (d *DetailsModel) SetContent(title, desc, homepage, repo, npmLink string) {
	if title != d.title {
		d.lastTitle = d.title
		d.title = title
		// selection changed -> reset scroll to top next render
		d.resetTop = true
	} else {
		d.title = title
	}
	d.description = desc
	d.homepage = homepage
	d.repository = repo
	d.npmLink = npmLink
	d.dirty = true
}

// SetStats sets the one-line stats string (version/downloads/license/author)
func (d *DetailsModel) SetStats(s string) { d.stats = s; d.dirty = true }

// SetDownloadsValues sets the downloads-over-time values to be displayed as a sparkline.
func (d *DetailsModel) SetDownloadsValues(vals []float64) {
	d.dlValues = vals
	d.dlRendered = "" // invalidate cache
	d.dirty = true
}

// SetDownloadsPoints sets the time axis for the downloads chart.
func (d *DetailsModel) SetDownloadsPoints(times []time.Time) {
	d.dlTimes = times
	d.dlRendered = ""
	d.dirty = true
}

func (d *DetailsModel) Update(msg tea.Msg) tea.Cmd {
	switch t := msg.(type) {
	case tea.MouseMsg:
		// Only process wheel scroll; ignore other mouse events to avoid conflicts
		mm := t
		if mm.Type == tea.MouseWheelUp || mm.Type == tea.MouseWheelDown {
			// Smooth, small-step scrolling to reduce flicker/jumps
			step := 2
			if mm.Type == tea.MouseWheelUp {
				d.vp.SetYOffset(max(0, d.vp.YOffset-step))
			} else {
				d.vp.SetYOffset(min(d.vp.YOffset+step, max(0, d.vp.TotalLineCount()-d.vp.Height)))
			}
			return nil
		}
		return nil
	case tea.KeyMsg:
		switch t.Type {
		// Allow common paging and arrow keys to control the sidebar viewport so
		// terminals that synthesize Up/Down from the mouse wheel will scroll the
		// sidebar as expected.
		case tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd, tea.KeyUp, tea.KeyDown:
			var cmd tea.Cmd
			d.vp, cmd = d.vp.Update(msg)
			return cmd
		default:
			// ignore other keys to avoid conflicting with list navigation
			return nil
		}
	default:
		var cmd tea.Cmd
		d.vp, cmd = d.vp.Update(msg)
		return cmd
	}
}

func (d *DetailsModel) View() string {
	innerW := intMax(0, d.width-2)
	innerH := intMax(0, d.height-2)

	if innerW == 0 || innerH == 0 {
		return d.style.Width(intMax(0, d.width-2)).Height(intMax(0, d.height-2)).Render("")
	}

	// Build content lines when dirty or width changed relative to cache
	var b strings.Builder
	// Make the package name heading the same color as other headings
	titleStyle := lipgloss.NewStyle().Foreground(theme.Crust).Background(theme.Lavender).Bold(true).Padding(0, 1)
	headingStyle := lipgloss.NewStyle().Foreground(theme.Crust).Background(theme.Lavender).Bold(true).Padding(0, 1)
	linkStyle := lipgloss.NewStyle().Foreground(theme.Blue)
	mutedStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	sep := mutedStyle.Render(strings.Repeat("─", intMax(0, innerW)))

	wrap := lipgloss.NewStyle().Width(innerW).MaxWidth(innerW)

	if d.title != "" {
		b.WriteString(wrap.Render(titleStyle.Render(d.title)))
		b.WriteString("\n\n")
	}
	if d.stats != "" {
		b.WriteString(wrap.Render(d.stats))
		// If we have downloads series, render chart with extra spacing below the info
		if len(d.dlValues) > 1 {
			// Add an extra blank line between stats and the chart
			b.WriteString("\n")
			// Time series chart width fits innerW, height 6 for readability within sidebar
			if d.dlRendered == "" {
				width := innerW
				if width < 8 {
					width = intMax(1, innerW)
				}
				h := 6
				// Use a compact Y label formatter (e.g., 1.2k, 3.4M)
				yFmt := func(i int, v float64) string {
					av := math.Abs(v)
					switch {
					case av >= 1_000_000_000:
						return fmt.Sprintf("%.1fB", v/1_000_000_000)
					case av >= 1_000_000:
						return fmt.Sprintf("%.1fM", v/1_000_000)
					case av >= 1_000:
						return fmt.Sprintf("%.0fk", v/1_000)
					default:
						return fmt.Sprintf("%.0f", v)
					}
				}
				var lastYear string
				xFmt := func(i int, v float64) string {
					// Reset yearly state at the start of a draw pass
					if i == 0 {
						lastYear = ""
					}
					t := time.Unix(int64(v), 0).UTC()
					y := t.Format("'06")
					md := t.Format("02/01") // DD/MM
					if y != lastYear {
						lastYear = y
						return y + " " + md
					}
					return md
				}
				axisStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
				labelStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
				graphStyle := lipgloss.NewStyle().Foreground(theme.Blue)
				lc := timeserieslinechart.New(
					width, h,
					timeserieslinechart.WithYLabelFormatter(yFmt),
					timeserieslinechart.WithXLabelFormatter(xFmt),
					timeserieslinechart.WithAxesStyles(axisStyle, labelStyle),
					timeserieslinechart.WithStyle(graphStyle),
				)
				// If we have timestamps, use them; else, synthesize evenly spaced days
				if len(d.dlTimes) == len(d.dlValues) && len(d.dlTimes) > 0 {
					for i := range d.dlValues {
						lc.Push(timeserieslinechart.TimePoint{Time: d.dlTimes[i], Value: d.dlValues[i]})
					}
				} else {
					// fallback: create pseudo-dates one week apart ending today
					end := time.Now()
					start := end.AddDate(0, 0, -7*(len(d.dlValues)-1))
					for i, v := range d.dlValues {
						t := start.AddDate(0, 0, 7*i)
						lc.Push(timeserieslinechart.TimePoint{Time: t, Value: v})
					}
				}
				// Plot-area background: shade only the graphing columns across the time range
				// Use a higher-contrast surface for better visibility.
				bgStyle := lipgloss.NewStyle().Background(theme.Surface0)
				var minT, maxT time.Time
				if len(d.dlTimes) == len(d.dlValues) && len(d.dlTimes) > 0 {
					minT, maxT = d.dlTimes[0], d.dlTimes[0]
					for _, tt := range d.dlTimes {
						if tt.Before(minT) {
							minT = tt
						}
						if tt.After(maxT) {
							maxT = tt
						}
					}
				} else if len(d.dlValues) > 0 {
					maxT = time.Now()
					minT = maxT.AddDate(0, 0, -7*(len(d.dlValues)-1))
				}
				if !minT.IsZero() && !maxT.IsZero() {
					// Align to UTC midnight boundaries for consistent column mapping
					toMidnight := func(t time.Time) time.Time {
						u := t.UTC()
						y, m, d := u.Date()
						return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
					}
					minMid := toMidnight(minT)
					// include the last day fully by extending one more day beyond the max midnight
					maxMid := toMidnight(maxT).AddDate(0, 0, 1)
					// Ensure the chart uses the same visible window we are shading
					lc.SetTimeRange(minMid, maxMid)
					lc.SetViewTimeRange(minMid, maxMid)
					for t := minMid; t.Before(maxMid); t = t.AddDate(0, 0, 1) {
						lc.SetColumnBackgroundStyle(t, bgStyle)
					}
				}

				// Use braille for smoother lines in tight areas
				lc.DrawBraille()
				d.dlRendered = lc.View()
			}
			b.WriteString("\n")
			b.WriteString(d.dlRendered)
		}
		// One extra blank line under the graph for padding
		b.WriteString("\n\n")
		b.WriteString(sep)
		b.WriteString("\n")
	}
	if d.description != "" {
		// Section label
		descLabel := headingStyle.Render("Description")
		b.WriteString(wrap.Render(descLabel))
		// add a blank line between the heading and the text
		b.WriteString("\n\n")
		// Inline style for code and links, then wrap
		styledDesc := styleDescription(d.description)
		b.WriteString(wrap.Render(styledDesc))
		b.WriteString("\n\n")
	}
	// Links section with truncation and aligned icons only (no text labels)
	labelW := 8 // space for [home] + space
	linkW := intMax(8, innerW-labelW)
	row := func(icon, url string) {
		if url == "" {
			return
		}
		// icon + single trailing space (no left/half padding), clickable
		widthStyle := lipgloss.NewStyle().Width(labelW)
		cell := widthStyle.Render(icon + " ")
		lbl := osc8(url, cell)
		// display text without scheme while keeping actual URL intact
		disp := shortenLinkDisplay(url, linkW)
		val := osc8(url, linkStyle.Render(disp))
		b.WriteString(lbl)
		b.WriteString(val)
		b.WriteString("\n")
	}
	// Only normalize repository links; keep homepage/npm as-is
	repoURL := ensureScheme(normalizeURL(d.repository))
	homeURL := ensureScheme(d.homepage)
	npmURL := ensureScheme(d.npmLink)
	// Icons without backgrounds; try alternative glyphs that appear larger
	// repo: GitHub logo (Font Awesome) in white
	// Fancy ASCII word-icons
	repoIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render("[repo]")
	homeIcon := lipgloss.NewStyle().Foreground(theme.Blue).Render("[home]")
	npmIcon := lipgloss.NewStyle().Foreground(theme.Red).Render("[npm]")
	hasLinks := repoURL != "" || homeURL != "" || npmURL != ""
	if hasLinks {
		if b.Len() > 0 {
			b.WriteString(sep)
			b.WriteString("\n")
		}
		// Heading for links
		linksLabel := headingStyle.Render("Links")
		b.WriteString(wrap.Render(linksLabel))
		b.WriteString("\n\n")
		row(repoIcon, repoURL)
		row(homeIcon, homeURL)
		row(npmIcon, npmURL)
	}
	// Update viewport content only when changed to preserve scroll offset
	newContent := b.String()
	if d.dirty || newContent != d.content {
		d.content = newContent
		d.vp.SetContent(d.content)
		d.dirty = false
		if d.resetTop {
			d.vp.GotoTop()
			d.resetTop = false
		}
	}
	// Render the viewport inside the bordered container
	body := lipgloss.Place(innerW, innerH, lipgloss.Left, lipgloss.Top, d.vp.View())
	return d.style.Width(innerW).Height(innerH).Render(body)
}

// styleDescription applies lightweight inline styling to description text:
// - `code` spans get a subtle background
// - http(s) URLs become clickable and blue
func styleDescription(s string) string {
	if s == "" {
		return s
	}
	codeStyle := lipgloss.NewStyle().Foreground(theme.Text).Background(theme.Surface1)
	linkStyle := lipgloss.NewStyle().Foreground(theme.Blue)
	// Inline code: `code`
	reCode := regexp.MustCompile("`[^`]+`")
	s = reCode.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(m, "`"), "`")
		return codeStyle.Render(inner)
	})
	// URLs: http(s)://...
	reURL := regexp.MustCompile(`https?://[^\s)]+`)
	s = reURL.ReplaceAllStringFunc(s, func(u string) string {
		return osc8(u, linkStyle.Render(u))
	})
	return s
}

// shortenLinkDisplay returns a display-friendly URL that strips http(s) scheme
// while middle-truncating to fit maxW. The underlying URL should still include
// the scheme when used in osc8 for clickability.
func shortenLinkDisplay(url string, maxW int) string {
	if maxW <= 0 || url == "" {
		return ""
	}
	// Remove scheme only for display
	disp := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	// Trim trailing slash for neatness
	disp = strings.TrimRight(disp, "/")
	// If it already fits, return as-is
	if len(disp) <= maxW {
		return disp
	}
	// Middle truncate
	if maxW <= 1 {
		return "…"
	}
	left := maxW / 2
	right := maxW - left - 1 // for ellipsis
	if right < 1 {
		right = 1
		if left > 1 {
			left--
		}
	}
	if left+right+1 > len(disp) {
		if len(disp) > maxW {
			disp = disp[:maxW]
		}
		return disp
	}
	return disp[:left] + "…" + disp[len(disp)-right:]
}

//

// shortenLink normalizes and middle-truncates a link to fit maxW cells.
//

// ensureScheme adds https:// to URLs that lack http(s) scheme
func ensureScheme(s string) string {
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	return "https://" + s
}

// osc8 wraps a label with an OSC 8 hyperlink escape sequence.
// See: https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
func osc8(url, label string) string {
	if url == "" || label == "" {
		return label
	}
	const esc = "\x1b"
	// OSC 8 ; params ; URI ST  label OSC 8 ; ; ST
	return esc + "]8;;" + url + esc + "\\" + label + esc + "]8;;" + esc + "\\"
}

// normalizeURL attempts to coerce various npm repo/home link formats into
// a clean https URL for display (and potential copying).
func normalizeURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// strip git+ prefix
	s = strings.TrimPrefix(s, "git+")
	// git protocol -> https
	if strings.HasPrefix(s, "git://") {
		s = "https://" + strings.TrimPrefix(s, "git://")
	}
	// ssh style: git@host:user/repo(.git)
	if strings.HasPrefix(s, "git@") {
		rest := strings.TrimPrefix(s, "git@")
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			host := parts[0]
			path := parts[1]
			path = strings.TrimSuffix(path, ".git")
			s = "https://" + host + "/" + path
		}
	}
	// add https for bare domains commonly seen
	if strings.HasPrefix(s, "github.com/") || strings.HasPrefix(s, "gitlab.com/") || strings.HasPrefix(s, "bitbucket.org/") {
		s = "https://" + s
	}
	// remove trailing .git for display cleanliness
	s = strings.TrimSuffix(s, ".git")
	return s
}
