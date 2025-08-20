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
	Wanted    map[string]string
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
			return ScanDepsMsg{Installed: map[string]bool{}, Wanted: map[string]string{}}
		}
		// Base directory of the project (where package.json lives)
		baseDir := filepath.Dir(pkgPath)
		b, err := os.ReadFile(pkgPath)
		if err != nil {
			return ScanDepsMsg{Installed: map[string]bool{}, Wanted: map[string]string{}, Path: pkgPath, Err: err}
		}
		// minimal struct for the sections we care about
		var data struct {
			Dependencies         map[string]string `json:"dependencies"`
			DevDependencies      map[string]string `json:"devDependencies"`
			OptionalDependencies map[string]string `json:"optionalDependencies"`
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return ScanDepsMsg{Installed: map[string]bool{}, Wanted: map[string]string{}, Path: pkgPath, Err: err}
		}
		set := map[string]bool{}
		wanted := map[string]string{}
		for k, v := range data.Dependencies {
			// mark installed only if present in node_modules
			set[k] = isPkgInstalled(baseDir, k)
			wanted[k] = v
		}
		for k, v := range data.DevDependencies {
			set[k] = isPkgInstalled(baseDir, k)
			wanted[k] = v
		}
		for k, v := range data.OptionalDependencies {
			set[k] = isPkgInstalled(baseDir, k)
			wanted[k] = v
		}
		return ScanDepsMsg{Installed: set, Wanted: wanted, Path: pkgPath}
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
