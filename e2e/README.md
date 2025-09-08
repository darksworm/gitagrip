# 🧪 GitaGrip End-to-End Testing

This directory contains comprehensive end-to-end tests for GitaGrip using a native Go testing framework with real PTY (pseudo-terminal) interaction.

## 🚀 Quick Start

```bash
# Run all tests
cd e2e
go test -v -timeout=60s

# Run specific test
go test -v -run TestBasicRepositoryDiscovery

# Run with coverage
go test -v -cover
```

## 📦 Test Framework Architecture

### Core Components

The test framework uses:
- **Real PTY sessions** via `github.com/creack/pty` for authentic terminal interaction
- **Go's native testing** with `testing` package and `testify` assertions
- **Isolated test environments** with temporary workspaces and Git repositories
- **Parallel execution** for faster test runs

### TUITestFramework Methods

```go
// Application lifecycle
tf.StartApp(args...)              // Launch gitagrip with arguments
tf.Cleanup()                      // Clean up resources

// User interaction
tf.SendKeys(keys)                 // Send keystrokes to app
tf.SendEnter()                    // Send Enter key
tf.SendCtrlC()                    // Send Ctrl+C

// Output verification
tf.ReadOutput(timeout)            // Read app output with timeout
tf.WaitForText(text, timeout)     // Wait for specific text

// Test environment
tf.CreateTestWorkspace()          // Create temp directory
tf.CreateTestRepo(name, options)  // Create Git repo with options
```

## 🧪 Current Test Suite

### 1. TestHelpCommand
- Validates `--help` flag functionality
- Checks command-line argument parsing
- Verifies usage information display

### 2. TestBasicRepositoryDiscovery
- Creates multiple test repositories
- Tests application startup and repository discovery
- Validates TUI rendering and repository listing

### 3. TestKeyboardNavigation
- Tests j/k navigation keys
- Verifies UI responsiveness to keyboard input
- Checks navigation state changes

### 4. TestRepositorySelection
- Tests spacebar selection functionality
- Validates selection input handling
- Verifies UI feedback for selections

### 5. TestConfigFileCreation
- Tests automatic `.gitagrip.toml` generation
- Validates config file format and content
- Verifies workspace path inclusion

### 6. TestApplicationExit
- Tests graceful application termination
- Validates 'q' key exit functionality
- Ensures clean process shutdown

## 📝 Writing New Tests

### Basic Test Structure

```go
func TestYourFeature(t *testing.T) {
    t.Parallel() // Enable parallel execution
    tf := NewTUITest(t)
    defer tf.Cleanup()
    
    // 1. Create test environment
    workspace, err := tf.CreateTestWorkspace()
    require.NoError(t, err)
    
    // 2. Create test repositories
    _, err = tf.CreateTestRepo("test-repo", WithDirtyState())
    require.NoError(t, err)
    
    // 3. Start application
    err = tf.StartApp("-d", workspace)
    require.NoError(t, err)
    
    // 4. Wait for startup
    time.Sleep(3 * time.Second)
    
    // 5. Test interactions
    tf.SendKeys("your-key-sequence")
    
    // 6. Verify results
    output, _ := tf.ReadOutput(1 * time.Second)
    assert.Contains(t, output, "expected-content")
}
```

### Repository Options

```go
// Create repo with initial commit (default)
tf.CreateTestRepo("basic-repo")

// Create repo with various states
tf.CreateTestRepo("complex-repo", 
    WithCommit(true),    // Has initial commit
    WithDirtyState(),    // Has uncommitted changes
    WithRemote(),        // Has remote configured
)

// Create empty repo
tf.CreateTestRepo("empty-repo", WithCommit(false))
```

### Assertion Patterns

```go
// Content verification
output, _ := tf.ReadOutput(1 * time.Second)
assert.Contains(t, output, "gitagrip", "Should show app title")
assert.Greater(t, len(output), 100, "Should produce substantial output")

// Wait for specific content
found := tf.WaitForText("Repository discovered", 5*time.Second)
assert.True(t, found, "Should discover repository")

// Check application state
containsRepo := strings.Contains(output, "my-repo")
t.Logf("Output: %d chars, shows repo=%v", len(output), containsRepo)
```

