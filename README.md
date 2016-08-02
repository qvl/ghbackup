# github-backup

An embarrassing simple Github backup tool

    Usage: github-backup user backupdir

    user       github user name to get the repositories from
    backupdir  directory path to save the repositories to


## Install

    go get github.com/jorinvo/github-backup


## What happens?

Get all repositories of a Github user.
Save them to folder.
Update already cloned repositories.

Best served as a scheduled job to keep your backups up to date!