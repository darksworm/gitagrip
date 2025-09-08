//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFilterFunctionality(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create two repos with different names for filtering
	_, err = tf.CreateTestRepo("alpha-project")
	require.NoError(t, err, "Failed to create alpha-project repo")

	_, err = tf.CreateTestRepo("beta-project")
	require.NoError(t, err, "Failed to create beta-project repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repos to be discovered and displayed
	require.True(t, tf.OutputContainsPlain("alpha-project", 5*time.Second), "alpha-project should be discovered")
	require.True(t, tf.OutputContainsPlain("beta-project", 5*time.Second), "beta-project should be discovered")

	// Get initial output to verify both repos are visible
	initialOutput := tf.SnapshotPlain()
	require.Contains(t, initialOutput, "alpha-project", "Should show alpha-project initially")
	require.Contains(t, initialOutput, "beta-project", "Should show beta-project initially")

	// Enter filter mode with 'F'
	err = tf.SendKeys("F")
	require.NoError(t, err, "Failed to send F key to enter filter mode")

	// Wait for filter prompt to appear
	require.True(t, tf.WaitFor(func(s string) bool {
		return tf.SeePlain("Filter:")
	}, 2*time.Second), "Filter prompt should appear")

	// Type filter text that matches only alpha-project
	filterText := "alpha"
	for _, char := range filterText {
		err = tf.SendKeys(string(char))
		require.NoError(t, err, "Failed to send character: %c", char)
		// Brief pause between keystrokes to ensure they're processed
		time.Sleep(50 * time.Millisecond)
	}

	// Submit the filter with Enter
	err = tf.SendKeys("\r")
	require.NoError(t, err, "Failed to send Enter to submit filter")

	// Wait for filtering to take effect - wait for filter indicator to appear
	require.True(t, tf.OutputContains("[Filter: alpha]", 3*time.Second), "Filter indicator should appear")

	// Get filtered output
	filteredOutput := tf.SnapshotPlain()

	// Check if filter indicator is shown
	require.Contains(t, filteredOutput, "[Filter: alpha]", "Filter indicator should be visible")

	// The filtering should work by now. Let's check the final state by looking
	// for the filter indicator and then checking what comes after it
	parts := strings.Split(filteredOutput, "[Filter: alpha]")
	if len(parts) < 2 {
		t.Fatal("Filter indicator not found in output")
	}

	// Get the content after the filter indicator
	afterFilter := parts[1]
	t.Logf("Content after filter: %s", afterFilter)

	// Verify that alpha-project appears after the filter indicator
	require.Contains(t, afterFilter, "alpha-project", "alpha-project should be visible after filtering")

	// Verify that beta-project does NOT appear after the filter indicator
	require.NotContains(t, afterFilter, "beta-project", "beta-project should be filtered out")

	// Test clearing the filter by entering filter mode again and submitting empty text
	err = tf.SendKeys("F")
	require.NoError(t, err, "Failed to send F key to enter filter mode again")

	// Wait for filter prompt
	require.True(t, tf.WaitFor(func(s string) bool {
		return tf.SeePlain("Filter:")
	}, 2*time.Second), "Filter prompt should appear again")

	// Submit empty filter with Enter (should clear filter)
	err = tf.SendKeys("\r")
	require.NoError(t, err, "Failed to send Enter to clear filter")

	// Wait for filter to be cleared - check that both repos are visible again
	require.True(t, tf.OutputContainsPlain("beta-project", 3*time.Second), "beta-project should be visible again after clearing filter")

	// Get output after clearing filter
	clearedOutput := tf.SnapshotPlain()

	// Verify both repos are visible again
	require.Contains(t, clearedOutput, "alpha-project", "alpha-project should be visible after clearing filter")
	require.Contains(t, clearedOutput, "beta-project", "beta-project should be visible after clearing filter")
}

func TestFilterWithPartialMatch(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create repos with names that have partial matches
	_, err = tf.CreateTestRepo("my-project")
	require.NoError(t, err, "Failed to create my-project repo")

	_, err = tf.CreateTestRepo("your-project")
	require.NoError(t, err, "Failed to create your-project repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repos to be discovered
	require.True(t, tf.OutputContainsPlain("my-project", 5*time.Second), "my-project should be discovered")
	require.True(t, tf.OutputContainsPlain("your-project", 5*time.Second), "your-project should be discovered")

	// Verify both repos are initially visible
	initialOutput := tf.SnapshotPlain()
	require.Contains(t, initialOutput, "my-project", "Should show my-project initially")
	require.Contains(t, initialOutput, "your-project", "Should show your-project initially")

	// Enter filter mode and filter with "my"
	err = tf.SendKeys("F")
	require.NoError(t, err, "Failed to enter filter mode")

	require.True(t, tf.WaitFor(func(s string) bool {
		return tf.SeePlain("Filter:")
	}, 2*time.Second), "Filter prompt should appear")

	// Type filter text
	filterText := "my"
	for _, char := range filterText {
		err = tf.SendKeys(string(char))
		require.NoError(t, err, "Failed to send character: %c", char)
		// Brief pause between keystrokes to ensure they're processed
		time.Sleep(50 * time.Millisecond)
	}

	// Submit filter with Enter
	err = tf.SendKeys("\r")
	require.NoError(t, err, "Failed to submit filter")

	// Wait for filtering to take effect - wait for filter indicator to appear
	require.True(t, tf.OutputContains("[Filter: my]", 3*time.Second), "Filter indicator should appear")

	// Get filtered output
	filteredOutput := tf.SnapshotPlain()

	// Check if filter indicator is shown
	require.Contains(t, filteredOutput, "[Filter: my]", "Filter indicator should be visible")

	// Get the content after the filter indicator
	parts := strings.Split(filteredOutput, "[Filter: my]")
	if len(parts) < 2 {
		t.Fatal("Filter indicator not found in output")
	}

	afterFilter := parts[1]
	t.Logf("Content after 'my' filter: %s", afterFilter)

	// Verify filtering works with partial matches
	require.Contains(t, afterFilter, "my-project", "my-project should match 'my' filter")
	require.NotContains(t, afterFilter, "your-project", "your-project should not match 'my' filter")
}
