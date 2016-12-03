# github-backup

[![GoDoc](https://godoc.org/github.com/qvl/github-backup?status.svg)](https://godoc.org/github.com/qvl/github-backup)
[![Go Report Card](https://goreportcard.com/badge/github.com/qvl/github-backup)](https://goreportcard.com/report/github.com/qvl/github-backup)


Embarrassing simple Github backup tool

    Usage: github-backup githubname backupdir

    githubname  github user or organization name to get the repositories from
    backupdir   directory path to save the repositories to

    -verbose
    	print progress information


## Install

- Via [Go](https://golang.org/) setup: `go get github.com/qvl/github-backup`

- Or download latest binary: https://github.com/qvl/github-backup/releases


## What happens?

Get all repositories of a Github user.
Save them to a folder.
Update already cloned repositories.

Best served as a scheduled job to keep your backups up to date!


## Limits

It's repos only. And public only.
If you are more serious about it pick one of the fancy solutions out there
and backup your issues and wikis and private stuff.
Or fork me!


## Development

Make sure to use `gofmt` and create a [Pull Request](https://github.com/qvl/github-backup/pulls).

### Releasing

Run `./release.sh <version>` and upload the binaries on Github.


## License

[MIT](./LICENSE)
