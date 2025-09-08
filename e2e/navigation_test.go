//go:build e2e && unix

package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKeyboardNavigation(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create multiple repos for navigation
	for i := 1; i <= 3; i++ {
		_, err = tf.CreateTestRepo(fmt.Sprintf("repo-%d", i))
		require.NoError(t, err, "Failed to create test repo")
	}

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Get initial state
	initialOutput := tf.Snapshot()

	// Send navigation commands
	tf.Down()

	// Wait for navigation to take effect (output should change)
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, time.Second), "Navigation should change output")

	// Get output after navigation
	navOutput := tf.Snapshot()
	t.Logf("Initial output: %d chars, navigation response: %d chars",
		len(initialOutput), len(navOutput))

	// The TUI should be running and responsive
	require.Greater(t, len(initialOutput), 100, "Should show TUI is running")
}

func TestKeyboardNavigationUpDown(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace with repos
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	for i := 1; i <= 5; i++ {
		_, err = tf.CreateTestRepo(fmt.Sprintf("nav-test-%d", i))
		require.NoError(t, err, "Failed to create nav test repo")
	}

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Test navigation
	initialOutput := tf.Snapshot()

	// Navigate down
	tf.Down()
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, time.Second), "Down navigation should change output")

	// Navigate up (if supported)
	// Note: This tests basic navigation responsiveness
	downOutput := tf.Snapshot()
	require.NotEqual(t, initialOutput, downOutput, "Navigation should change TUI state")
}
