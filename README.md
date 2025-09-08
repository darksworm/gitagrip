# E2E Testing with Microsoft TUI Test

This directory contains end-to-end tests for GitaGrip using Microsoft's TUI Test framework.

## Setup

```bash
# Install dependencies
npm install

# Build test binary
npm run build-test
```

## Running Tests

```bash
# Run all tests
npm test

# Run tests with traces for debugging
npm run test:trace

# Run specific test file
npx @microsoft/tui-test e2e/01-basic-launch.spec.ts
```

## Test Structure

### 01-basic-launch.spec.ts
- Application startup with various flag combinations
- Repository discovery and auto-grouping
- Basic UI rendering validation

### 02-config-persistence.spec.ts  
- Configuration file creation and loading
- Group persistence across sessions
- Settings preservation

### 03-git-operations.spec.ts
- Git refresh, fetch, and pull operations
- Status updates and progress indicators
- Bulk operations on selected repositories

### 04-ui-interactions.spec.ts
- Keyboard navigation (j/k, arrows, page up/down)
- Repository selection and group expansion
- Search, help dialog, and other UI features

### 05-error-handling.spec.ts
- Error recovery for various failure scenarios
- Network errors, permission issues
- Corrupted repositories and edge cases

### 06-advanced-features.spec.ts
- Group management (create, rename, move)
- Sorting and filtering
- Repository hiding and advanced operations

## Test Utilities

The `helpers.ts` file provides utilities for:

- **TestWorkspace**: Creates temporary git repositories for testing
- **buildTestBinary()**: Compiles the Go application for testing
- **waitForText()**: Waits for specific text to appear in terminal
- Repository creation with various states (clean, dirty, with remotes)

## Debugging

When tests fail, trace files are automatically generated in `tui-traces/`. Use:

```bash
npx @microsoft/tui-test show-trace <trace-file>
```

To replay the exact terminal interactions and debug issues.

## Key Features Tested

✅ **Application Lifecycle**
- Startup with different directory arguments
- Configuration persistence
- Graceful shutdown

✅ **Repository Management** 
- Auto-discovery and grouping
- Status refresh and git operations
- Bulk operations on selections

✅ **User Interface**
- Keyboard navigation and shortcuts
- Search and filtering
- Help system and popups

✅ **Error Handling**
- Network failures and git errors
- File system permission issues  
- Corrupted repository handling

✅ **Advanced Features**
- Group creation and management
- Repository hiding/showing
- Sorting by different criteria

## CI Integration

These tests run in GitHub Actions across multiple platforms (Linux, macOS, Windows) to ensure cross-platform compatibility of the terminal UI.