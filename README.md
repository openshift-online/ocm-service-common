# OCM COMMON

This project contains a Go library with utility functions.

## Usage

To use it, run the command `export GOPRIVATE=gitlab.cee.redhat.com` in the terminal,
and then import the `github.com/openshift-online/ocm-service-common` package.

## How To Release

Use the Makefile `release` target to create and push a new version tag:

```shell
git checkout master
git pull
make release VERSION=v0.1.39
```

The `make release` target will:

- Validate that you're on the `master` branch
- Check that the version follows semantic versioning format (vX.Y.Z)
- Ensure the new version is higher than the latest existing tag
- Create an annotated tag with a proper release message
- Push the tag to the `upstream` remote

Note that a repository administrator may need to push the tag to the repository due to access restrictions.
