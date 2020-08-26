package ghbackup

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type repoState int

const (
	stateNew = iota
	stateChanged
	stateUnchanged
	stateFailed
)

// Clone new repo or pull in existing repo.
// Returns state of repo.
func (c Config) backup(r repo) (repoState, error) {
	repoDir := getRepoDir(c.Dir, r.Path, c.Account)

	repoExists, err := exists(repoDir)
	if err != nil {
		return stateFailed, fmt.Errorf("cannot check if repo exists: %v", err)
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
		if !repoExists {
			// clean up clone dir after a failed clone
			// if it was a clean clone only
			_ = os.RemoveAll(repoDir)
		}
		return stateFailed, fmt.Errorf("error running command %v (%v): %v (%v)", maskSecrets(cmd.Args, []string{c.Secret}), cmd.Path, string(out), err)
	}
	return gitState(repoExists, string(out)), nil
}

// maskSecrets hides sensitive data
func maskSecrets(values, secrets []string) []string {
	out := make([]string, len(values))
	for vIndex, value := range values {
		out[vIndex] = value
	}
	for _, secret := range secrets {
		for vIndex, value := range out {
			out[vIndex] = strings.Replace(value, secret, "###", -1)
		}
	}
	return out
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

// Get the state of a repo from command output.
func gitState(repoExisted bool, out string) repoState {
	if !repoExisted {
		return stateNew
	}
	if lines := strings.Split(out, "\n"); len(lines) > 2 {
		return stateChanged
	}
	return stateUnchanged
}
