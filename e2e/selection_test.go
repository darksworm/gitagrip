//go:build e2e && unix

package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRepositorySelection(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	_, err = tf.CreateTestRepo("selectable-repo")
	require.NoError(t, err, "Failed to create selectable repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Get initial output
	initialOutput := tf.Snapshot()

	// Try to select with spacebar
	tf.Select()

	// Wait for selection to take effect
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, time.Second), "Selection should change output")

	output := tf.Snapshot()
	t.Logf("Selection test: output %d chars, app responsive", len(output))

	// Basic test that the app is running
	require.Greater(t, len(output), 100, "App should be running with TUI content")
}

func TestRepositorySelectionMultiple(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create multiple repos
	for i := 1; i <= 3; i++ {
		_, err = tf.CreateTestRepo(fmt.Sprintf("multi-select-%d", i))
		require.NoError(t, err, "Failed to create multi-select repo")
	}

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Test multiple selections
	initialOutput := tf.Snapshot()

	// Select first repo
	tf.Select()
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, time.Second), "First selection should change output")

	firstSelectOutput := tf.Snapshot()

	// Navigate and select second repo
	tf.Down()
	tf.Select()
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != firstSelectOutput
	}, time.Second), "Second selection should change output")

	finalOutput := tf.Snapshot()
	require.NotEqual(t, initialOutput, finalOutput, "Multiple selections should change TUI state")
}
