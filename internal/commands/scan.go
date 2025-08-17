package commands

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// ScanDepsMsg is emitted after scanning package.json for installed deps
type ScanDepsMsg struct {
	Installed map[string]bool
	Path      string
	Err       error
}

// ScanInstalledDeps walks up from CWD to find a package.json and returns a set
// of dependency names from dependencies/devDependencies/optionalDependencies.
func ScanInstalledDeps() tea.Cmd {
	return func() tea.Msg {
		cwd, _ := os.Getwd()
		pkgPath := findPackageJSON(cwd)
		if pkgPath == "" {
			return ScanDepsMsg{Installed: map[string]bool{}}
		}
		b, err := os.ReadFile(pkgPath)
		if err != nil {
			return ScanDepsMsg{Installed: map[string]bool{}, Path: pkgPath, Err: err}
		}
		// minimal struct for the sections we care about
		var data struct {
			Dependencies         map[string]string `json:"dependencies"`
			DevDependencies      map[string]string `json:"devDependencies"`
			OptionalDependencies map[string]string `json:"optionalDependencies"`
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return ScanDepsMsg{Installed: map[string]bool{}, Path: pkgPath, Err: err}
		}
		set := map[string]bool{}
		for k := range data.Dependencies {
			set[k] = true
		}
		for k := range data.DevDependencies {
			set[k] = true
		}
		for k := range data.OptionalDependencies {
			set[k] = true
		}
		return ScanDepsMsg{Installed: set, Path: pkgPath}
	}
}

// findPackageJSON searches up the directory tree for a package.json file.
func findPackageJSON(start string) string {
	dir := start
	for dir != "" {
		p := filepath.Join(dir, "package.json")
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
