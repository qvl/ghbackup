package ghbackup

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Clone new repo or pull in existing repo.
// Returns the count of objects fetched and a boolean indicating if the repository is new.
func (c Config) backup(r repo) (int, bool, error) {
	repoDir := getRepoDir(c.Dir, r.Path, c.Account)

	repoExists, err := exists(repoDir)
	if err != nil {
		return 0, false, fmt.Errorf("cannot check if repo exists: %v", err)
	}

	var cmd *exec.Cmd
	if repoExists {
		c.Log.Printf("Updating %s", r.Path)
		cmd = exec.Command("git", "remote", "update")
		cmd.Dir = repoDir
	} else {
		c.Log.Printf("Cloning %s", r.Path)
		cmd = exec.Command("git", "clone", "--mirror", "--no-checkout", "--progress", getCloneURL(r, c.Secret), repoDir)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, false, fmt.Errorf("error running command %v (%v): %v (%v)", cmd.Args, cmd.Path, string(out), err)
	}

	return gitObjectCount(string(out)), !repoExists, nil
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

// Add secret token to URL of private repos.
// Allows cloning without manual authentication or SSH setup.
// However, this saves the secret in the git config file.
func getCloneURL(r repo, secret string) string {
	if !r.Private {
		return r.URL
	}
	u, err := url.Parse(r.URL)
	if err != nil {
		return ""
	}
	u.User = url.User(secret)
	return u.String()
}

// Get the object count from git output.
// Returns 0 if parsing failed.
// Works for `clone` and `remote update` with output like this:
//
// Cloning into bare repository 'ghbackup.git'...
// remote: Counting objects: 334, done.
// remote: Total 334 (delta 0), reused 0 (delta 0), pack-reused 334
// Receiving objects: 100% (334/334), 55.41 KiB | 209.00 KiB/s, done.
// Resolving deltas: 100% (172/172), done.

// Fetching origin
// remote: Counting objects: 5, done.
// remote: Total 5 (delta 3), reused 3 (delta 3), pack-reused 2
// Unpacking objects: 100% (5/5), done.
func gitObjectCount(out string) int {
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return 0
	}
	fields := strings.Split(lines[1], " ")
	if len(fields) < 4 {
		return 0
	}
	count, err := strconv.Atoi(strings.TrimSuffix(fields[3], ","))
	if err != nil {
		return 0
	}
	return count
}
