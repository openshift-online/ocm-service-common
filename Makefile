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
