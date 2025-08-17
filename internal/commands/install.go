package commands

import (
	"context"
	"log"
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

func InstallNPM(pkg string, dev bool) tea.Cmd {
	return func() tea.Msg {
		if pkg == "" {
			return NpmInstallMsg{Package: pkg, Dev: dev, Err: nil}
		}
		args := []string{"install"}
		if dev {
			args = append(args, "--save-dev")
		}
		args = append(args, pkg)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "npm", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("npm install failed for %s (dev=%v): %v\n%s", pkg, dev, err, string(out))
			return NpmInstallMsg{Package: pkg, Dev: dev, Output: string(out), Err: err}
		}
		return NpmInstallMsg{Package: pkg, Dev: dev, Output: string(out), Err: nil}
	}
}
