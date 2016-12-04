package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

// Printed for -help, -h or with wrong number of arguments
const usage = `Usage: %s name directory

  name       github user or organization name to get the repositories from
  directory  directory path to save the repositories to

`

type repo struct {
	Name   string
	GitURL string `json:"git_url"`
}

const defaultMaxWorkers = 10
const defaultGithubAPI = "https://api.github.com"

// Get command line arguments and start updating repositories
func main() {
	// Flags
	verbose := flag.Bool("verbose", false, "print progress information")

	// Parse args
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	// Logger for verbose mode
	var verboseLogger *log.Logger
	if *verbose {
		verboseLogger = log.New(os.Stderr, "", log.LstdFlags|log.LUTC)
	} else {
		verboseLogger = log.New(ioutil.Discard, "", 0)
	}

	backup(backupOpts{
		name:       args[0],
		dir:        args[1],
		httpClient: http.DefaultClient,
		// Log errors with line numbers
		err:     log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile|log.LUTC),
		verbose: verboseLogger,
	})
}

type backupOpts struct {
	name       string
	dir        string
	httpClient *http.Client
	err        *log.Logger
	verbose    *log.Logger
}

// Update repos for the given options
func backup(opts backupOpts) {
	category, err := getCategory(opts.name, opts.httpClient)
	if err != nil {
		opts.err.Fatal(err)
	}

	url, err := setMaxPageSize(strings.Join([]string{defaultGithubAPI, category, opts.name, "repos"}, "/"))
	if err != nil {
		opts.err.Fatal(err)
	}

	repos, err := getRepos(url, opts.httpClient)
	if err != nil {
		opts.err.Fatal(err)
	}

	opts.verbose.Println("Backup for", category[:len(category)-1], opts.name, "with", len(repos), "repositories")

	jobs := make(chan repo)

	workers := defaultMaxWorkers
	if len(repos) < defaultMaxWorkers {
		workers = len(repos)
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for r := range jobs {
				err := updateRepo(opts.dir, r, opts.verbose)
				if err != nil {
					opts.err.Println(err)
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

// Returns "users" or "orgs" depending on type of account
func getCategory(name string, client *http.Client) (string, error) {
	res, err := client.Get(strings.Join([]string{defaultGithubAPI, "users", name}, "/"))
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
func getRepos(url string, client *http.Client) ([]repo, error) {
	var allRepos []repo

	// Go through all pages
	for {
		res, err := client.Get(url)
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
func updateRepo(backupDir string, r repo, info *log.Logger) error {
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
		cmd = exec.Command("git", "clone", r.GitURL, repoDir)
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
