package main_test

import (
	"bytes"
	"os/exec"
	"testing"
)

const help = `Usage: ghbackup [flags] directory

  directory  path to save the repositories to


  At least one of -account or -secret must be specified.

  -account string
    	Github user or organization name to get repositories from.
	If not specified, all repositories the authenticated user has access to will be loaded.
  -secret string
    	Authentication secret for Github API.
	Can use the users password or a personal access token (https://github.com/settings/tokens).
	Authentication increases rate limiting (https://developer.github.com/v3/#rate-limiting) and enables backup of private repositories.
  -verbose
    	print progress information
`

func TestHelp(t *testing.T) {
	args := [][]string{
		{""},
		{"-h"},
		{"-help"},
		{"--help"},
	}

	for _, a := range args {
		stdout, stderr, ok := run(a)
		if ok {
			t.Error("Zero exit code")
		}
		if stdout != "" {
			t.Error("Unexpected stdout:", stdout)
		}
		if stderr != help {
			t.Error("Unexpected stderr:", stderr)
		}
	}
}

func run(a []string) (string, string, bool) {
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command("ghbackup", a...)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()

	return outbuf.String(), errbuf.String(), err == nil
}
