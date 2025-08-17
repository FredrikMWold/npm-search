package list

import (
	"fmt"
	"io"

	bblist "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredrikmwold/npm-search/internal/ui/theme"
)

// delegate customizes row rendering to show a spinner for installing items.
type delegate struct {
	bblist.DefaultDelegate
	installing map[string]bool
	installed  map[string]bool
	wanted     map[string]string // manifest (wanted) versions by name
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
	suffix := ""
	if d.installing != nil && d.installing[it.Name()] {
		// show spinner after the name while installing
		suffix = " " + d.frame
	} else if d.installed != nil && d.installed[it.Name()] {
		// If installed, check if an update is available compared to latest
		if d.wanted != nil {
			if want, ok := d.wanted[it.Name()]; ok && updateRecommended(it.latest, want) {
				// show 'Outdated old -> new'
				label := " Outdated"
				if oldV, newV, okp := updatePath(it.latest, want); okp {
					label = fmt.Sprintf(" Outdated %s -> %s", oldV, newV)
				}
				warn := lipgloss.NewStyle().Foreground(theme.Peach).Bold(true).Render(label)
				suffix = " " + warn
			} else {
				installed := lipgloss.NewStyle().Foreground(theme.Green).Render("✔ Installed")
				suffix = " " + installed
			}
		} else {
			installed := lipgloss.NewStyle().Foreground(theme.Green).Render("✔ Installed")
			suffix = " " + installed
		}
	}
	// Wrap the item to override Title() with spinner prefix/suffix while preserving
	// default height/formatting.
	wi := wrappedItem{item: it, pre: prefix, suf: suffix}
	d.DefaultDelegate.Render(w, m, index, wi)
}

// wrappedItem decorates an item with a prefix/suffix for the Title while delegating
// other methods to the embedded item.
type wrappedItem struct {
	item
	pre string
	suf string
}

func (w wrappedItem) Title() string { return w.pre + w.item.Title() + w.suf }
