//go:build e2e && unix

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGitFetchOperation(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a repo with remote configured
	_, err = tf.CreateTestRepo("fetch-test-repo", WithRemote(), WithCommit(true))
	require.NoError(t, err, "Failed to create test repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repo to be discovered
	require.True(t, tf.OutputContainsPlain("fetch-test-repo", 5*time.Second), "Repo should be discovered")

	// Get initial output
	initialOutput := tf.SnapshotPlain()

	// Trigger fetch operation
	err = tf.Fetch()
	require.NoError(t, err, "Failed to send fetch command")

	// Wait for fetch operation to be acknowledged (UI should change)
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, 3*time.Second), "UI should respond to fetch command")

	// Verify that some status indication appears
	// (We don't test actual git success since we're not mocking the remote)
	output := tf.SnapshotPlain()
	require.Contains(t, output, "fetch-test-repo", "Repo should still be visible")

	t.Logf("✅ Fetch operation test passed - UI responded to fetch command")
}

func TestGitPullOperation(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a repo with remote configured
	_, err = tf.CreateTestRepo("pull-test-repo", WithRemote(), WithCommit(true))
	require.NoError(t, err, "Failed to create test repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repo to be discovered
	require.True(t, tf.OutputContainsPlain("pull-test-repo", 5*time.Second), "Repo should be discovered")

	// Get initial output
	initialOutput := tf.SnapshotPlain()

	// Trigger pull operation
	err = tf.Pull()
	require.NoError(t, err, "Failed to send pull command")

	// Wait for pull operation to be acknowledged
	require.True(t, tf.WaitFor(func(s string) bool {
		return s != initialOutput
	}, 3*time.Second), "UI should respond to pull command")

	// Verify that repo is still visible
	output := tf.SnapshotPlain()
	require.Contains(t, output, "pull-test-repo", "Repo should still be visible")

	t.Logf("✅ Pull operation test passed - UI responded to pull command")
}

func TestGitOperationsWithSelection(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create multiple repos
	_, err = tf.CreateTestRepo("repo1", WithRemote(), WithCommit(true))
	require.NoError(t, err)
	_, err = tf.CreateTestRepo("repo2", WithRemote(), WithCommit(true))
	require.NoError(t, err)

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	require.True(t, tf.Ready(), "Should receive ready signal")

	// Wait for repos to be discovered
	require.True(t, tf.OutputContainsPlain("repo1", 5*time.Second), "repo1 should be discovered")
	require.True(t, tf.OutputContainsPlain("repo2", 5*time.Second), "repo2 should be discovered")

	// Select first repo
	tf.SendKeys(" ")
	time.Sleep(500 * time.Millisecond)

	// Select second repo
	tf.SendKeys("j ")
	time.Sleep(500 * time.Millisecond)

	// Get output after selections
	selectedOutput := tf.SnapshotPlain()

	// Trigger fetch on selected repos
	err = tf.Fetch()
	require.NoError(t, err, "Failed to send fetch command")

	// Wait for operation to be acknowledged
	require.True(t, tf.WaitFor(func(s string) bool {
		current := tf.SnapshotPlain()
		return current != selectedOutput
	}, 3*time.Second), "UI should respond to fetch on selected repos")

	// Verify repos are still visible
	finalOutput := tf.SnapshotPlain()
	require.Contains(t, finalOutput, "repo1", "repo1 should still be visible")
	require.Contains(t, finalOutput, "repo2", "repo2 should still be visible")

	t.Logf("✅ Bulk operations test passed - UI handled operations on selected repos")
}
