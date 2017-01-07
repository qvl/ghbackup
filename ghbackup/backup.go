package ghbackup

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// Clone new repo or pull in existing repo
func backup(backupDir, account, secret string, r repo, updates chan Update) error {
	repoDir := getRepoDir(backupDir, r.Path, account)

	repoExists, err := exists(repoDir)
	if err != nil {
		return fmt.Errorf("cannot check if repo exists: %v", err)
	}

	var cmd *exec.Cmd
	if repoExists {
		updates <- Update{UInfo, fmt.Sprintf("Updating %s", r.Path)}
		cmd = exec.Command("git", "remote", "update")
		cmd.Dir = repoDir
	} else {
		updates <- Update{UInfo, fmt.Sprintf("Cloning %s", r.Path)}
		cmd = exec.Command("git", "clone", "--mirror", "--no-checkout", getCloneURL(r, secret), repoDir)
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
