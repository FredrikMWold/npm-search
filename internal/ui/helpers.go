package ui

import (
	"fmt"
	"math"
)

// UI formatting and layout helpers

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
	// keep at least 22 cols for sidebar, then add 12 extra cols for readability
	side := int(math.Max(22, math.Round(float64(total)*0.32))) + 12
	// ensure list has room
	if side > total-20 {
		side = total - 20
	}
	if side < 0 {
		side = 0
	}
	return total - side, side
}
