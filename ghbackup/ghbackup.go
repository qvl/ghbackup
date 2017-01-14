// Package ghbackup provides access to run all the functionality of ghbackup.
// The binary is just a wrapper around the Run function of package ghbackup.
// This way you can directly use it from any other Go program.
package ghbackup

import "net/http"

// Config should be passed to Run.
// Only Account, Dir, Updates are required.
type Config struct {
	Account string
	Dir     string
	Updates chan Update
	// Optional:
	Secret  string
	API     string
	Workers int
	Doer
}

// Doer makes HTTP requests.
// http.HTTPClient implements Doer but simpler implementations can be used too.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Update is the format of updates emitted while running.
type Update struct {
	Type    UpdateType
	Message string
}

// UpdateType helps you to decide what to do with an Update .
type UpdateType int

const (
	// UErr occurs when something went wrong, but the backup can keep running.
	UErr UpdateType = iota
	// UInfo contains progress information that could be optionally logged.
	UInfo
)

type repo struct {
	Path    string `json:"full_name"`
	URL     string `json:"clone_url"`
	Private bool   `json:"private"`
}

const defaultMaxWorkers = 10
const defaultAPI = "https://api.github.com"
