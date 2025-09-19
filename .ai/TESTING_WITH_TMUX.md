# Testing GitaGrip with tmux-cli

This document describes how to use `tmux-cli` for interactive testing of GitaGrip's TUI interface.

## Key Principles

### Always Reuse the Same Pane
To maintain consistency and avoid creating multiple panes, we should:
1. Check if a testing pane already exists before creating a new one
2. Reuse the existing pane for all test runs
3. Only create a new pane if none exists or the existing one is dead

### Pane Management Strategy
```bash
# Check existing panes first
tmux-cli list_panes

# Look for our dedicated testing pane (e.g., pane 2 or 3)
# If it exists and is alive, reuse it
# If not, create a new one
```

## Setup Testing Environment

### 1. Initial Setup (One-Time)
```bash
# Launch a shell in a dedicated pane for testing
# ALWAYS launch zsh first to prevent losing output on errors
tmux-cli launch "zsh"  # Returns pane ID, e.g., "2"

# Save this pane ID for all subsequent testing
export GITAGRIP_TEST_PANE=2  # Or whatever pane was returned
```

### 2. Reusing the Testing Pane
```bash
# Before each test session, verify the pane is still alive
tmux-cli capture --pane=$GITAGRIP_TEST_PANE

# If the pane is dead or doesn't exist, recreate it
if [ $? -ne 0 ]; then
    tmux-cli launch "zsh"
    # Update the pane ID
    export GITAGRIP_TEST_PANE=<new_pane_id>
fi
```

## Testing Workflow

### 1. Build and Run GitaGrip
```bash
# Build the application
tmux-cli send "go build -o gitagrip ." --pane=$GITAGRIP_TEST_PANE

# Wait for build to complete
tmux-cli wait_idle --pane=$GITAGRIP_TEST_PANE --idle-time=2.0

# Run GitaGrip with test directory
tmux-cli send "./gitagrip -d test-repo" --pane=$GITAGRIP_TEST_PANE
```

### 2. Test Navigation
```bash
# Navigate down (j key)
tmux-cli send "j" --pane=$GITAGRIP_TEST_PANE --enter=False

# Navigate up (k key)
tmux-cli send "k" --pane=$GITAGRIP_TEST_PANE --enter=False

# Capture current state
tmux-cli capture --pane=$GITAGRIP_TEST_PANE
```

### 3. Test Selection
```bash
# Select repository (spacebar)
tmux-cli send " " --pane=$GITAGRIP_TEST_PANE --enter=False

# Select all (a key)
tmux-cli send "a" --pane=$GITAGRIP_TEST_PANE --enter=False

# Deselect all (A key)
tmux-cli send "A" --pane=$GITAGRIP_TEST_PANE --enter=False
```

### 4. Test Search Mode
```bash
# Enter search mode
tmux-cli send "/" --pane=$GITAGRIP_TEST_PANE --enter=False

# Type search term
tmux-cli send "test" --pane=$GITAGRIP_TEST_PANE --delay-enter=0.5

# Navigate search results (n/N)
tmux-cli send "n" --pane=$GITAGRIP_TEST_PANE --enter=False
tmux-cli send "N" --pane=$GITAGRIP_TEST_PANE --enter=False

# Exit search (Escape)
tmux-cli escape --pane=$GITAGRIP_TEST_PANE
```

### 5. Test Filter Mode
```bash
# Enter filter mode
tmux-cli send "F" --pane=$GITAGRIP_TEST_PANE --enter=False

# Apply filter (e.g., modified repos)
tmux-cli send "m" --pane=$GITAGRIP_TEST_PANE --enter=False

# Clear filter
tmux-cli send "c" --pane=$GITAGRIP_TEST_PANE --enter=False
```

### 6. Test Group Operations
```bash
# Create new group
tmux-cli send "N" --pane=$GITAGRIP_TEST_PANE --enter=False
tmux-cli send "Test Group" --pane=$GITAGRIP_TEST_PANE

# Rename group
tmux-cli send "r" --pane=$GITAGRIP_TEST_PANE --enter=False
tmux-cli send "Renamed Group" --pane=$GITAGRIP_TEST_PANE

# Delete group
tmux-cli send "X" --pane=$GITAGRIP_TEST_PANE --enter=False
tmux-cli send "y" --pane=$GITAGRIP_TEST_PANE --enter=False  # Confirm
```

