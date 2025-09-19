# E2E Testing Guide for GitaGrip

This document provides comprehensive guidance for running and writing end-to-end tests for GitaGrip.

## Running E2E Tests

### Using Make (Recommended)
```bash
# Run all E2E tests
make test-e2e

# Run unit tests
make test

# Build the application
make build

# Clean build artifacts
make clean
```

### Using Go Test Directly
```bash
# Run all E2E tests with build tags
go test -tags e2e ./e2e -v

# Run specific test
go test -tags e2e ./e2e -v -run TestBasicRepositoryDiscovery

# Run with timeout
go test -tags e2e ./e2e -v -timeout=60s

# Run with coverage
go test -tags e2e ./e2e -v -cover

# Run tests in parallel (default behavior)
go test -tags e2e ./e2e -v -parallel=4
```

### From E2E Directory
```bash
cd e2e

# Run all tests
go test -v -timeout=60s

# Run specific test
go test -v -run TestKeyboardNavigation

# Watch mode (requires entr)
find . -name "*.go" | entr -r go test -v

# Generate coverage report
go test -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Architecture

### Build Tags
- Tests use `//go:build e2e && unix` build tags
- This ensures tests only run on Unix-like systems (Linux, macOS)
- Tests are excluded from regular `go test ./...` runs

### TestMain Setup
The `testmain_test.go` file:
1. Builds a test binary `gitagrip_e2e` from the main project
2. Sets the binary path for all tests
3. Cleans up the binary after tests complete

### Core Framework Components

#### TUITestFramework
Main testing utility providing:
- PTY (pseudo-terminal) management via `github.com/creack/pty`
- Ring buffer for continuous output capture
- Synchronized output reading with condition variables
- ANSI escape sequence normalization

#### Key Methods

**Application Control:**
```go
tf.StartApp(args...)     // Launch gitagrip with arguments
tf.Cleanup()             // Clean up resources
tf.SendKeys(keys)        // Send keystrokes
tf.SendEnter()           // Send Enter key
tf.SendCtrlC()           // Send Ctrl+C
```

**Output Verification:**
```go
tf.ReadOutput(timeout)   // Read raw output with timeout
tf.Snapshot()            // Get current normalized output
tf.SeePlain(text)        // Check if text exists in output
tf.WaitFor(func, timeout) // Wait for condition
tf.Ready()               // Wait for app initialization
```

**Test Environment:**
```go
tf.CreateTestWorkspace() // Create temp directory
tf.CreateTestRepo(name, options...) // Create Git repo
```

## Writing E2E Tests

### Basic Test Structure
```go
func TestFeatureName(t *testing.T) {
    t.Parallel() // Enable parallel execution
    tf := NewTUITest(t)
    defer tf.Cleanup()
    
    // 1. Setup test environment
    workspace, err := tf.CreateTestWorkspace()
    require.NoError(t, err)
    
    // 2. Create test repositories
    _, err = tf.CreateTestRepo("test-repo", WithDirtyState())
    require.NoError(t, err)
    
    // 3. Start application
    err = tf.StartApp("-d", workspace)
    require.NoError(t, err)
    
    // 4. Wait for initialization
    require.True(t, tf.Ready(), "App should be ready")
    require.True(t, tf.SeePlain("gitagrip"), "Should show title")
    
    // 5. Perform test actions
    tf.SendKeys("j")  // Navigate down
    
    // 6. Verify results
    output := tf.Snapshot()
    require.Contains(t, output, "expected-text")
}
```

### Repository Creation Options

```go
// Basic repository with initial commit
tf.CreateTestRepo("basic-repo")

// Empty repository without commits
tf.CreateTestRepo("empty-repo", WithCommit(false))

// Repository with uncommitted changes
tf.CreateTestRepo("dirty-repo", WithDirtyState())

// Repository with remote configured
tf.CreateTestRepo("remote-repo", WithRemote())

// Repository with custom files
tf.CreateTestRepo("custom-repo", WithFiles(map[string]string{
    "README.md": "# Test Project",
    "main.go": "package main\n",
}))

// Combine multiple options
tf.CreateTestRepo("complex-repo", 
    WithCommit(true),
    WithDirtyState(),
    WithRemote(),
    WithFiles(map[string]string{
        "config.toml": "[settings]\nkey = value",
    }),
)
```

### Testing Keyboard Navigation

```go
// Navigation keys
tf.SendKeys("j")     // Down (vim style)
tf.SendKeys("k")     // Up (vim style)
tf.SendKeys("G")     // Go to bottom
tf.SendKeys("g")     // Go to top

// Alternative helper methods
tf.Down()            // Send 'j' key
tf.Up()              // Send 'k' key
```

