# This file prepares the environment for the jobs that run in the Jenkins
# slave. It should be included from scripts like `pr_check.sh` and
# `build_deploy.sh`.

# Prepare a local Go environment that will be removed together with the Jenkins
# workspace:
export GOROOT="/opt/go/1.19.5"
export GOPATH="${WORKSPACE}/.local"
export GOBIN="${WORKSPACE}/.local/bin"
export PATH="${GOBIN}:${GOROOT}/bin:${PATH}"

# Print details of the Go version:
go version
go env
go list ./...

