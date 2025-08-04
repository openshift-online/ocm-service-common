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

.PHONY: validate-version
validate-version:
ifndef VERSION
	$(error VERSION is required. Usage: make release VERSION=v0.1.0)
endif
	@# Validate that we're on the master branch
	@CURRENT_BRANCH=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$CURRENT_BRANCH" != "master" ]; then \
		echo "Error: Tags must be created from the master branch. Currently on $$CURRENT_BRANCH"; \
		exit 1; \
	fi
	@# Validate version format (vX.Y.Z)
	@if ! echo "$(VERSION)" | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' > /dev/null; then \
		echo "Error: VERSION must follow the format vX.Y.Z (e.g., v0.1.0)"; \
		exit 1; \
	fi
	@# Check if tag already exists
	@if git tag -l | grep -q "^$(VERSION)$$"; then \
		echo "Error: Tag $(VERSION) already exists"; \
		exit 1; \
	fi
	@# Get the latest tag and compare versions
	@LATEST_TAG=$$(git tag -l 'v*.*.*' | sort -V | tail -1); \
	if [ -n "$$LATEST_TAG" ]; then \
		if ! printf '%s\n%s\n' "$$LATEST_TAG" "$(VERSION)" | sort -V -C; then \
			echo "Error: VERSION $(VERSION) must be higher than the latest tag $$LATEST_TAG"; \
			exit 1; \
		fi; \
	fi

.PHONY: release
release: validate-version
	@echo "Creating and pushing release $(VERSION)..."
	git tag -a -m 'Release $(VERSION)' $(VERSION)
	git push upstream $(VERSION)
	@echo "Release $(VERSION) created and pushed successfully!"