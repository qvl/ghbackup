package main

import (
	"encoding/json"
	"flag"
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

// Printed for -help, -h or with wrong number of arguments
var usage = `Usage: %s githubname backupdir

  githubname  github user or organization name to get the repositories from
  backupdir   directory path to save the repositories to

`

type Repo struct {
	Name   string
	GitUrl string `json:"git_url"`
}

var maxWorkers = 10
var githubApi = "https://api.github.com"

var verboseFlag = flag.Bool("verbose", false, "print progress information")

// Get command line arguments and start updating repositories
func main() {
	name, backupDir := parseArgs()

	category, err := getCategory(name)
	if err != nil {
		log.Fatal(err)
	}
	url, err := setMaxPageSize(strings.Join([]string{githubApi, category, name, "repos"}, "/"))
	if err != nil {
		log.Fatal(err)
	}
	repos, err := getRepos(url)
	if err != nil {
		log.Fatal(err)
	}

	verbose("Backup for", category[:len(category)-1], name, "with", len(repos), "repositories")

	jobs := make(chan Repo)

	workers := maxWorkers
	if len(repos) < maxWorkers {
		workers = len(repos)
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for repo := range jobs {
				err := updateRepo(backupDir, repo)
				if err != nil {
					log.Println(err)
				}
			}
		}()
	}

	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)
	wg.Wait()
}

// Get the two positional arguments githubname and backupdir
func parseArgs() (string, string) {
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
	return args[0], args[1]
}

// Returns "users" or "orgs" depending on type of account
func getCategory(name string) (string, error) {
	res, err := http.Get(strings.Join([]string{githubApi, "users", name}, "/"))
	if err != nil {
		return "", fmt.Errorf("cannot get user info: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("bad response from %s: %v", res.Request.URL, res.Status)
	}

	var account struct {
		Type string
	}
	json.NewDecoder(res.Body).Decode(&account)

	if account.Type == "User" {
		return "users", nil
	}
	if account.Type == "Organization" {
		return "orgs", nil
	}
	return "", fmt.Errorf("unknown type of account %s for name %s", account.Type)
}

// Get repositories from Github.
// Follow "next" links recursivly.
func getRepos(u string) ([]Repo, error) {
	res, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("cannot get repos: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("bad response from %s: %v", res.Request.URL, res.Status)
	}

	var repos []Repo
	json.NewDecoder(res.Body).Decode(&repos)

	linkHeader := res.Header["Link"]
	if len(linkHeader) > 0 {
		firstLink := strings.Split(linkHeader[0], ",")[0]
		if strings.Contains(firstLink, "rel=\"next\"") {
			urlInBrackets := strings.Split(firstLink, ";")[0]
			nextRepos, err := getRepos(urlInBrackets[1 : len(urlInBrackets)-1])
			return append(repos, nextRepos...), err
		}
	}

	return repos, nil
}

//  Adds per_page=100 to a URL
func setMaxPageSize(rawUrl string) (string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		fmt.Errorf("cannot parse url: %v", err)
	}
	q := u.Query()
	q.Set("per_page", "100")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Clone new repo or pull in existing repo
func updateRepo(backupDir string, repo Repo) error {
	repoDir := path.Join(backupDir, repo.Name)

	var cmd *exec.Cmd
	repoExists, err := exists(repoDir)
	if err != nil {
		return fmt.Errorf("cannot check if repo exists: %v", err)
	}
	if repoExists {
		verbose("Update repository:", repo.Name)
		cmd = exec.Command("git", "pull")
		cmd.Dir = repoDir
	} else {
		verbose("Clone  repository:", repo.Name)
		cmd = exec.Command("git", "clone", repo.GitUrl, repoDir)
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

func verbose(info ...interface{}) {
	if *verboseFlag {
		log.Println(info...)
	}
}
