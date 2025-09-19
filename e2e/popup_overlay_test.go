//go:build e2e && unix

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHelpPager(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a test repo
	_, err = tf.CreateTestRepo("test-repo", WithDirtyState())
	require.NoError(t, err, "Failed to create test repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Get initial state
	initialOutput := tf.Snapshot()
	require.Greater(t, len(initialOutput), 100, "Should have initial TUI content")

	// Open help pager (? key)
	tf.SendKeys("?")

	// Wait for pager to open (output should change)
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, 2*time.Second), "Help pager should open and change TUI state")

	// Press 'q' to exit pager
	tf.Quit()

	// Verify we're back to main TUI
	require.True(t, tf.SeePlain("gitagrip"), "Should return to main TUI after closing help pager")
	require.True(t, tf.SeePlain("test-repo"), "Should show repository again")

	t.Logf("Help pager test passed - pager opened and closed successfully")
}
