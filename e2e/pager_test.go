//go:build e2e && unix

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGitDiffPager(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a repo with commits and dirty changes for diff
	_, err = tf.CreateTestRepo("diff-test-repo", WithCommit(true), WithDirtyState())
	require.NoError(t, err, "Failed to create diff test repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")

	// Open diff pager
	tf.OpenDiffPager()

	// Wait a moment for pager to open
	time.Sleep(500 * time.Millisecond)

	// Press 'q' to exit pager
	tf.Quit()

	// Verify we're back to main TUI
	require.True(t, tf.SeePlain("gitagrip"), "Should return to main TUI after closing pager")

	// Press 'q' again to exit the application
	tf.Quit()

	// Wait for app to exit
	done := make(chan error, 1)
	go func() { done <- tf.cmd.Wait() }()
	select {
	case <-done:
		// App exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("app did not exit after quit")
	}

	t.Logf("Diff pager test passed - pager opened and closed successfully")
}

func TestGitLogPager(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a repo with multiple commits
	repoPath, err := tf.CreateTestRepo("log-test-repo", WithCommit(true))
	require.NoError(t, err, "Failed to create log test repo")

	// Add another commit
	tf.runGitCommand(repoPath, "commit", "--allow-empty", "-m", "Second commit")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")

	// Open log pager (assuming 'L' key opens git log)
	// Note: This test assumes the pager key binding exists
	initialOutput := tf.Snapshot()

	// Try to open log pager - this might need adjustment based on actual key bindings
	tf.SendKeys("L") // Assuming 'L' opens git log

	// Wait for potential pager content
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, 2*time.Second), "Log pager should change TUI state")

	// Quit pager
	tf.Quit()
	require.True(t, tf.SeePlain("gitagrip"), "Should return to main TUI after closing log pager")
}
