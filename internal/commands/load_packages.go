package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// LoadProjectPackages requests metadata for all deps in the nearest package.json
// and returns them as a NpmSearchMsg (Query=""). This populates the initial list.
func LoadProjectPackages() tea.Cmd {
	return func() tea.Msg {
		// Find deps via existing scan
		cwd, _ := os.Getwd()
		// Reuse ScanInstalledDeps logic by calling directly
		// Instead of sending a message, we replicate the scan here for simplicity
		pkgPath := findPackageJSON(cwd)
		if pkgPath == "" {
			return NpmSearchMsg{Query: "", Result: NpmSearchResult{Objects: []NpmSearchObject{}}, Err: nil}
		}
		b, err := os.ReadFile(pkgPath)
		if err != nil {
			return NpmSearchMsg{Query: "", Result: NpmSearchResult{}, Err: err}
		}
		var data struct {
			Dependencies         map[string]string `json:"dependencies"`
			DevDependencies      map[string]string `json:"devDependencies"`
			OptionalDependencies map[string]string `json:"optionalDependencies"`
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return NpmSearchMsg{Query: "", Result: NpmSearchResult{}, Err: err}
		}
		// Gather names (unique)
		names := make([]string, 0, len(data.Dependencies)+len(data.DevDependencies)+len(data.OptionalDependencies))
		seen := map[string]struct{}{}
		for k := range data.Dependencies {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				names = append(names, k)
			}
		}
		for k := range data.DevDependencies {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				names = append(names, k)
			}
		}
		for k := range data.OptionalDependencies {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				names = append(names, k)
			}
		}
		if len(names) == 0 {
			return NpmSearchMsg{Query: "", Result: NpmSearchResult{Objects: []NpmSearchObject{}}, Err: nil}
		}
		client := &http.Client{Timeout: 8 * time.Second}
		type out struct {
			idx int
			obj NpmSearchObject
		}
		// pre-size slice
		result := make([]NpmSearchObject, len(names))
		sem := make(chan struct{}, 6)
		done := make(chan out, len(names))
		for i, nm := range names {
			i, nm := i, nm
			sem <- struct{}{}
			go func() {
				defer func() { <-sem }()
				// Try cache first
				if cached, ok := cacheGetPkg(nm); ok {
					done <- out{idx: i, obj: cached}
					return
				}
				// Fetch https://registry.npmjs.com/<name>
				metaURL := fmt.Sprintf("https://registry.npmjs.com/%s", url.PathEscape(nm))
				var obj NpmSearchObject
				if resp, err := client.Get(metaURL); err == nil && resp != nil {
					defer resp.Body.Close()
					// We only need latest dist-tags and metadata
					var raw map[string]any
					if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
						// Get latest version from dist-tags.latest
						latest := ""
						if dt, ok := raw["dist-tags"].(map[string]any); ok {
							if lv, ok := dt["latest"].(string); ok {
								latest = lv
							}
						}
						// Switch to versions[latest] for details; fallback to top-level if missing
						var ver map[string]any
						if vs, ok := raw["versions"].(map[string]any); ok && latest != "" {
							if v, ok := vs[latest].(map[string]any); ok {
								ver = v
							}
						}
						if ver == nil {
							ver = raw
						}
						// Build NpmPackage
						var pkg NpmPackage
						pkg.Name = nm
						pkg.Version = latest
						if d, ok := ver["description"].(string); ok {
							pkg.Description = d
						}
						if l, ok := ver["license"].(string); ok {
							pkg.License = l
						} else if lobj, ok := ver["license"].(map[string]any); ok {
							if t, ok := lobj["type"].(string); ok {
								pkg.License = t
							}
						}
						if a, ok := ver["author"].(map[string]any); ok {
							if n, ok := a["name"].(string); ok {
								pkg.Author = n
							}
						} else if as, ok := ver["author"].(string); ok {
							pkg.Author = as
						}
						// Links
						pkg.Links.NPM = fmt.Sprintf("https://www.npmjs.com/package/%s", nm)
						if lh, ok := ver["homepage"].(string); ok {
							pkg.Links.Homepage = lh
						}
						if rep, ok := ver["repository"].(map[string]any); ok {
							if u, ok := rep["url"].(string); ok {
								pkg.Links.Repository = u
							}
						} else if rs, ok := ver["repository"].(string); ok {
							pkg.Links.Repository = rs
						}
						// downloads last week
						dlURL := "https://api.npmjs.org/downloads/point/last-week/" + url.PathEscape(nm)
						if r2, e2 := client.Get(dlURL); e2 == nil && r2 != nil {
							defer r2.Body.Close()
							var dl downloadsPointResponse
							if err := json.NewDecoder(r2.Body).Decode(&dl); err == nil {
								pkg.DownloadsLastWeek = dl.Downloads
							}
						}
						obj = NpmSearchObject{Package: pkg}
					}
				}
				// Store in cache (even if empty, to avoid tight refetch loops on failures)
				cacheSetPkg(nm, obj)
				done <- out{idx: i, obj: obj}
			}()
		}
		for i := 0; i < len(names); i++ {
			o := <-done
			result[o.idx] = o.obj
		}
		return NpmSearchMsg{Query: "", Result: NpmSearchResult{Objects: result}}
	}
}
