package ghbackup_test

import (
	"io/ioutil"
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
		err := os.RemoveAll(dir)
		if err != nil {
			t.Error(err)
		}
	}()

	updates := make(chan ghbackup.Update)
	go func() {
		for u := range updates {
			switch u.Type {
			case ghbackup.UErr:
				t.Error("Unexpected error:", u.Message)
			}
		}
	}()

	err = ghbackup.Run(ghbackup.Config{
		Account: "qvl",
		Dir:     dir,
		Secret:  os.Getenv("SECRET"),
		Updates: updates,
	})

	if err != nil {
		t.Error("Unexpected error:", err)
	}
}