### 7. Test Git Operations
```bash
# View git log (L key)
tmux-cli send "L" --pane=$GITAGRIP_TEST_PANE --enter=False
tmux-cli wait_idle --pane=$GITAGRIP_TEST_PANE --idle-time=2.0
tmux-cli send "q" --pane=$GITAGRIP_TEST_PANE --enter=False  # Exit pager

# View git diff (D key)
tmux-cli send "D" --pane=$GITAGRIP_TEST_PANE --enter=False
tmux-cli wait_idle --pane=$GITAGRIP_TEST_PANE --idle-time=2.0
tmux-cli send "q" --pane=$GITAGRIP_TEST_PANE --enter=False  # Exit pager

# Git pull (p key)
tmux-cli send "p" --pane=$GITAGRIP_TEST_PANE --enter=False
```

### 8. Exit Application
```bash
# Quit GitaGrip
tmux-cli send "q" --pane=$GITAGRIP_TEST_PANE --enter=False

# Or force quit with Ctrl+C
tmux-cli interrupt --pane=$GITAGRIP_TEST_PANE
```

## Automated Test Script Example

```bash
#!/bin/bash

# Configuration
GITAGRIP_TEST_PANE=${GITAGRIP_TEST_PANE:-2}

# Function to ensure test pane exists
ensure_test_pane() {
    if ! tmux-cli capture --pane=$GITAGRIP_TEST_PANE 2>/dev/null; then
        echo "Creating new test pane..."
        GITAGRIP_TEST_PANE=$(tmux-cli launch "zsh" | grep -oE '[0-9]+$')
        echo "Test pane created: $GITAGRIP_TEST_PANE"
    else
        echo "Reusing existing test pane: $GITAGRIP_TEST_PANE"
    fi
}

# Function to run GitaGrip test
test_gitagrip() {
    ensure_test_pane
    
    # Clean any running process
    tmux-cli interrupt --pane=$GITAGRIP_TEST_PANE
    sleep 1
    
    # Build
    echo "Building GitaGrip..."
    tmux-cli send "go build -o gitagrip ." --pane=$GITAGRIP_TEST_PANE
    tmux-cli wait_idle --pane=$GITAGRIP_TEST_PANE --idle-time=3.0
    
    # Run
    echo "Starting GitaGrip..."
    tmux-cli send "./gitagrip -d test-repo" --pane=$GITAGRIP_TEST_PANE
    sleep 3
    
    # Test navigation
    echo "Testing navigation..."
    tmux-cli send "j" --pane=$GITAGRIP_TEST_PANE --enter=False
    sleep 0.5
    tmux-cli send "k" --pane=$GITAGRIP_TEST_PANE --enter=False
    sleep 0.5
    
    # Capture final state
    echo "Capturing output..."
    tmux-cli capture --pane=$GITAGRIP_TEST_PANE
    
    # Cleanup
    echo "Cleaning up..."
    tmux-cli send "q" --pane=$GITAGRIP_TEST_PANE --enter=False
}

# Run test
test_gitagrip
```

## Best Practices

1. **Always Launch Shell First**: Never launch GitaGrip directly - use a shell to prevent losing output on crashes
2. **Reuse Panes**: Check for existing test pane before creating new ones
3. **Use wait_idle**: Instead of fixed sleep times, use `wait_idle` to wait for operations to complete
4. **Capture State**: Regularly capture pane output to verify test results
5. **Clean Exit**: Always properly exit GitaGrip with 'q' before killing the pane
6. **Handle Errors**: Check return codes and handle pane death gracefully

## Troubleshooting

### Pane Not Responding
```bash
# Check if pane is alive
tmux-cli capture --pane=$GITAGRIP_TEST_PANE

# If dead, recreate
tmux-cli launch "zsh"
```

### Lost Output
```bash
# Always launch shell first
tmux-cli launch "zsh"
# Then run commands in the shell
tmux-cli send "./gitagrip" --pane=2
```

### Application Stuck
```bash
# Send interrupt
tmux-cli interrupt --pane=$GITAGRIP_TEST_PANE

# Or send escape
tmux-cli escape --pane=$GITAGRIP_TEST_PANE

# Last resort - kill pane
tmux-cli kill --pane=$GITAGRIP_TEST_PANE
```

## Integration with E2E Tests

This tmux-cli approach can complement the existing E2E test framework:
- E2E tests use PTY for automated testing
- tmux-cli enables manual/interactive testing
- Both approaches can share test scenarios and validation logic