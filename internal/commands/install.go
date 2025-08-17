package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type NpmInstallMsg struct {
	Package string
	Dev     bool
	Output  string
	Err     error
}

// PackageManager enumerates supported JS package managers
type PackageManager string

const (
	PMNPM  PackageManager = "npm"
	PMPNPM PackageManager = "pnpm"
	PMYarn PackageManager = "yarn"
	PMBun  PackageManager = "bun"
)

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

func InstallNPM(pkg string, dev bool) tea.Cmd {
	return func() tea.Msg {
		if pkg == "" {
			return NpmInstallMsg{Package: pkg, Dev: dev, Err: nil}
		}
		// Decide which package manager to use based on project files
		wd, _ := os.Getwd()
		pm := detectPackageManager(wd)

		var cmdName string
		var args []string
		switch pm {
		case PMPNPM:
			cmdName = "pnpm"
			args = []string{"add"}
			if dev {
				args = append(args, "--save-dev")
			}
			args = append(args, pkg)
		case PMYarn:
			cmdName = "yarn"
			args = []string{"add"}
			if dev {
				args = append(args, "-D")
			}
			args = append(args, pkg)
		case PMBun:
			cmdName = "bun"
			args = []string{"add"}
			if dev {
				args = append(args, "-d")
			}
			args = append(args, pkg)
		default: // npm
			cmdName = "npm"
			args = []string{"install"}
			if dev {
				args = append(args, "--save-dev")
			}
			args = append(args, pkg)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		// If a specific PM was detected but binary is missing, return an error instead
		if _, err := exec.LookPath(cmdName); err != nil {
			if pm != PMNPM { // only auto-use npm when npm was selected by detection
				e := fmt.Errorf("%s not found on PATH", cmdName)
				log.Printf("%v", e)
				return NpmInstallMsg{Package: pkg, Dev: dev, Output: "", Err: e}
			}
			// pm is npm; proceed
		}
		cmd := exec.CommandContext(ctx, cmdName, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("%s install failed for %s (dev=%v): %v\n%s", cmdName, pkg, dev, err, string(out))
			return NpmInstallMsg{Package: pkg, Dev: dev, Output: string(out), Err: err}
		}
		return NpmInstallMsg{Package: pkg, Dev: dev, Output: string(out), Err: nil}
	}
}
