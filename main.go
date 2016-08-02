package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
)

var usage = `Usage: %s githubname backupdir

githubname  github user or organization name to get the repositories from
backupdir   directory path to save the repositories to`

type Repo struct {
	Name   string
	GitUrl string `json:"git_url"`
}

var maxWorkers = 10
var githubApi = "https://api.github.com"

func main() {
	name, backupDir := parseArgs()

	category := getCategory(name)
	repos := getRepos(setMaxPageSize(strings.Join([]string{githubApi, category, name, "repos"}, "/")))

	fmt.Println("Backup for", category[:len(category)-1], name, "with", len(repos), "repositories")

	jobs := make(chan Repo)
	for w := 0; w < maxWorkers; w++ {
		go func() {
			for repo := range jobs {
				updateRepo(backupDir, repo)
			}
		}()
	}
	for _, repo := range repos {
		jobs <- repo
	}
}

// Get the two positional arguments githubname and backupdir
func parseArgs() (string, string) {
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		os.Exit(1)
	}
	return args[0], args[1]
}

// Returns "users" or "orgs" depending on type of account
func getCategory(name string) string {
	r, err := http.Get(strings.Join([]string{githubApi, "users", name}, "/"))
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 {
		panic(fmt.Sprint("Request to ", r.Request.URL, " with bad status code ", r.StatusCode))
	}

	var account struct {
		Type string
	}
	json.NewDecoder(r.Body).Decode(&account)

	if account.Type == "User" {
		return "users"
	}
	if account.Type == "Organization" {
		return "orgs"
	}
	panic(fmt.Sprint("Unknown type of account ", account.Type, " for name ", name))
}

// Get repositories from Github.
// Follow "next" links recursivly.
func getRepos(u string) []Repo {
	r, err := http.Get(u)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	if r.StatusCode >= 300 {
		panic(fmt.Sprint("Request to ", u, " with bad status code ", r.StatusCode))
	}

	var repos []Repo
	json.NewDecoder(r.Body).Decode(&repos)

	linkHeader := r.Header["Link"]
	if len(linkHeader) > 0 {
		firstLink := strings.Split(linkHeader[0], ",")[0]
		if strings.Contains(firstLink, "rel=\"next\"") {
			urlInBrackets := strings.Split(firstLink, ";")[0]
			return append(repos, getRepos(urlInBrackets[1:len(urlInBrackets)-1])...)
		}
	}

	return repos
}

func setMaxPageSize(rawUrl string) string {
	u, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}
	q := u.Query()
	q.Set("per_page", "100")
	u.RawQuery = q.Encode()
	return u.String()
}

// Clone new repo or pull in existing repo
func updateRepo(backupDir string, repo Repo) {
	repoDir := path.Join(backupDir, repo.Name)

	var cmd *exec.Cmd
	if exists(repoDir) {
		defer fmt.Println("Updated repository:", repo.Name)

		cmd = exec.Command("git", "pull")
		cmd.Dir = repoDir
	} else {
		defer fmt.Println("Cloned  repository:", repo.Name)

		cmd = exec.Command("git", "clone", repo.GitUrl, repoDir)
	}

	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// Check if a file or directory exists
func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			panic(err)
		}
	}
	return true
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
