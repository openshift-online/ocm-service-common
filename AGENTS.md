# AGENTS.md

This file provides guidance to AI coding assistants when working with this repository.

## Project Overview

OCM Service Common — a shared Go library with utility functions used across OCM backend services. Provides common patterns for authentication, database access, logging, and service infrastructure.

## Build & Test Commands

```bash
make test            # Run all tests
make test-unit       # Run unit tests only
make verify          # Run source verification
make release         # Create a new release
```

## Architecture

- **pkg/**: Library code organized by domain
  - Common service infrastructure utilities
  - Database helpers and patterns
  - Authentication and authorization utilities
- **docs/**: Documentation
- **utils/**: Standalone utility scripts

## Key Conventions

- Module path: `github.com/openshift-online/ocm-service-common`
- Default branch is `master`
- Pure library — no main package
- Follow existing code patterns for new additions