## 🔧 Test Best Practices

### 1. Use Parallel Execution
```go
func TestSomething(t *testing.T) {
    t.Parallel() // Always add this for independent tests
    // ...
}
```

### 2. Clean Output Logging
```go
// ❌ Don't log full TUI output (too verbose)
t.Logf("Full output: %s", output)

// ✅ Log summaries instead
t.Logf("Output: %d chars, contains 'gitagrip'=%v", len(output), strings.Contains(output, "gitagrip"))
```

### 3. Proper Timing
```go
// Wait for app startup
time.Sleep(3 * time.Second)

// Clear output buffer before interactions
tf.ReadOutput(500 * time.Millisecond)

// Send input and wait for response
tf.SendKeys("j")
time.Sleep(1 * time.Second)
```

### 4. Graceful Cleanup
```go
func TestExample(t *testing.T) {
    tf := NewTUITest(t)
    defer tf.Cleanup() // Always defer cleanup
    // ...
}
```

## 🚦 Integration with CI/CD

### GitHub Actions
```yaml
- name: Run E2E Tests
  run: |
    cd e2e
    go test -v -timeout=120s
```

### Local Development
```bash
# Watch mode (requires entr)
find . -name "*.go" | entr -r go test -v

# Coverage report
go test -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 🎯 Advanced Features

### Custom Keyboard Sequences
```go
// Navigation
tf.SendKeys("jjjkkk")  // Down, down, down, up, up, up

// Selection
tf.SendKeys(" ")       // Spacebar to select
tf.SendKeys("a")       // Select all

// Exit
tf.SendKeys("q")       // Quit application
tf.SendCtrlC()         // Force quit
```

### Complex Scenarios
```go
// Multi-step workflow test
func TestComplexWorkflow(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    defer tf.Cleanup()
    
    // Setup multiple repos
    workspace, _ := tf.CreateTestWorkspace()
    for i := 1; i <= 5; i++ {
        tf.CreateTestRepo(fmt.Sprintf("project-%d", i), WithDirtyState())
    }
    
    // Test full workflow
    tf.StartApp("-d", workspace)
    time.Sleep(3 * time.Second)
    
    // Navigate and select
    tf.SendKeys("jj ")    // Down twice, select
    tf.SendKeys("j ")     // Down, select another
    
    // Verify selections
    output, _ := tf.ReadOutput(1 * time.Second)
    // ... assertions
}
```

## 🐛 Debugging Tests

### Enable Verbose Logging
```go
// Add detailed logging for debugging
t.Logf("App started, waiting for repositories...")
output, _ := tf.ReadOutput(2 * time.Second)
t.Logf("Initial output length: %d", len(output))

if strings.Contains(output, "error") {
    t.Logf("Found error in output: %s", output)
}
```

### Manual Testing
```bash
# Test the built binary manually
./gitagrip_e2e -d /path/to/test/repos

# Check if binary was built
ls -la gitagrip_e2e
```

### Common Issues

1. **Test timeouts**: Increase sleep duration for slow machines
2. **PTY issues**: Ensure proper cleanup with `defer tf.Cleanup()`
3. **Race conditions**: Use `t.Parallel()` correctly and avoid shared state
4. **Output buffering**: Clear buffers with `tf.ReadOutput()` before interactions

## 📊 Performance

The framework is optimized for speed:
- ⚡ **Parallel execution** - Tests run concurrently
- 🎯 **Isolated environments** - No test interference  
- 📝 **Clean output** - Concise logging without TUI dumps
- ⏱️ **Fast execution** - Full suite runs in ~6-7 seconds

## 🎉 Benefits

1. **Native Go Integration** - Same language as the main application
2. **Real TUI Testing** - Actual PTY interaction with Bubble Tea
3. **Professional Quality** - Proper test framework with assertions
4. **Easy Maintenance** - Familiar Go patterns and tooling
5. **CI/CD Ready** - Standard Go testing workflow
6. **IDE Integration** - Full debugging and test runner support

This framework provides **production-grade TUI testing** that integrates seamlessly with your Go development workflow!