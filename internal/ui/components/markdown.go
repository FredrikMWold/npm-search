package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-tui/internal/ui/theme"
)

// MarkdownViewer renders markdown in a scrollable viewport filling the given size.
type MarkdownViewer struct {
	width  int
	height int
	vp     viewport.Model
	md     string // raw markdown
	html   string // rendered ansi
	// cached renderer for current wrap width
	renderer *glamour.TermRenderer
	rwidth   int
}

func NewMarkdownViewer() *MarkdownViewer {
	vp := viewport.New(0, 0)
	// Styled similarly to other panes for consistency
	vp.Style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(theme.BorderFocused).Foreground(theme.Text)
	mv := &MarkdownViewer{vp: vp}
	return mv
}

func (m *MarkdownViewer) Init() tea.Cmd { return nil }

func (m *MarkdownViewer) SetSize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 0 {
		h = 0
	}
	m.width, m.height = w, h
	// Account for style frame (borders/padding) so the total area == w x h
	fw, fh := m.vp.Style.GetFrameSize()
	// Add a tiny fudge to compensate for terminal/font rounding issues
	innerW := intMax(0, w-fw+2)
	innerH := intMax(0, h-fh+2)
	// Set viewport content area to inner dimensions
	m.vp.Width = innerW
	m.vp.Height = innerH
	// Set style to the outer allocated size to guarantee fill
	m.vp.Style = m.vp.Style.Width(w).Height(h)
	// Re-render with new wrap width
	if m.md != "" && m.html != "" {
		// re-wrap existing rendered content by re-rendering
		m.render()
	} else if m.html != "" {
		// just update viewport size on plain content
		m.vp.SetContent(m.html)
	}
	// No dynamic height calibration here; rely on computed inner height
}

// SetMarkdown sets the markdown content and renders it.
func (m *MarkdownViewer) SetMarkdown(md string) {
	m.md = md
	// Show plain markdown immediately to avoid perceived lag; glamour async may update later
	m.vp.SetContent(md)
	m.vp.GotoTop()
	m.render()
}

// SetPlain sets a plain, already-rendered message without glamour. Instant.
func (m *MarkdownViewer) SetPlain(s string) {
	m.md = ""
	m.html = s
	m.vp.SetContent(s)
	m.vp.GotoTop()
}

func (m *MarkdownViewer) render() {
	// viewport width already accounts for borders
	w := m.vp.Width
	if w <= 0 {
		w = intMax(0, m.width-2)
	}
	if w <= 0 {
		w = 80
	}
	// Ensure cached renderer for current width
	var err error
	if m.renderer == nil || m.rwidth != w {
		m.renderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(w),
		)
		m.rwidth = w
	}
	if err != nil || m.renderer == nil {
		// fallback: no formatting
		m.html = m.md
	} else {
		out, e := m.renderer.Render(m.md)
		if e != nil {
			m.html = m.md
		} else {
			// Ensure no leading newline that adds extra spacing at top
			m.html = strings.TrimLeft(out, "\n")
		}
	}
	m.vp.SetContent(m.html)
	// Do not recalibrate height here; stick to SetSize-provided height
}

// MarkdownRenderedMsg is emitted after asynchronous glamour rendering.
type MarkdownRenderedMsg struct {
	Content string
	Err     error
	Seq     int
}

// RenderAsync renders markdown to ANSI off the UI thread and returns a message.
func (m *MarkdownViewer) RenderAsync(md string, seq int) tea.Cmd {
	w := m.vp.Width
	if w <= 0 {
		w = intMax(0, m.width-2)
	}
	// Snapshot a renderer to avoid recreating per render when width is unchanged
	var r *glamour.TermRenderer
	if m.renderer != nil && m.rwidth == w {
		r = m.renderer
	}
	return func() tea.Msg {
		var out string
		var e error
		if r == nil {
			// Create a local renderer if none cached for this width
			var err error
			r, err = glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(w),
			)
			if err != nil {
				return MarkdownRenderedMsg{Content: md, Err: err, Seq: seq}
			}
		}
		// Render full content; avoid early preview return that prevents final render
		out, e = r.Render(md)
		if e != nil {
			return MarkdownRenderedMsg{Content: md, Err: e, Seq: seq}
		}
		return MarkdownRenderedMsg{Content: strings.TrimLeft(out, "\n"), Seq: seq}
	}
}

func (m *MarkdownViewer) Update(msg tea.Msg) tea.Cmd {
	switch t := msg.(type) {
	case MarkdownRenderedMsg:
		// set rendered content
		m.html = t.Content
		m.vp.SetContent(m.html)
		m.vp.GotoTop()
		// fall through to allow viewport to process message & refresh
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return cmd
}

func (m *MarkdownViewer) View() string {
	// Return the viewport's own view; its style/border already matches SetSize.
	return m.vp.View()
}

// countLines returns the number of lines in a rendered string.
//
