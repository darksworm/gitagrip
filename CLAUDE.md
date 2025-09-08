# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Run
```bash
# Build the application
go build

# Build and run directly
go run main.go

# Build with specific flags for release
go build -ldflags="-s -w" -o gitagrip

# Install locally
go install
```

### Testing
```bash
# Run all tests with race detection and coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Run tests for specific package
go test ./internal/ui/...

# Run a specific test function
go test -run TestSpecificFunction ./internal/packagename
```

### Code Quality
```bash
# Run standard Go tooling
go vet ./...
go fmt ./...

# Static analysis (install first: go install honnef.co/go/tools/cmd/staticcheck@2025.1)
staticcheck ./...

# Vulnerability check (install first: go install golang.org/x/vuln/cmd/govulncheck@latest)  
govulncheck ./...

# Run golangci-lint (requires separate installation)
golangci-lint run --timeout=5m
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go get -u ./...
go mod tidy
```

## Architecture Overview

This is a terminal UI application built with Go and Bubble Tea for managing multiple Git repositories. The architecture follows a clean, modular design:

### Core Components

- **main.go**: Entry point handling CLI arguments and application initialization
- **internal/domain/**: Core domain models (Repository, Group, RepoStatus)
- **internal/ui/**: Complete UI layer using Bubble Tea framework
- **internal/git/**: Git operations and status checking
- **internal/discovery/**: Repository discovery and scanning
- **internal/groups/**: Repository grouping logic
- **internal/config/**: Configuration management (.gitagrip.toml)
- **internal/eventbus/**: Event-driven communication between components

### UI Architecture (Bubble Tea Model)

The UI follows the Elm architecture pattern via Bubble Tea:

- **model.go**: Main application state and Bubble Tea model
- **messages.go**: Message types for state updates
- **views/**: UI rendering components (repository, group, popup views)
- **input/**: Input handling with different modes (normal, search, filter, etc.)
- **logic/**: Business logic for navigation, filtering, and sorting
- **state/**: Application state management
- **commands/**: Asynchronous command execution

### Key Patterns

1. **Event-Driven**: Uses an event bus for component communication
2. **State Management**: Centralized state with immutable updates
3. **Command Pattern**: Async operations (git fetch, pull) via commands
4. **Repository Pattern**: Abstract data access for repositories and groups
5. **Input Modes**: Modal input handling (normal, search, filter, confirmation)

### Configuration

- Configuration stored in `.gitagrip.toml` in TOML format
- Settings include UI preferences, group definitions, and base directory
- Auto-save functionality for groups and UI state

The application scans directories for Git repositories, organizes them into groups, displays their status with colors and indicators, and allows various Git operations through keyboard shortcuts.