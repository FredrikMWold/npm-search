package commands

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// GitHubReadmeMsg is emitted when fetching a README from GitHub completes.
type GitHubReadmeMsg struct {
	Repo    string // owner/repo
	Content string // decoded markdown
	Err     error
	Req     int // request sequence
}

// FetchGitHubReadme fetches the README for a GitHub repo URL and returns the markdown.
// Accepts repository URLs like:
// - https://github.com/owner/repo
// - git+https://github.com/owner/repo.git
// - git@github.com:owner/repo.git
func FetchGitHubReadmeWithReq(repoURL string, req int) tea.Cmd {
	return func() tea.Msg {
		owner, repo, err := parseGitHubRepo(repoURL)
		if err != nil {
			return GitHubReadmeMsg{Repo: "", Content: "", Err: err, Req: req}
		}
		// Only use raw.githubusercontent.com fallbacks.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if md := tryRawFallbackCtx(ctx, owner, repo); md != "" {
			// Trim a potential BOM and leading whitespace/newlines to avoid ghost lines
			md = strings.TrimLeft(md, "\ufeff\n\r\t ")
			return GitHubReadmeMsg{Repo: owner + "/" + repo, Content: md, Req: req}
		}
		return GitHubReadmeMsg{Repo: owner + "/" + repo, Err: errors.New("could not fetch README"), Req: req}
	}
}

// Backwards-compatible wrapper without sequence
func FetchGitHubReadme(repoURL string) tea.Cmd { return FetchGitHubReadmeWithReq(repoURL, 0) }

// parseGitHubRepo extracts owner and repo from a variety of repo URL formats.
func parseGitHubRepo(s string) (owner, repo string, err error) {
	if s == "" {
		return "", "", errors.New("no repository URL")
	}
	// strip git+ prefix
	s = strings.TrimPrefix(s, "git+")
	// handle npm shorthand like github:owner/repo
	if strings.HasPrefix(s, "github:") {
		rest := strings.TrimPrefix(s, "github:")
		rest = strings.TrimSuffix(rest, ".git")
		parts := strings.Split(strings.TrimPrefix(rest, "/"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
		return "", "", errors.New("invalid github shorthand repo path")
	}
	// ssh: git@github.com:owner/repo(.git)
	if strings.HasPrefix(s, "git@github.com:") {
		rest := strings.TrimPrefix(s, "git@github.com:")
		rest = strings.TrimSuffix(rest, ".git")
		parts := strings.Split(rest, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
		return "", "", errors.New("invalid github ssh repo path")
	}
	// Try URL parsing
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}
	u, e := url.Parse(s)
	if e != nil {
		return "", "", e
	}
	host := strings.ToLower(u.Host)
	if host != "github.com" && !strings.HasSuffix(host, ".github.com") && !strings.HasSuffix(host, "githubusercontent.com") {
		return "", "", errors.New("repository is not a GitHub URL")
	}
	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", errors.New("invalid github repo path")
	}
	return parts[0], parts[1], nil
}

// fetchViaAPI queries the GitHub API for README content with base64 decode.
// (API path removed as requested)

// tryRawFallback attempts common raw README locations to bypass API rate limits.
// tryRawFallback remains as context-less helper if needed elsewhere (not used now)

// Context-aware variant used in the command flow
func tryRawFallbackCtx(ctx context.Context, owner, repo string) string {
	client := &http.Client{Timeout: 3 * time.Second}
	branches := []string{"main", "master"}
	names := []string{
		"README.md", "Readme.md", "readme.md",
		"README.MD", "README.markdown", "README.rst", "README.txt", "README",
	}
	for _, br := range branches {
		for _, nm := range names {
			u := "https://raw.githubusercontent.com/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/" + url.PathEscape(br) + "/" + nm
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			req.Header.Set("User-Agent", "npm-search (https://github.com/fredrikmwold/npm-search)")
			start := time.Now()
			if r, err := client.Do(req); err == nil && r != nil {
				if r.StatusCode == http.StatusOK {
					b, _ := io.ReadAll(r.Body)
					r.Body.Close()
					if len(b) > 0 {
						return string(b)
					} else {
						// empty body
					}
				} else {
					// Read a small snippet for diagnostics (discarded)
					snip, _ := io.ReadAll(io.LimitReader(r.Body, 256))
					r.Body.Close()
					_ = snip
				}
			} else if err != nil {
				_ = time.Since(start)
			}
			if ctx.Err() != nil {
				return ""
			}
		}
	}
	return ""
}
