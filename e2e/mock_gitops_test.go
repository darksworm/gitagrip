//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGitFetchTUIIntegration(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a simple repository with remote configured
	_, err = tf.CreateTestRepo("test-repo", WithRemote(), WithCommit(true))
	require.NoError(t, err, "Failed to create test repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repo to be discovered
	require.True(t, tf.OutputContainsPlain("test-repo", 5*time.Second), "Repo should be discovered")

	// Ensure the repository is selected (spacebar to select)
	err = tf.SendKeys(" ") // Select the repository
	require.NoError(t, err, "Failed to select repository")

	// Wait a moment for selection to register
	time.Sleep(500 * time.Millisecond)

	// Get initial TUI state after selection
	initialOutput := tf.SnapshotPlain()

	// Trigger fetch operation on the selected repository
	err = tf.Fetch()
	require.NoError(t, err, "Failed to send fetch command")

	// Wait for TUI to process the command
	time.Sleep(2 * time.Second)

	// Verify TUI responded to the command
	afterFetchOutput := tf.SnapshotPlain()
	require.NotEqual(t, initialOutput, afterFetchOutput, "TUI output should change after fetch command")

	// Verify repository is still visible
	require.True(t, tf.SeePlain("test-repo"), "Repository should still be visible after fetch")

	// Check for any status indicators that the command was processed
	output := tf.SnapshotPlain()
	if strings.Contains(output, "fetch") || strings.Contains(output, "Fetching") ||
		strings.Contains(output, "completed") || strings.Contains(output, "failed") {
		t.Logf("✅ Found status message indicating fetch command was processed")
	} else {
		t.Logf("ℹ️  No explicit status message found, but TUI responded to command")
	}

	t.Logf("✅ TUI Integration Test PASSED:")
	t.Logf("   - Fetch command sent successfully: ✅")
	t.Logf("   - TUI processed command: ✅")
	t.Logf("   - Repository remained visible: ✅")
	t.Logf("   - No application crashes: ✅")
}

/*
NOTE: Full end-to-end testing of git fetch/pull operations requires:

1. **GitService Initialization**: The GitService must be running to handle FetchRequestedEvent
2. **Event Bus Connection**: Proper event bus setup between TUI and GitService
3. **Repository Discovery**: The repository must be discovered by the discovery service
4. **Remote Repository**: A working git remote (local bare repo or actual remote server)

For a complete end-to-end test, you would need to:
- Initialize the full application stack (discovery, git service, event bus)
- Set up a proper remote repository with commits ahead of local
- Verify that fetch/pull actually retrieves the commits
- Check that the local repository state is updated correctly

This test focuses on TUI integration - verifying that the interface correctly
responds to user commands and maintains stability during git operations.
*/

func TestGitPullUIInteraction(t *testing.T) {
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

	// Get initial TUI output
	initialOutput := tf.SnapshotPlain()

	// Trigger pull operation
	err = tf.Pull()
	require.NoError(t, err, "Failed to send pull command")

	// Wait for command processing
	time.Sleep(1 * time.Second)

	// Verify TUI responded
	afterPullOutput := tf.SnapshotPlain()
	require.NotEqual(t, initialOutput, afterPullOutput, "TUI output should change after pull command")

	// Verify repo is still visible
	require.True(t, tf.SeePlain("pull-test-repo"), "Repository should still be visible after pull")

	t.Logf("✅ Pull UI interaction test passed - TUI properly handled pull command")
}

func TestGitBulkOperations(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create multiple repos with remotes
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

	// Select both repos
	tf.SendKeys(" ") // Select repo1
	time.Sleep(100 * time.Millisecond)
	tf.SendKeys("j ") // Navigate to repo2 and select it
	time.Sleep(100 * time.Millisecond)

	// Get output before bulk operation
	beforeBulkOutput := tf.SnapshotPlain()

	// Trigger fetch on selected repos
	err = tf.Fetch()
	require.NoError(t, err, "Failed to send fetch command")

	// Wait for operations to complete
	time.Sleep(2 * time.Second)

	// Verify TUI still shows both repos
	require.True(t, tf.SeePlain("repo1"), "repo1 should still be visible")
	require.True(t, tf.SeePlain("repo2"), "repo2 should still be visible")

	// Verify output changed (indicating operations were processed)
	afterBulkOutput := tf.SnapshotPlain()
	require.NotEqual(t, beforeBulkOutput, afterBulkOutput, "TUI output should change after bulk operations")

	t.Logf("✅ Bulk operations test passed - TUI handled operations on multiple selected repos")
}
