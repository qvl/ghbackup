// Package ghbackup provides access to run all the functionality of ghbackup.
// The binary is just a wrapper around the Run function of package ghbackup.
// This way you can directly use it from any other Go program.
package ghbackup

import (
	"log"
	"net/http"
)

// Config should be passed to Run.
// Only Account, Dir, Updates are required.
type Config struct {
	Account string
	Dir     string
	Skip    []string
	// Optional:
	Err     *log.Logger
	Log     *log.Logger
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

type repo struct {
	Path    string `json:"full_name"`
	URL     string `json:"clone_url"`
	Private bool   `json:"private"`
}

const defaultMaxWorkers = 10
const defaultAPI = "https://api.github.com"
