# Verifies that source passes standard checks.
verify: check-gopath
	go vet \
		./pkg/...
.PHONY: verify

# Checks if a GOPATH is set, or emits an error message
check-gopath:
ifndef GOPATH
	$(error GOPATH is not set)
endif
.PHONY: check-gopath

export ACK_GINKGO_DEPRECATIONS = 2.7.0
.PHONY: test
test:
	ginkgo $(ginkgo_flags) -r ./...


.PHONY: test-unit
test-unit:
ifndef JUNITFILE
	go test -race ./...
else
ifeq (, $(shell which gotestsum 2>/dev/null))
	$(error gotestsum not found! Get it by `go get -mod='' -u github.com/openshift/release/tools/gotest2junit`.)
endif
	gotestsum --junitfile $(JUNITFILE) -- -race ./...
endif
.PHONY: test-unit