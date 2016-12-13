#  :floppy_disk: gh-backup

[![GoDoc](https://godoc.org/github.com/qvl/gh-backup?status.svg)](https://godoc.org/github.com/qvl/gh-backup)
[![Go Report Card](https://goreportcard.com/badge/github.com/qvl/gh-backup)](https://goreportcard.com/report/github.com/qvl/gh-backup)


Embarrassing simple Github backup tool

    Usage: gh-backup name directory

      name       github user or organization name to get the repositories from
      directory  path to save the repositories to

      -auth string
            Authentication for Github as <user>:<password>.
            Can also use a personal access token instead of password (https://github.com/settings/tokens).
            Authentication increases rate limiting (https://developer.github.com/v3/#rate-limiting).`
      -verbose
            print progress information


## Install

- Via [Go](https://golang.org/) setup: `go get qvl.io/gh-backup`

- Or download latest binary: https://github.com/qvl/gh-backup/releases


## Setup

Mostly, we like to setup backups to run automatically in an interval.
There are different tools to do this:

### Cron

Cron is a job scheduler that already runs on most Unix systems.

Let's setup `gh-backup` on a Linux server and make it run daily at 1am. This works similar on other platforms.

1. Install `gh-backup`: `go get qvl.io/gh-backup`

2. Setup Cron job

- Run `crontab -e`
- Add a new line and replace `NAME` and `DIR` with your options:

``` sh
0 1 * * * gh-backup NAME DIR
```

For example:

``` sh
0 1 * * * gh-backup qvl /home/qvl/backup-qvl
```


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

Make sure to use `gofmt` and create a [Pull Request](https://github.com/qvl/gh-backup/pulls).

### Releasing

Run `./release.sh <version>` and upload the binaries on Github.


## License

[MIT](./LICENSE)
