package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// SearchNPM issues an HTTP GET to the npm search API asynchronously and
// returns the parsed results as a tea.Msg.
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

// FetchDownloadsRange fetches downloads per day for a given package over the last N days
// using the npm downloads range API and returns an NpmDownloadsRangeMsg.
// days must be >= 1. Values are ordered oldest..newest.
func FetchDownloadsRange(pkg string, days int) tea.Cmd {
	return func() tea.Msg {
		if pkg == "" || days < 1 {
			return NpmDownloadsRangeMsg{Package: pkg, Values: nil}
		}
		// Check cache first
		key := pkg + "|" + strconv.Itoa(days)
		if vals, ok := cacheGetDLRange(key); ok {
			return NpmDownloadsRangeMsg{Package: pkg, Values: vals}
		}
		// Compute date window: inclusive start:end, YYYY-MM-DD
		end := time.Now().AddDate(0, 0, -1) // yesterday to avoid partial current day
		start := end.AddDate(0, 0, -(days - 1))
		startStr := start.Format("2006-01-02")
		endStr := end.Format("2006-01-02")

		u := "https://api.npmjs.org/downloads/range/" + startStr + ":" + endStr + "/" + url.PathEscape(pkg)
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Get(u)
		if err != nil {
			return NpmDownloadsRangeMsg{Package: pkg, Err: err}
		}
		defer resp.Body.Close()
		var parsed rangeDownloadsResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return NpmDownloadsRangeMsg{Package: pkg, Err: err}
		}
		// Aggregate to weekly (ISO week) averages to reduce columns while showing trend
		vals := make([]float64, 0, 64)
		pts := make([]DownloadPoint, 0, 64)
		var curWeek string
		var sum int
		var count int
		var weekStart time.Time
		flush := func() {
			if count > 0 {
				avg := float64(sum) / float64(count)
				vals = append(vals, avg)
				// place the point at mid-week for better spacing
				mid := weekStart.AddDate(0, 0, 3)
				pts = append(pts, DownloadPoint{Time: mid, Value: avg})
			}
			sum = 0
			count = 0
		}
		for _, d := range parsed.Downloads {
			t, err := time.Parse("2006-01-02", d.Day)
			if err != nil {
				// skip invalid date entries
				continue
			}
			y, w := t.ISOWeek()
			wk := fmt.Sprintf("%04d-%02d", y, w)
			if curWeek == "" {
				curWeek = wk
				weekStart = t
			}
			if wk != curWeek {
				flush()
				curWeek = wk
				weekStart = t
			}
			sum += d.Downloads
			count++
		}
		flush()
		cacheSetDLRange(key, vals)
		return NpmDownloadsRangeMsg{Package: pkg, Values: vals, Points: pts}
	}
}
