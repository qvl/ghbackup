package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"qvl.io/ghbackup/ghbackup"
)

const (
	// Printed for -help, -h or with wrong number of arguments
	usage = `Usage: %s name directory

  name       github user or organization name to get the repositories from
  directory  path to save the repositories to

`
	authUsage = `Basic auth for Github API as <user>:<password>.
	Can also use a personal access token instead of password (https://github.com/settings/tokens).
	Authentication increases rate limiting (https://developer.github.com/v3/#rate-limiting).`
)

// Get command line arguments and start updating repositories
func main() {
	// Flags
	auth := flag.String("auth", "", authUsage)
	verboseFlag := flag.Bool("verbose", false, "print progress information")

	// Parse args
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	// Logger for verbose mode
	var verboseLogger *log.Logger
	if *verboseFlag {
		verboseLogger = log.New(os.Stderr, "", log.LstdFlags|log.LUTC)
	} else {
		verboseLogger = log.New(ioutil.Discard, "", 0)
	}

	ghbackup.Run(ghbackup.Config{
		Name: args[0],
		Dir:  args[1],
		Auth: *auth,
		// Log errors with line numbers
		Error:   log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile|log.LUTC),
		Verbose: verboseLogger,
	})
}
