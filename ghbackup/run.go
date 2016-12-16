package ghbackup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
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
	Name string
	URL  string `json:"git_url"`
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
	u, err := getURL(config.Account, config.API, config.Doer)
	if err != nil {
		return err
	}
	repos, err := getRepos(u, config.Account, config.Secret, config.Doer)
	if err != nil {
		return err
	}
	config.Updates <- Update{UInfo, fmt.Sprintf("Backup for %s with %d repositories", config.Account, len(repos))}

	// Backup repositories in parallel
	jobs := make(chan repo)

	workers := config.Workers
	if len(repos) < workers {
		workers = len(repos)
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for r := range jobs {
				err := backupRepo(config.Dir, r, config.Updates)
				if err != nil {
					config.Updates <- Update{UErr, err.Error()}
				}
			}
		}()
	}

	for _, r := range repos {
		jobs <- r
	}
	close(jobs)
	wg.Wait()

	return nil
}

func getURL(account, api string, doer Doer) (string, error) {
	category, err := getCategory(account, api, doer)
	if err != nil {
		return "", err
	}
	url := api + "/" + category + "/" + account + "/repos?per_page=100&type=owner"
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

// Get repositories from Github.
// Follow all "next" links.
func getRepos(url, account, secret string, doer Doer) ([]repo, error) {
	var allRepos []repo

	// Go through all pages
	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %v", err)
		}
		if len(secret) > 0 {
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

		allRepos = append(allRepos, repos...)

		linkHeader := res.Header["Link"]
		if len(linkHeader) == 0 {
			break
		}
		firstLink := strings.Split(linkHeader[0], ",")[0]
		if !strings.Contains(firstLink, "rel=\"next\"") {
			break
		}
		urlInBrackets := strings.Split(firstLink, ";")[0]
		// Set url for next iteration
		url = urlInBrackets[1 : len(urlInBrackets)-1]
	}

	return allRepos, nil
}

// Clone new repo or pull in existing repo
func backupRepo(backupDir string, r repo, updates chan Update) error {
	repoDir := path.Join(backupDir, r.Name)

	var cmd *exec.Cmd
	repoExists, err := exists(repoDir)
	if err != nil {
		return fmt.Errorf("cannot check if repo exists: %v", err)
	}
	if repoExists {
		updates <- Update{UInfo, fmt.Sprintf("Updating	%s", r.Name)}
		cmd = exec.Command("git", "remote", "update")
		cmd.Dir = repoDir
	} else {
		updates <- Update{UInfo, fmt.Sprintf("Cloning	%s", r.Name)}
		cmd = exec.Command("git", "clone", "--mirror", "--no-checkout", r.URL, repoDir)
	}

	err = cmd.Run()
	if err != nil {
		// Give enough information to reproduce command
		return fmt.Errorf("error running command `%v` (`%v`) in dir `%v` with env `%v`: %v", cmd.Args, cmd.Path, cmd.Dir, cmd.Env, err)
	}
	return nil
}

// Check if a file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot get stats for path `%s`: %v", path, err)
	}
	return true, nil
}
