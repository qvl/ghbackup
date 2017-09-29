package ghbackup_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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

	var errs bytes.Buffer
	err = ghbackup.Run(ghbackup.Config{
		Account: "qvl",
		Dir:     dir,
		Secret:  os.Getenv("SECRET"),
		Err:     log.New(&errs, "", log.Lshortfile),
	})

	if errs.Len() != 0 {
		t.Error("Unexpected error messages:", errs.String())
	}
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	// Check contents of backup directory
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Error(err)
	}
	minLen := len(strings.Split(strings.TrimSpace(expectedRepos), " "))
	if len(files) < minLen {
		t.Errorf("Expected to fetch at least %d repositories; got %d", minLen, len(files))
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
