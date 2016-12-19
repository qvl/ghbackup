#  :floppy_disk: ghbackup

[![GoDoc](https://godoc.org/qvl.io/ghbackup?status.svg)](https://godoc.org/qvl.io/ghbackup)
[![Go Report Card](https://goreportcard.com/badge/github.com/qvl/ghbackup)](https://goreportcard.com/report/github.com/qvl/ghbackup)


Embarrassing simple Github backup tool

    Usage: ghbackup [flags] directory

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


## Install

- Via [Go](https://golang.org/) setup: `go get qvl.io/ghbackup`

- Or download latest binary: https://github.com/qvl/ghbackup/releases


## Setup

Mostly, we like to setup backups to run automatically in an interval.
There are different tools to do this:

### Cron

Cron is a job scheduler that already runs on most Unix systems.

Let's setup `ghbackup` on a Linux server and make it run daily at 1am. This works similar on other platforms.

1. Install `ghbackup`: `go get qvl.io/ghbackup`

2. Setup Cron job

- Run `crontab -e`
- Add a new line and replace `NAME` and `DIR` with your options:

``` sh
0 1 * * * ghbackup -account NAME DIR
```

For example:

``` sh
0 1 * * * ghbackup -account qvl /home/qvl/backup-qvl
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


## Use as Go package

From another Go program you can directly use the `ghbackup` sub-package.
Have a look at the [GoDoc](https://godoc.org/qvl.io/ghbackup/ghbackup).


## Development

Make sure to use `gofmt` and create a [Pull Request](https://github.com/qvl/ghbackup/pulls).

### Releasing

Run `./release.sh <version>` and upload the binaries on Github.


## License

[MIT](./license)
