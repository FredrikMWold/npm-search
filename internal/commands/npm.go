package commands

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// NpmSearchMsg is emitted when an npm search completes
type NpmSearchMsg struct {
	Query  string
	Result NpmSearchResult
	Err    error
}

// NpmSearchResult models the subset of the npm search payload we care about
type NpmSearchResult struct {
	Objects []struct {
		Package struct {
			Name        string   `json:"name"`
			Version     string   `json:"version"`
			Description string   `json:"description"`
			Keywords    []string `json:"keywords"`
			Date        string   `json:"date"`
			Links       struct {
				NPM        string `json:"npm"`
				Homepage   string `json:"homepage"`
				Repository string `json:"repository"`
				Bugs       string `json:"bugs"`
			} `json:"links"`
			Publisher struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"publisher"`
			// Augmented field: not from the API, we populate it after fetching downloads
			DownloadsLastWeek int `json:"-"`
			// Augmented field: latest license string
			License string `json:"-"`
			// Augmented field: author (from latest metadata or fallback to publisher username)
			Author string `json:"-"`
		} `json:"package"`
		Score struct {
			Final  float64 `json:"final"`
			Detail struct {
				Quality     float64 `json:"quality"`
				Popularity  float64 `json:"popularity"`
				Maintenance float64 `json:"maintenance"`
			} `json:"detail"`
		} `json:"score"`
		SearchScore float64 `json:"searchScore"`
	} `json:"objects"`
	Total int    `json:"total"`
	Time  string `json:"time"`
}

// downloadsPointResponse is the payload from the per-package downloads API
type downloadsPointResponse struct {
	Downloads int    `json:"downloads"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Package   string `json:"package"`
}

// SearchNPM issues an HTTP GET to the npm search API asynchronously and logs
// the JSON response to the configured logger (file).
func SearchNPM(query string) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return NpmSearchMsg{Query: query, Err: nil, Result: NpmSearchResult{}}
		}
		u, _ := url.Parse("http://registry.npmjs.com/-/v1/search")
		q := u.Query()
		q.Set("text", query)
		q.Set("size", "10")
		u.RawQuery = q.Encode()

		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Get(u.String())
		if err != nil {
			return NpmSearchMsg{Query: query, Err: err}
		}
		defer resp.Body.Close()

		var parsed NpmSearchResult
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&parsed); err != nil {
			return NpmSearchMsg{Query: query, Err: err}
		}

		// For each package, fetch weekly downloads and augment the result.
		type result struct {
			idx       int
			downloads int
			license   string
			author    string
		}
		sem := make(chan struct{}, 5)                  // limit concurrency
		done := make(chan result, len(parsed.Objects)) // buffer to avoid deadlock before we start reading
		for i := range parsed.Objects {
			name := parsed.Objects[i].Package.Name
			sem <- struct{}{}
			go func(idx int, pkg string) {
				defer func() { <-sem }()
				// Use same client with short timeout
				reqURL := "https://api.npmjs.org/downloads/point/last-week/" + url.PathEscape(pkg)
				r, e := client.Get(reqURL)
				if e != nil {
					// still try to fetch license even if downloads failed
				}
				downloads := 0
				if r != nil {
					defer r.Body.Close()
					var dl downloadsPointResponse
					if err := json.NewDecoder(r.Body).Decode(&dl); err == nil {
						downloads = dl.Downloads
					}
				}
				// Fetch latest metadata for license
				lic := ""
				author := ""
				latestURL := "https://registry.npmjs.com/" + url.PathEscape(pkg) + "/latest"
				if r2, e2 := client.Get(latestURL); e2 == nil {
					defer r2.Body.Close()
					var raw map[string]any
					if err := json.NewDecoder(r2.Body).Decode(&raw); err == nil {
						if lv, ok := raw["license"]; ok {
							switch t := lv.(type) {
							case string:
								lic = t
							case map[string]any:
								if tt, ok := t["type"].(string); ok {
									lic = tt
								}
							}
						}
						if av, ok := raw["author"]; ok {
							switch t := av.(type) {
							case string:
								author = t
							case map[string]any:
								if nm, ok := t["name"].(string); ok {
									author = nm
								}
							}
						}
					}
				}
				done <- result{idx: idx, downloads: downloads, license: lic, author: author}
			}(i, name)
		}
		// Collect results
		for i := 0; i < len(parsed.Objects); i++ {
			res := <-done
			parsed.Objects[res.idx].Package.DownloadsLastWeek = res.downloads
			parsed.Objects[res.idx].Package.License = res.license
			parsed.Objects[res.idx].Package.Author = res.author
		}

		// Fill author from publisher username if author unavailable
		for i := range parsed.Objects {
			if parsed.Objects[i].Package.Author == "" && parsed.Objects[i].Package.Publisher.Username != "" {
				parsed.Objects[i].Package.Author = parsed.Objects[i].Package.Publisher.Username
			}
		}

		return NpmSearchMsg{Query: query, Result: parsed}
	}
}
