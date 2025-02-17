# OCM COMMON

This project contains a Go library with utility functions.

## Usage

To use it, run the command `export GOPRIVATE=gitlab.cee.redhat.com` in the terminal,
and then import the `gitlab.cee.redhat.com/service/ocm-common` package.

## How To Release

1. Merge any PRs to master
2. go to https://gitlab.cee.redhat.com/service/ocm-common/-/releases/new
3. select an existing or create a new version tag, e.g. v0.0.78
4. generate release notes from the last version tag: `git log --no-merges --oneline --pretty=format:'%h%x09%an%x09%ad%x09%s' v0.0.77..HEAD`
