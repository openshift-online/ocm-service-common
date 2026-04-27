# OCM COMMON

This project contains a Go library with utility functions.

## Quick Start

```bash
make test        # Run all tests
make verify      # Run source verification
```

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

## Installation

```bash
export GOPRIVATE=gitlab.cee.redhat.com
go get github.com/openshift-online/ocm-service-common
```

## Development

### Testing

```bash
make test             # Run all tests
make test-unit        # Run unit tests only
make verify           # Run source verification
```

### Contributing

1. Fork this repository
2. Create a feature branch from `master`
3. Make your changes
4. Run `make test` and `make verify`
5. Submit a pull request
