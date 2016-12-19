package ghbackup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

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
	Path string `json:"full_name"`
	URL  string `json:"ssh_url"`
}

const defaultMaxWorkers = 10
const defaultAPI = "https://api.github.com"

// Run update for the given Config.
func Run(config Config) error {
	// Defaults
	if config.Workers == 0 {
		config.Workers = defaultMaxWorkers
	}
	if config.API == "" {
		config.API = defaultAPI
	}
	if config.Doer == nil {
		config.Doer = http.DefaultClient
	}

	// Fetch list of repositories
	repos, err := getRepos(config.Account, config.Secret, config.API, config.Doer)
	if err != nil {
		return err
	}
	config.Updates <- Update{UInfo, fmt.Sprintf("Backup %d repositories", len(repos))}

	// Backup repositories in parallel
	each(repos, config.Workers, func(r repo) {
		err := backupRepo(config.Dir, config.Account, r, config.Updates)
		if err != nil {
			config.Updates <- Update{UErr, err.Error()}
		}
	})

	return nil
}

// Get repositories from Github.
// Follow all "next" links.
func getRepos(account, secret, api string, doer Doer) ([]repo, error) {
	var allRepos []repo

	currentURL, err := getURL(account, secret, api, doer)
	if err != nil {
		return allRepos, err
	}

	// Go through all pages
	for {
		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %v", err)
		}
		if secret != "" {
			// For token authentication `account` will be ignored
			req.SetBasicAuth(account, secret)
		}
		res, err := doer.Do(req)
		if err != nil {
			return nil, fmt.Errorf("cannot get repos: %v", err)
		}
		defer func() {
			_ = res.Body.Close()
		}()
		if res.StatusCode >= 300 {
			return nil, fmt.Errorf("bad response from %s: %v", res.Request.URL, res.Status)
		}

		var repos []repo
		err = json.NewDecoder(res.Body).Decode(&repos)
		if err != nil {
			return nil, fmt.Errorf("cannot decode JSON response: %v", err)
		}

		allRepos = append(allRepos, selectRepos(repos, account)...)

		// Set url for next iteration
		currentURL = getNextURL(res.Header)

		// Done if no next URL
		if currentURL == "" {
			return allRepos, nil
		}
	}
}

func getURL(account, secret, api string, doer Doer) (string, error) {
	user := "user"
	if secret == "" {
		category, err := getCategory(account, api, doer)
		if err != nil {
			return "", err
		}
		user = category + "/" + account
	}
	url := api + "/" + user + "/repos?per_page=100"
	return url, nil
}

// Returns "users" or "orgs" depending on type of account
func getCategory(account, api string, doer Doer) (string, error) {
	req, err := http.NewRequest("GET", strings.Join([]string{api, "users", account}, "/"), nil)
	if err != nil {
		return "", fmt.Errorf("cannot create HTTP request: %v", err)
	}
	res, err := doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot get user info: %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("bad response from %s: %v", res.Request.URL, res.Status)
	}

	var a struct {
		Type string
	}
	err = json.NewDecoder(res.Body).Decode(&a)
	if err != nil {
		return "", fmt.Errorf("cannot decode JSON response: %v", err)
	}

	if a.Type == "User" {
		return "users", nil
	}
	if a.Type == "Organization" {
		return "orgs", nil
	}
	return "", fmt.Errorf("unknown type of account %s for %s", a.Type, account)
}

func selectRepos(repos []repo, account string) []repo {
	if account == "" {
		return repos
	}
	var res []repo
	for _, r := range repos {
		if path.Dir(r.Path) == account {
			res = append(res, r)
		}
	}
	return res
}

func getNextURL(header http.Header) string {
	linkHeader := header["Link"]
	if len(linkHeader) == 0 {
		return ""
	}
	parts := strings.Split(linkHeader[0], ",")
	if len(parts) == 0 {
		return ""
	}
	firstLink := parts[0]
	if !strings.Contains(firstLink, "rel=\"next\"") {
		return ""
	}
	parts = strings.Split(firstLink, ";")
	if len(parts) == 0 {
		return ""
	}
	urlInBrackets := parts[0]
	if len(urlInBrackets) < 3 {
		return ""
	}
	return urlInBrackets[1 : len(urlInBrackets)-1]
}

func each(repos []repo, workers int, worker func(repo)) {
	if len(repos) < workers {
		workers = len(repos)
	}

	jobs := make(chan repo)

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for r := range jobs {
				worker(r)
			}
		}()
	}

	for _, r := range repos {
		jobs <- r
	}
	close(jobs)
	wg.Wait()
}

// Clone new repo or pull in existing repo
func backupRepo(backupDir, account string, r repo, updates chan Update) error {
	repoDir := getRepoDir(backupDir, r.Path, account)

	repoExists, err := exists(repoDir)
	if err != nil {
		return fmt.Errorf("cannot check if repo exists: %v", err)
	}

	var cmd *exec.Cmd
	if repoExists {
		updates <- Update{UInfo, fmt.Sprintf("Updating	%s", r.Path)}
		cmd = exec.Command("git", "remote", "update")
		cmd.Dir = repoDir
	} else {
		updates <- Update{UInfo, fmt.Sprintf("Cloning	%s", r.Path)}
		cmd = exec.Command("git", "clone", "--mirror", "--no-checkout", r.URL, repoDir)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running command %v (%v): %v (%v)", cmd.Args, cmd.Path, string(out), err)
	}
	return nil
}

func getRepoDir(backupDir, repoPath, account string) string {
	repoGit := repoPath + ".git"
	// For single account, skip sub-directories
	if account != "" {
		return filepath.Join(backupDir, path.Base(repoGit))
	}
	return filepath.Join(backupDir, repoGit)
}

// Check if a file or directory exists
func exists(f string) (bool, error) {
	_, err := os.Stat(f)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot get stats for path `%s`: %v", f, err)
	}
	return true, nil
}
