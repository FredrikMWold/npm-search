package list

// version utilities used to decide update recommendations and display paths.

// updateRecommended returns true when the manifest's wanted spec does not
// equal the latest version string (ignoring leading ^/~ and 'v'). This keeps
// semantics simple: if package.json isn't explicitly at the latest, suggest update.
func updateRecommended(latest, wanted string) bool {
	if wanted == "" || latest == "" {
		return false
	}
	w := trimPrefixSet(wanted, "^~vV")
	l := trimPrefixSet(latest, "vV")
	// Extract the numeric base (digits and dots) at the start
	w = numericPrefix(w)
	l = numericPrefix(l)
	if w == "" || l == "" {
		return false
	}
	return l != w
}

func trimPrefixSet(s, set string) string {
	for len(s) > 0 {
		matched := false
		for i := 0; i < len(set); i++ {
			if s[0] == set[i] {
				s = s[1:]
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}
	return s
}

func numericPrefix(s string) string {
	i := 0
	for i < len(s) {
		c := s[i]
		if (c < '0' || c > '9') && c != '.' {
			break
		}
		i++
	}
	return s[:i]
}

// updatePath returns old and new versions for display (old->new) given latest and wanted specs.
// It trims ^/~ and leading v, then extracts numeric prefixes.
func updatePath(latest, wanted string) (string, string, bool) {
	if latest == "" || wanted == "" {
		return "", "", false
	}
	old := numericPrefix(trimPrefixSet(wanted, "^~vV"))
	neu := numericPrefix(trimPrefixSet(latest, "vV"))
	if old == "" || neu == "" || old == neu {
		return "", "", false
	}
	return old, neu, true
}
