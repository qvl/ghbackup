# github-backup

[![Go Report Card](https://goreportcard.com/badge/github.com/jorinvo/github-backup)](https://goreportcard.com/report/github.com/jorinvo/github-backup)


Embarrassing simple Github backup tool

    Usage: github-backup githubname backupdir

    githubname  github user or organization name to get the repositories from
    backupdir   directory path to save the repositories to


## Install

    go get github.com/jorinvo/github-backup


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

## License

[MIT](./LICENSE)