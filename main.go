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
var usage = `Usage: %s githubname backupdir

  githubname  github user or organization name to get the repositories from
  backupdir   directory path to save the repositories to

`

type repo struct {
	Name   string
	GitURL string `json:"git_url"`
}

var maxWorkers = 10
var githubAPI = "https://api.github.com"

// Get command line arguments and start updating repositories
func main() {
	name, backupDir, verbose := parseArgs()
	// Get line numbers for errors
	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile|log.LUTC)
	// Logger for verbose mode
	var info *log.Logger
	if verbose {
		info = log.New(os.Stderr, "", log.LstdFlags|log.LUTC)
	} else {
		info = log.New(ioutil.Discard, "", 0)
	}

	client := http.DefaultClient

	category, err := getCategory(name, client)
	if err != nil {
		logger.Fatal(err)
	}
	url, err := setMaxPageSize(strings.Join([]string{githubAPI, category, name, "repos"}, "/"))
	if err != nil {
		logger.Fatal(err)
	}
	repos, err := getRepos(url, client)
	if err != nil {
		logger.Fatal(err)
	}

	info.Println("Backup for", category[:len(category)-1], name, "with", len(repos), "repositories")

	jobs := make(chan repo)

	workers := maxWorkers
	if len(repos) < maxWorkers {
		workers = len(repos)
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for r := range jobs {
				err := updateRepo(backupDir, r, info)
				if err != nil {
					logger.Println(err)
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

// Get the two positional arguments githubname and backupdir, and the -verbose flag
func parseArgs() (string, string, bool) {
	verbose := flag.Bool("verbose", false, "print progress information")
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
	return args[0], args[1], *verbose
}

// Returns "users" or "orgs" depending on type of account
func getCategory(name string, client *http.Client) (string, error) {
	res, err := client.Get(strings.Join([]string{githubAPI, "users", name}, "/"))
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
// Follow "next" links recursivly.
func getRepos(u string, client *http.Client) ([]repo, error) {
	res, err := client.Get(u)
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

	linkHeader := res.Header["Link"]
	if len(linkHeader) > 0 {
		firstLink := strings.Split(linkHeader[0], ",")[0]
		if strings.Contains(firstLink, "rel=\"next\"") {
			urlInBrackets := strings.Split(firstLink, ";")[0]
			nextRepos, err := getRepos(urlInBrackets[1:len(urlInBrackets)-1], client)
			return append(repos, nextRepos...), err
		}
	}

	return repos, nil
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
