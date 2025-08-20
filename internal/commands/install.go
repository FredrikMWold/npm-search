package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

//

// installMutex is a simple semaphore (buffered channel) to serialize
// install/update operations. Running multiple package manager commands
// concurrently in the same project often causes lock/contention issues
// and leads to hangs or inconsistent state. Limiting to 1 ensures
// predictable behavior when users trigger several actions quickly.
var installMutex = make(chan struct{}, 1)

func InstallNPM(pkg string, dev bool) tea.Cmd {
	return func() tea.Msg {
		if pkg == "" {
			return NpmInstallMsg{Package: pkg, Dev: dev, Err: nil}
		}
		// Serialize install/update operations
		installMutex <- struct{}{}
		defer func() { <-installMutex }()

		// Decide which package manager to use based on project files
		wd, _ := os.Getwd()
		pm := detectPackageManager(wd)

		// Re-check installed state within the critical section to avoid
		// stale decisions when multiple actions are queued.
		installed := isPkgInstalled(wd, pkg)

		var cmdName string
		var args []string
		switch pm {
		case PMPNPM:
			cmdName = "pnpm"
			if installed {
				args = []string{"up"}
			} else {
				args = []string{"add"}
				if dev {
					args = append(args, "--save-dev")
				}
			}
			args = append(args, pkg)
		case PMYarn:
			cmdName = "yarn"
			if installed {
				args = []string{"upgrade", "--latest", pkg}
			} else {
				args = []string{"add"}
				if dev {
					args = append(args, "-D")
				}
				args = append(args, pkg)
			}
		case PMBun:
			cmdName = "bun"
			if installed {
				args = []string{"upgrade", pkg}
			} else {
				args = []string{"add"}
				if dev {
					args = append(args, "-d")
				}
				args = append(args, pkg)
			}
		default: // npm
			cmdName = "npm"
			// Use install <pkg>@latest for both installed and new; add --save-dev for dev.
			args = []string{"install"}
			if dev && !installed { // keep dev flag only for fresh installs
				args = append(args, "--save-dev")
			}
			args = append(args, pkg+"@latest")
		}

		// Timeout per actual execution; starts after we acquired the mutex.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// If a specific PM was detected but binary is missing, return an error instead
		if _, err := exec.LookPath(cmdName); err != nil {
			if pm != PMNPM { // only auto-use npm when npm was selected by detection
				e := fmt.Errorf("%s not found on PATH", cmdName)
				return NpmInstallMsg{Package: pkg, Dev: dev, Output: "", Err: e}
			}
			// pm is npm; proceed
		}
		cmd := exec.CommandContext(ctx, cmdName, args...)
		// Ensure we run in the detected working directory
		if wd != "" {
			cmd.Dir = wd
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return NpmInstallMsg{Package: pkg, Dev: dev, Output: string(out), Err: err}
		}
		return NpmInstallMsg{Package: pkg, Dev: dev, Output: string(out), Err: nil}
	}
}

//