### Testing Selection

```go
// Selection operations
tf.SendKeys(" ")     // Toggle selection (spacebar)
tf.SendKeys("a")     // Select all
tf.SendKeys("A")     // Deselect all

// Helper method
tf.Select()          // Send spacebar
```

### Testing Search Mode

```go
// Enter search mode
tf.SendKeys("/")
time.Sleep(100 * time.Millisecond)

// Type search term
tf.SendKeys("repo-name")
tf.SendEnter()

// Navigate search results
tf.SendKeys("n")     // Next match
tf.SendKeys("N")     // Previous match

// Exit search
tf.SendKeys("\x1b")  // Escape key
```

### Testing Filter Mode

```go
// Enter filter mode
tf.SendKeys("F")

// Apply filters
tf.SendKeys("m")     // Modified repos
tf.SendKeys("c")     // Clean repos
tf.SendKeys("u")     // Untracked files

// Clear filter
tf.SendKeys("c")     // Clear all filters
```

### Testing Git Operations

```go
// Git operations
tf.SendKeys("f")     // Fetch
tf.SendKeys("p")     // Pull
tf.SendKeys("P")     // Push
tf.SendKeys("s")     // Status
tf.SendKeys("L")     // View log (opens pager)
tf.SendKeys("D")     // View diff (opens pager)

// Pager operations
tf.OpenDiffPager()   // Helper to open diff
tf.Quit()            // Send 'q' to exit pager
```

### Testing Group Management

```go
// Create new group
tf.SendKeys("N")
tf.SendKeys("My Group Name")
tf.SendEnter()

// Rename group
tf.SendKeys("r")
tf.SendKeys("New Name")
tf.SendEnter()

// Delete group
tf.SendKeys("X")
tf.SendKeys("y")     // Confirm deletion

// Move repositories
tf.SendKeys("m")     // Enter move mode
tf.SendKeys("j")     // Select target group
tf.SendEnter()       // Confirm move
```

## Assertion Patterns

### Basic Assertions
```go
// Check for text presence
require.True(t, tf.SeePlain("gitagrip"), "Should show title")
require.Contains(t, output, "expected-text", "Should contain text")

// Check output changes
initialOutput := tf.Snapshot()
tf.SendKeys("j")
require.True(t, tf.WaitFor(func(s string) bool {
    return s != initialOutput
}, time.Second), "Output should change")

// Check output length
require.Greater(t, len(output), 100, "Should produce output")
```

### Waiting for Conditions
```go
// Wait for specific text
found := tf.WaitForText("Repository discovered", 5*time.Second)
require.True(t, found, "Should discover repository")

// Wait for custom condition
require.True(t, tf.WaitFor(func(output string) bool {
    return strings.Contains(output, "ready") && 
           strings.Contains(output, "3 repositories")
}, 3*time.Second), "Should show 3 repositories")

// Wait for app readiness
require.True(t, tf.Ready(), "App should be ready")
```

### Logging for Debugging
```go
// Concise logging (preferred)
t.Logf("Output: %d chars, contains repo=%v", 
    len(output), strings.Contains(output, "repo"))

// Debug specific issues
if !tf.SeePlain("expected") {
    t.Logf("Missing expected text. Output: %s", tf.Snapshot())
}

// Progress logging
t.Logf("Test stage: Creating repositories...")
t.Logf("Test stage: Starting application...")
t.Logf("Test stage: Testing navigation...")
```

## Best Practices

### 1. Always Use Parallel Execution
```go
func TestSomething(t *testing.T) {
    t.Parallel() // Enables concurrent test execution
    // ...
}
```

### 2. Proper Resource Cleanup
```go
func TestExample(t *testing.T) {
    tf := NewTUITest(t)
    defer tf.Cleanup() // Always defer cleanup
    // ...
}
```

### 3. Timing Considerations
```go
// Wait for app initialization
require.True(t, tf.Ready())

// Small delays for UI updates
time.Sleep(100 * time.Millisecond)

// Use WaitFor instead of fixed sleeps when possible
tf.WaitFor(condition, timeout)
```

### 4. Isolated Test Environments
```go
// Each test gets its own workspace
workspace, _ := tf.CreateTestWorkspace()

// Environment variables are isolated
// HOME is set to workspace
// GIT_CONFIG_GLOBAL is disabled
```

