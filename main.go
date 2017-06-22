// Package main is the entry point for the ghbackup binary.
// Here is where you can find argument parsing, usage information and the actual execution.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"qvl.io/ghbackup/ghbackup"
)

// Can be set in build step using -ldflags
var version string

const (
	// Printed for -help, -h or with wrong number of arguments
	usage = `Embarrassing simple GitHub backup tool

Usage: %s [flags] directory

  directory  path to save the repositories to

At least one of -account or -secret must be specified.

Flags:
`
	more         = "\nFor more visit https://qvl.io/ghbackup."
	accountUsage = `GitHub user or organization name to get repositories from.
	If not specified, all repositories the authenticated user has access to will be loaded.`
	secretUsage = `Authentication secret for GitHub API.
	Can use the users password or a personal access token (https://github.com/settings/tokens).
	Authentication increases rate limiting (https://developer.github.com/v3/#rate-limiting) and enables backup of private repositories.`
)

// Get command line arguments and start updating repositories
func main() {
	// Flags
	account := flag.String("account", "", accountUsage)
	secret := flag.String("secret", "", secretUsage)
	versionFlag := flag.Bool("version", false, "Print binary version")
	silent := flag.Bool("silent", false, "Suppress all output")

	// Parse args
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, more)
	}
	flag.Parse()

	if *versionFlag {
		fmt.Printf("ghbackup %s %s %s\n", version, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 || (*account == "" && *secret == "") {
		flag.Usage()
		os.Exit(1)
	}

	errs := log.New(os.Stderr, "", 0)
	logs := log.New(os.Stdout, "", 0)
	if *silent {
		logs = log.New(ioutil.Discard, "", 0)
	}

	err := ghbackup.Run(ghbackup.Config{
		Account: *account,
		Dir:     args[0],
		Secret:  *secret,
		Log:     logs,
		Err:     errs,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
