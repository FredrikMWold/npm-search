package commands

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
			// Augmented fields (not from the API)
			DownloadsLastWeek int    `json:"-"`
			License           string `json:"-"`
			Author            string `json:"-"`
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
