package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-search/internal/ui/theme"
)

// DetailsModel renders a sidebar with info about a selected package.
type DetailsModel struct {
	width  int
	height int
	style  lipgloss.Style

	// content
	title       string
	description string
	stats       string
	homepage    string
	repository  string
	npmLink     string
}

func NewDetails() *DetailsModel {
	st := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderUnfocused).
		Foreground(theme.Text)
	return &DetailsModel{style: st}
}

func (d *DetailsModel) Init() tea.Cmd { return nil }

func (d *DetailsModel) SetSize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 0 {
		h = 0
	}
	d.width, d.height = w, h
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
	d.title = title
	d.description = desc
	d.homepage = homepage
	d.repository = repo
	d.npmLink = npmLink
}

// SetStats sets the one-line stats string (version/downloads/license/author)
func (d *DetailsModel) SetStats(s string) { d.stats = s }

func (d *DetailsModel) Update(msg tea.Msg) tea.Cmd { return nil }

func (d *DetailsModel) View() string {
	innerW := maxInt(0, d.width-2)
	innerH := maxInt(0, d.height-2)

	if innerW == 0 || innerH == 0 {
		return d.style.Width(maxInt(0, d.width-2)).Height(maxInt(0, d.height-2)).Render("")
	}

	// Build content lines
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	linkStyle := lipgloss.NewStyle().Foreground(theme.Blue)
	mutedStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	sep := mutedStyle.Render(strings.Repeat("─", maxInt(0, innerW)))

	wrap := lipgloss.NewStyle().Width(innerW).MaxWidth(innerW)

	if d.title != "" {
		b.WriteString(wrap.Render(titleStyle.Render(d.title)))
		b.WriteString("\n\n")
	}
	if d.stats != "" {
		b.WriteString(wrap.Render(d.stats))
		b.WriteString("\n")
		b.WriteString(sep)
		b.WriteString("\n")
	}
	if d.description != "" {
		b.WriteString(wrap.Render(d.description))
		b.WriteString("\n\n")
	}
	// Links section with truncation and aligned labels
	linkCount := 0
	labelW := 6 // width for labels like "repo:" "home:" "npm:"
	linkW := maxInt(8, innerW-labelW)
	row := func(label, url string) {
		if url == "" {
			return
		}
		if linkCount == 0 {
			// first link: add subtle separator if there was preceding text
			if b.Len() > 0 {
				b.WriteString(sep)
				b.WriteString("\n")
			}
		}
		linkCount++
		lbl := labelStyle.Width(labelW).Render(label)
		disp := shortenLink(url, linkW)
		val := osc8(url, linkStyle.Render(disp))
		b.WriteString(lbl)
		b.WriteString(val)
		b.WriteString("\n")
	}
	// Only normalize repository links; keep homepage/npm as-is
	repoURL := ensureScheme(normalizeURL(d.repository))
	homeURL := ensureScheme(d.homepage)
	npmURL := ensureScheme(d.npmLink)
	row("repo:", repoURL)
	row("home:", homeURL)
	row("npm:", npmURL)

	// Clamp by lines with ellipsis if overflow (avoid re-wrapping hyperlinks)
	lines := strings.Split(b.String(), "\n")
	if innerH > 0 && len(lines) > innerH {
		lines = lines[:innerH]
		// Add overflow indicator to the last visible line if there was overflow
		if innerH-1 >= 0 {
			lines[innerH-1] = strings.TrimRight(lines[innerH-1], " ") + " …"
		}
	}
	clamped := strings.Join(lines, "\n")
	content := lipgloss.Place(innerW, innerH, lipgloss.Left, lipgloss.Top, clamped)
	return d.style.Width(innerW).Height(innerH).Render(content)
}

// local max to avoid importing others
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// shortenLink normalizes and middle-truncates a link to fit maxW cells.
func shortenLink(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	// ensure scheme for clickability and preserve it at the start
	scheme := ""
	if strings.HasPrefix(s, "https://") {
		scheme = "https://"
	} else if strings.HasPrefix(s, "http://") {
		scheme = "http://"
	}
	rest := s
	if scheme != "" {
		rest = s[len(scheme):]
	}
	// trim trailing slash for cleanliness
	rest = strings.TrimRight(rest, "/")

	// If it already fits, return as-is (with scheme)
	if len(scheme)+len(rest) <= maxW {
		return scheme + rest
	}
	// If space is too small, try to at least return the scheme (or ellipsis)
	if maxW <= len(scheme)+1 {
		if maxW <= len(scheme) {
			// Even scheme doesn't fit fully; return truncated scheme
			return (scheme + rest)[:maxW]
		}
		// show scheme + ellipsis
		return scheme + "…"
	}
	// Truncate the rest in the middle to fit remaining width
	remain := maxW - len(scheme)
	if remain <= 1 {
		return scheme + "…"
	}
	// middle truncation for rest
	left := remain / 2
	right := remain - left - 1 // for ellipsis
	if right < 1 {
		right = 1
		if left > 1 {
			left--
		}
	}
	if left < 1 {
		left = 1
	}
	if left+right+1 > len(rest) {
		// shouldn't happen often, but guard anyway
		if len(rest) > remain {
			rest = rest[:remain]
		}
		return scheme + rest
	}
	return scheme + rest[:left] + "…" + rest[len(rest)-right:]
}

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