### 5. Deterministic Git Setup
```go
// Always use main branch
tf.runGitCommand(repoPath, "checkout", "-b", "main")

// Set consistent Git config
tf.runGitCommand(repoPath, "config", "user.email", "test@example.com")
tf.runGitCommand(repoPath, "config", "user.name", "Test User")
```

## Common Test Scenarios

### Testing Multiple Repositories
```go
func TestMultipleRepos(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    defer tf.Cleanup()
    
    workspace, _ := tf.CreateTestWorkspace()
    
    // Create various repo states
    tf.CreateTestRepo("clean-repo")
    tf.CreateTestRepo("dirty-repo", WithDirtyState())
    tf.CreateTestRepo("empty-repo", WithCommit(false))
    
    tf.StartApp("-d", workspace)
    require.True(t, tf.Ready())
    
    // Verify all repos are shown
    output := tf.Snapshot()
    require.Contains(t, output, "clean-repo")
    require.Contains(t, output, "dirty-repo")
    require.Contains(t, output, "empty-repo")
}
```

### Testing Error Conditions
```go
func TestInvalidDirectory(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    defer tf.Cleanup()
    
    // Start with non-existent directory
    err := tf.StartApp("-d", "/non/existent/path")
    require.NoError(t, err) // App should start
    
    // Check for error message or empty state
    require.True(t, tf.Ready())
    output := tf.Snapshot()
    // Verify appropriate error handling
}
```

### Testing Configuration
```go
func TestConfigFileCreation(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    defer tf.Cleanup()
    
    workspace, _ := tf.CreateTestWorkspace()
    tf.CreateTestRepo("test-repo")
    
    tf.StartApp("-d", workspace)
    require.True(t, tf.Ready())
    
    // Quit to save config
    tf.Quit()
    
    // Check config file was created
    configPath := filepath.Join(workspace, ".config", "gitagrip", ".gitagrip.toml")
    require.FileExists(t, configPath)
}
```

## Troubleshooting

### Test Failures

**Timeout Issues:**
```go
// Increase timeout for slow operations
require.True(t, tf.WaitFor(condition, 5*time.Second))

// Add more wait time for initialization
time.Sleep(3 * time.Second)
```

**PTY Issues:**
```go
// Ensure proper cleanup
defer tf.Cleanup()

// Check if process is still running
if tf.cmd != nil && tf.cmd.Process != nil {
    // Process is running
}
```

**Output Capture Issues:**
```go
// Clear buffer before critical operations
tf.ReadOutput(100 * time.Millisecond)

// Use Snapshot for normalized output
output := tf.Snapshot() // Removes ANSI codes
```

### Debugging Tips

1. **Enable Verbose Logging:**
```go
t.Logf("Current output: %s", tf.Snapshot())
```

2. **Check Binary Build:**
```bash
ls -la e2e/gitagrip_e2e
file e2e/gitagrip_e2e
```

3. **Run Single Test:**
```bash
go test -tags e2e ./e2e -v -run TestSpecificTest
```

4. **Check Environment:**
```go
t.Logf("Workspace: %s", workspace)
t.Logf("Binary: %s", binPath)
```

## CI/CD Integration

### GitHub Actions
The tests run automatically in CI via `.github/workflows/test.yml`:
```yaml
- name: Run E2E Tests
  run: make test-e2e
```

### Local CI Simulation
```bash
# Run tests as CI would
TERM=xterm-256color go test -tags e2e ./e2e -v -timeout=120s
```

## Performance Considerations

- Tests run in parallel by default
- Full suite completes in ~6-7 seconds
- Each test gets isolated environment
- Ring buffer prevents memory issues with output capture
- ANSI normalization is done efficiently with regex

## Key Constants and Helpers

```go
// Key constants for readability
KeyEnter = "\r"
KeyCtrlC = "\x03"
KeySpace = " "
KeyDown  = "j"
KeyUp    = "k"
KeyQuit  = "q"
KeyDiff  = "D"
KeyLog   = "L"
KeyFetch = "f"
KeyPull  = "p"

// Helper methods
tf.Down()          // Navigate down
tf.Up()            // Navigate up
tf.Select()        // Toggle selection
tf.Quit()          // Send quit key
tf.OpenDiffPager() // Open diff pager
```

## Summary

The E2E testing framework provides:
- ✅ Real PTY interaction with the TUI
- ✅ Parallel test execution
- ✅ Isolated test environments
- ✅ Comprehensive test helpers
- ✅ Clean output normalization
- ✅ Proper resource management
- ✅ CI/CD integration
- ✅ Fast execution (~6-7 seconds for full suite)

This framework ensures GitaGrip's TUI functionality remains stable and reliable across changes.