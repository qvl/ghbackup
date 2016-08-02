package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
)

var usage = `Usage: %s user backupdir

user       github user name to get the repositories from
backupdir  directory path to save the repositories to`

type Repo struct {
	Name   string
	GitUrl string `json:"git_url"`
}

var batchSize = 10

func main() {
	user, backupDir := parseArgs()

	repos := getRepos(fmt.Sprint("https://api.github.com/users/", user, "/repos"))

	fmt.Println("Backup for user", user, "with", len(repos), "repositories")

	jobs := make(chan Repo)
	for w := 0; w < batchSize; w++ {
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

// Get the two positional arguments user and backupdir
func parseArgs() (string, string) {
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		os.Exit(1)
	}
	return args[0], args[1]
}

// Get repositories from Github.
// Follow "next" links recursivly.
func getRepos(url string) []Repo {
	r, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	if r.StatusCode >= 300 {
		panic(fmt.Sprint("Request to ", url, "with bad status code ", r.StatusCode))
	}

	var repos []Repo
	json.NewDecoder(r.Body).Decode(&repos)

	firstLink := strings.Split(r.Header["Link"][0], ",")[0]
	if strings.Contains(firstLink, "rel=\"next\"") {
		urlInBrackets := strings.Split(firstLink, ";")[0]
		return append(repos, getRepos(urlInBrackets[1:len(urlInBrackets)-1])...)
	}

	return repos
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
