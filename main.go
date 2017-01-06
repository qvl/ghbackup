package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"qvl.io/ghbackup/ghbackup"
)

const version = "v1.2"

const (
	// Printed for -help, -h or with wrong number of arguments
	usage = `Embarrassing simple Github backup tool

Usage: %s [flags] directory

  directory  path to save the repositories to

At least one of -account or -secret must be specified.

Flags:
`
	more         = "\nFor more visit https://qvl.io/ghbackup."
	accountUsage = `Github user or organization name to get repositories from.
	If not specified, all repositories the authenticated user has access to will be loaded.`
	secretUsage = `Authentication secret for Github API.
	Can use the users password or a personal access token (https://github.com/settings/tokens).
	Authentication increases rate limiting (https://developer.github.com/v3/#rate-limiting) and enables backup of private repositories.`
)

// Get command line arguments and start updating repositories
func main() {
	// Flags
	account := flag.String("account", "", accountUsage)
	secret := flag.String("secret", "", secretUsage)
	versionFlag := flag.Bool("version", false, "Print binary version")
	silent := flag.Bool("silent", false, "Surpress all output")

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

	// Log updates
	updates := make(chan ghbackup.Update)
	go func() {
		for u := range updates {
			switch u.Type {
			case ghbackup.UErr:
				log.Println(u.Message)
			case ghbackup.UInfo:
				if !*silent {
					log.Println(u.Message)
				}
			}
		}
	}()

	err := ghbackup.Run(ghbackup.Config{
		Account: *account,
		Dir:     args[0],
		Secret:  *secret,
		Updates: updates,
	})

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
