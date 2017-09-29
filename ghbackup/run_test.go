package ghbackup_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"qvl.io/ghbackup/ghbackup"
)

const (
	expectedRepos = " ghbackup homebrew-tap promplot qvl.io slangbrain.com sleepto "
	gitFiles      = " HEAD branches config description hooks info objects packed-refs refs "
)

func TestRun(t *testing.T) {
	dir, err := ioutil.TempDir("", "qvl-backup")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	}()

	var logs, errs bytes.Buffer
	err = ghbackup.Run(ghbackup.Config{
		Account: "qvl",
		Dir:     dir,
		Secret:  os.Getenv("SECRET"),
		Log:     log.New(&logs, "", 0),
		Err:     log.New(&errs, "", log.Lshortfile),
	})

	if errs.Len() != 0 {
		t.Error("Unexpected error messages:", errs.String())
	}
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	// Check log output to be of the following form:
	//   6 repositories:
	//   Cloning qvl/ghbackup
	//   Cloning qvl/slangbrain.com
	//   Cloning qvl/qvl.io
	//   Cloning qvl/sleepto
	//   Cloning qvl/promplot
	//   Cloning qvl/homebrew-tap
	//   done: 6 new, 0 updated, 0 unchanged, 3979 total objects
	lines := strings.Split(logs.String(), "\n")
	countFirstLine, err := strconv.Atoi(strings.Split(lines[0], " ")[0])
	if err != nil {
		t.Errorf("Cannot parse repository count from first line of output: '%s'", lines[0])
	}
	if !strings.HasPrefix(lines[countFirstLine+1], fmt.Sprintf("done: %d new, 0 updated, 0 unchanged, ", countFirstLine)) {
		t.Errorf("Last line contains unexpected status information: '%s'", lines[countFirstLine+1])
	}

	// Check contents of backup directory
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Error(err)
	}
	minRepos := len(strings.Split(strings.TrimSpace(expectedRepos), " "))
	if len(files) < minRepos {
		t.Errorf("Expected to fetch at least %d repositories; got %d", minRepos, len(files))
	}

	for _, f := range files {
		if !f.IsDir() {
			t.Errorf("Expected %s to be a directory", f.Name())
		}
		strings.Contains(expectedRepos, " "+f.Name()+".git ")
		repoFiles, err := ioutil.ReadDir(filepath.Join(dir, f.Name()))
		if err != nil {
			t.Error(err)
		}

		if len(repoFiles) < 8 {
			t.Errorf("Expected repository %s to contain at least 8 files; found %d", f.Name(), len(repoFiles))
		}
		for _, r := range repoFiles {
			if !strings.Contains(gitFiles, " "+r.Name()+" ") {
				t.Errorf("Expected repo %s to contain only files '%s'; found '%s'", f.Name(), gitFiles, r.Name())
			}
		}
	}
}
