package ghbackup

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

// Config should be passed to .Run().
// Only Name, Dir, Error, Verbose are required.
type Config struct {
	Name    string
	Dir     string
	Error   *log.Logger
	Verbose *log.Logger
	// Optional:
	Auth       string
	API        string
	Workers    int
	HTTPClient *http.Client
}

type repo struct {
	Name string
	URL  string `json:"git_url"`
}

const defaultMaxWorkers = 10
const defaultAPI = "https://api.github.com"

// Run update for the given Config.
func Run(config Config) {
	// Defaults
	if config.Workers == 0 {
		config.Workers = defaultMaxWorkers
	}
	if config.API == "" {
		config.API = defaultAPI
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	// Fetch list of repositories
	u, err := getURL(config.Name, config.API, config.HTTPClient)
	if err != nil {
		config.Error.Fatal(err)
	}
	repos, err := getRepos(u, config.HTTPClient, config.Auth)
	if err != nil {
		config.Error.Fatal(err)
	}
	config.Verbose.Println("Backup for", config.Name, "with", len(repos), "repositories")

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
				err := backupRepo(config.Dir, r, config.Verbose)
				if err != nil {
					config.Error.Println(err)
				}
			}
		}()
	}

	for _, r := range repos {
		jobs <- r
	}
	close(jobs)
	wg.Wait()
}

func getURL(account, api string, httpClient *http.Client) (string, error) {
	category, err := getCategory(account, api, httpClient)
	if err != nil {
		return "", err
	}
	return setMaxPageSize(api + "/" + category + "/" + account + "/repos?type=owner")
}

// Returns "users" or "orgs" depending on type of account
func getCategory(name, api string, client *http.Client) (string, error) {
	res, err := client.Get(strings.Join([]string{api, "users", name}, "/"))
	if err != nil {
		return "", fmt.Errorf("cannot get user info: %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("bad response from %s: %v", res.Request.URL, res.Status)
	}

	var account struct {
		Type string
	}
	err = json.NewDecoder(res.Body).Decode(&account)
	if err != nil {
		return "", fmt.Errorf("cannot decode JSON response: %v", err)
	}

	if account.Type == "User" {
		return "users", nil
	}
	if account.Type == "Organization" {
		return "orgs", nil
	}
	return "", fmt.Errorf("unknown type of account %s for name %s", account.Type, name)
}

// Get repositories from Github.
// Follow all "next" links.
func getRepos(u string, client *http.Client, auth string) ([]repo, error) {
	var allRepos []repo

	// Go through all pages
	for {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %v", err)
		}
		if len(auth) > 0 {
			parts := strings.Split(auth, ":")
			req.SetBasicAuth(parts[0], parts[1])
		}
		res, err := client.Do(req)
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
		u = urlInBrackets[1 : len(urlInBrackets)-1]
	}

	return allRepos, nil
}

//  Adds per_page=100 to a URL
func setMaxPageSize(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("cannot parse url: %v", err)
	}
	q := u.Query()
	q.Set("per_page", "100")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Clone new repo or pull in existing repo
func backupRepo(backupDir string, r repo, info *log.Logger) error {
	repoDir := path.Join(backupDir, r.Name)

	var cmd *exec.Cmd
	repoExists, err := exists(repoDir)
	if err != nil {
		return fmt.Errorf("cannot check if repo exists: %v", err)
	}
	if repoExists {
		info.Println("Updating	", r.Name)
		cmd = exec.Command("git", "pull")
		cmd.Dir = repoDir
	} else {
		info.Println("Cloning	", r.Name)
		cmd = exec.Command("git", "clone", r.URL, repoDir)
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
