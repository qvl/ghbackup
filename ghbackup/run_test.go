package ghbackup_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"qvl.io/ghbackup/ghbackup"
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
}
