package commands

import (
	"os"
	"path/filepath"
)

// detect.go: helper routines for finding package manager/installed packages

// detectPackageManager inspects lockfiles to decide which package manager to use.
// Defaults to npm if none detected.
func detectPackageManager(cwd string) PackageManager {
	if cwd == "" {
		if w, err := os.Getwd(); err == nil {
			cwd = w
		}
	}
	if cwd != "" {
		tryFiles := []struct {
			file string
			pm   PackageManager
		}{
			{"pnpm-lock.yaml", PMPNPM},
			{"bun.lockb", PMBun},
			{"yarn.lock", PMYarn},
			{"package-lock.json", PMNPM},
		}
		dir := cwd
		for {
			for _, t := range tryFiles {
				if _, err := os.Stat(filepath.Join(dir, t.file)); err == nil {
					return t.pm
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir { // reached filesystem root
				break
			}
			dir = parent
		}
	}

	// Fallback
	return PMNPM
}

// isPkgInstalled checks if node_modules/<pkg>/package.json exists (supports scopes).
func isPkgInstalled(cwd, name string) bool {
	base := cwd
	if base == "" {
		if w, err := os.Getwd(); err == nil {
			base = w
		}
	}
	if base == "" {
		return false
	}
	nm := filepath.Join(base, "node_modules", name, "package.json")
	if _, err := os.Stat(nm); err == nil {
		return true
	}
	return false
}
