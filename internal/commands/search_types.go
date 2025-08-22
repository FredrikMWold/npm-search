package commands

import "time"

// NpmSearchMsg is emitted when an npm search completes
type NpmSearchMsg struct {
	Query  string
	Result NpmSearchResult
	Err    error
}

// NpmSearchResult models the subset of the npm search payload we care about
type NpmSearchResult struct {
	Objects []NpmSearchObject `json:"objects"`
	Total   int               `json:"total"`
	Time    string            `json:"time"`
}

// NpmSearchObject represents one entry from the search API
type NpmSearchObject struct {
	Package NpmPackage `json:"package"`
	Score   NpmScore   `json:"score"`
	// SearchScore is present in search API; may be 0 for constructed objects
	SearchScore float64 `json:"searchScore"`
}

// NpmPackage is the package metadata subset used by the UI
type NpmPackage struct {
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
	// Augmented fields (not from the API)
	DownloadsLastWeek int    `json:"-"`
	License           string `json:"-"`
	Author            string `json:"-"`
}

// NpmScore mirrors the score field in search API
type NpmScore struct {
	Final  float64 `json:"final"`
	Detail struct {
		Quality     float64 `json:"quality"`
		Popularity  float64 `json:"popularity"`
		Maintenance float64 `json:"maintenance"`
	} `json:"detail"`
}

// downloadsPointResponse is the payload from the per-package downloads API
type downloadsPointResponse struct {
	Downloads int    `json:"downloads"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Package   string `json:"package"`
}

// NpmDownloadsRangeMsg carries downloads-over-time values for a package.
// Values are ordered from oldest to newest.
type NpmDownloadsRangeMsg struct {
	Package string
	Values  []float64
	Points  []DownloadPoint
	Err     error
}

// rangeDownloadsResponse models the /downloads/range API response
type rangeDownloadsResponse struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Package   string `json:"package"`
	Downloads []struct {
		Day       string `json:"day"`
		Downloads int    `json:"downloads"`
	} `json:"downloads"`
}

// DownloadPoint is a typed time/value pair if needed by callers.
type DownloadPoint struct {
	Time  time.Time
	Value float64
}
