# Important: Task Documentation
**Always check `.ai/` directory for task-specific documentation before proceeding with any work.**

# Development Commands

## Build & Run
```bash
go build -o gitagrip .
go run main.go
make build
```

## Testing
```bash
# All tests
go test -v -race -coverprofile=coverage.out ./...

# Single test
go test -run TestFunctionName ./internal/package

# E2E tests
make test-e2e
go test -tags e2e ./e2e -v -run TestName
```

## Code Quality
```bash
go vet ./...
go fmt ./...
golangci-lint run --timeout=5m
staticcheck ./...
```

# Code Style Guidelines

## Imports & Formatting
- Use standard Go formatting: `go fmt ./...`
- Group imports: standard, third-party, local
- Use absolute imports for internal packages: `gitagrip/internal/...`

## Naming Conventions
- MixedCaps for exported names (PascalCase)
- mixedCaps for private names (camelCase)
- Interface names: simple noun or -er suffix
- Error variables: `ErrSomething`

## Types & Error Handling
- Use concrete types unless interface needed
- Return errors explicitly, don't panic
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Use domain models from `internal/domain/`

## Architecture Patterns
- Follow event-driven architecture via `internal/eventbus`
- Use Bubble Tea Elm architecture for UI
- Centralize state in `internal/ui/state/AppState`
- Async operations via `internal/ui/commands`
- Modal input system in `internal/ui/input/modes`