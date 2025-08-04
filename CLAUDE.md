# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Status

**Important**: This repository has been moved to https://github.com/openshift-online/ocm-service-common. This is likely a legacy or archived version.

## Development Commands

### Testing
- `make test` - Run Ginkgo tests for JIRA client and OCM logger packages
- `make test-unit` - Run all unit tests with race detection
  - Set `JUNITFILE` environment variable to output JUnit format
- `ginkgo -r pkg/client/jira pkg/ocmlogger` - Run specific package tests

### Code Quality
- `make verify` - Run standard Go checks including `go vet`
- Uses golangci-lint with configuration in `default.golangci.yml`
- Enabled linters: bodyclose, goconst, gosec, gosimple, ineffassign, lll, misspell, staticcheck, unconvert, govet, unused, forbidigo, gci
- **Important**: Forbids use of `fmt.Error*` functions - use `weberr` package instead (imported as `errors`)

### Build and CI
- `./pr_check.sh` - Full PR validation script (runs make + make test)
- Requires `GOPATH` to be set
- Uses Jenkins environment when `WORKSPACE` is set

## Architecture

This is a Go shared library (`ocm-common`) providing common utilities and middleware for OpenShift Cluster Manager (OCM) services.

### Core Packages

**Client Libraries**:
- `pkg/client/jira/` - JIRA integration client with token and basic auth support
- `pkg/client/notifications/` - Notification service client
- `pkg/client/segment/` - Segment analytics tracking client

**Middleware & HTTP Utilities**:
- `pkg/middleware/region_proxy.go` - Region-aware request proxying with Prometheus metrics
- `pkg/middleware/standard_claims.go` - JWT claims processing
- `pkg/middleware/token.go` - Token validation middleware
- `pkg/middleware/deprecation.go` - Deprecation warning middleware for API versioning
- `pkg/logging/transport.go` - HTTP transport logging wrappers

**Logging System**:
- `pkg/ocmlogger/` - Structured logging based on zerolog (replacement for deprecated glog)
- Supports Sentry integration for Error/Fatal levels
- Context-aware logging with extra field support
- SDK log wrapper for third-party library integration

**Utilities**:
- `pkg/test/` - OCM API testing framework with concurrent test execution, label-based filtering, and continuous monitoring capabilities
- `pkg/csv/` - CSV parsing utilities
- `pkg/grafana/` - Grafana dashboard generation
- `utils/retry.go` - Retry logic utilities
- `utils/validation_helpers.go` - Common validation functions

### Key Dependencies
- OCM SDK (`github.com/openshift-online/ocm-sdk-go`)
- Ginkgo/Gomega for testing
- Zerolog for structured logging
- Prometheus for metrics
- JWT for token handling
- Segment for analytics

### Import Structure
Uses GitLab module path: `gitlab.cee.redhat.com/service/ocm-common`

### Testing Strategy
- Ginkgo BDD testing framework
- Race condition detection enabled
- Mock implementations for external services (HTTP, notifications, segment)
- Continuous testing framework in `pkg/test/`

### Test Framework (`pkg/test/`)
The test package provides a comprehensive framework for OCM API testing with these key capabilities:

**Core Components**:
- `TestSuite` - Main test orchestrator with OCM SDK connection management
- `TestCase` - Individual test definition with setup/teardown hooks and response assertions
- `TestConfig` - Configuration for test execution (sample count, label filtering)
- `Result` - Test execution results with timing, error, and response size metrics

**Key Features**:
- **Concurrent Execution**: Tests run in parallel goroutines with configurable sample counts
- **Label-based Filtering**: Organize and run tests by labels (e.g., "performance", "regression")
- **OCM SDK Integration**: Direct connection to OCM API environments (stage, prod, integration)
- **Response Assertions**: Chainable assertions for validating API responses
- **Continuous Testing**: Long-running test execution with real-time result streaming
- **Environment Abstraction**: Supports multiple OCM environments with account ID mapping
- **Metrics Collection**: Latency, response size, and error rate tracking
- **Setup/Teardown**: Per-test lifecycle hooks for resource management

**Authentication Support**:
- OAuth token-based authentication
- Client ID/secret authentication
- Environment variable configuration (`UHC_TOKEN`, `AMS_CLIENT_ID`, `AMS_CLIENT_SECRET`)

**Usage Pattern**:
```go
suite := BuildTestSuite(&TestSuiteSpec{
    BaseURL: "https://api.stage.openshift.com",
    SdkConnector: &sdkConnector{},
})
results := suite.Run(&TestConfig{SampleCount: 10, Labels: ["performance"]})
```

### Logging Best Practices
- Use `ocmlogger.NewOCMLogger(context.Background())` for structured logging
- Keep log messages constant, use `.Extra()` for variable data
- Use `.Err(err)` to add error context
- Set trim list with `SetTrimList()` to clean file paths in logs
- Test logging by capturing output with `ocmlogger.SetOutput()`