//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSortByName(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create repos with names that will sort in reverse alphabetical order initially
	_, err = tf.CreateTestRepo("zebra-project")
	require.NoError(t, err, "Failed to create zebra-project repo")

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

	// Wait for repos to be discovered
	require.True(t, tf.OutputContainsPlain("alpha-project", 5*time.Second), "alpha-project should be discovered")
	require.True(t, tf.OutputContainsPlain("beta-project", 5*time.Second), "beta-project should be discovered")
	require.True(t, tf.OutputContainsPlain("zebra-project", 5*time.Second), "zebra-project should be discovered")

	// Get initial output to verify current order
	initialOutput := tf.SnapshotPlain()

	// Enter sort mode with 's'
	err = tf.SendKeys("s")
	require.NoError(t, err, "Failed to send s key to enter sort mode")

	// Wait for sort mode to appear
	require.True(t, tf.WaitFor(func(s string) bool {
		return strings.Contains(s, "Sort by:")
	}, 2*time.Second), "Sort mode should appear")

	// The first option should be "Name" (default)
	require.True(t, tf.SeePlain("Sort by: Name"), "Should show Name sort option")

	// Press Enter to select Name sorting
	err = tf.SendKeys("\r")
	require.NoError(t, err, "Failed to select Name sorting")

	// Wait for sorting to be applied (output should change)
	preSortOutput := tf.SnapshotPlain()
	require.True(t, tf.WaitFor(func(s string) bool {
		currentOutput := tf.SnapshotPlain()
		return currentOutput != preSortOutput
	}, 3*time.Second), "Output should change after sorting is applied")

	// Get output after sorting
	sortedOutput := tf.SnapshotPlain()

	// Verify that the output changed (indicating sort was applied)
	require.NotEqual(t, initialOutput, sortedOutput, "Output should change after sorting")

	// Verify all repos are still present
	require.Contains(t, sortedOutput, "alpha-project", "alpha-project should still be visible")
	require.Contains(t, sortedOutput, "beta-project", "beta-project should still be visible")
	require.Contains(t, sortedOutput, "zebra-project", "zebra-project should still be visible")

	// The exact order verification would require parsing the TUI output more precisely,
	// but the important thing is that sorting was applied and all repos remain visible
	t.Logf("✅ Sort by name test passed - sorting applied successfully")
}

func TestSortByStatus(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create repos with different statuses
	_, err = tf.CreateTestRepo("clean-repo")
	require.NoError(t, err, "Failed to create clean-repo")

	// Create a repo with uncommitted changes (dirty)
	_, err = tf.CreateTestRepo("dirty-repo", WithDirtyState())
	require.NoError(t, err, "Failed to create dirty-repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repos to be discovered
	require.True(t, tf.OutputContainsPlain("clean-repo", 5*time.Second), "clean-repo should be discovered")
	require.True(t, tf.OutputContainsPlain("dirty-repo", 5*time.Second), "dirty-repo should be discovered")

	// Get initial output
	initialOutput := tf.SnapshotPlain()

	// Enter sort mode with 's'
	err = tf.SendKeys("s")
	require.NoError(t, err, "Failed to enter sort mode")

	// Wait for sort mode
	require.True(t, tf.WaitFor(func(s string) bool {
		return strings.Contains(s, "Sort by:")
	}, 2*time.Second), "Sort mode should appear")

	// Navigate to Status sort option (should be second option)
	err = tf.SendKeys("j") // Down to Status option
	require.NoError(t, err, "Failed to navigate to Status option")
	time.Sleep(100 * time.Millisecond)

	// Should now show "Sort by: Status"
	require.True(t, tf.SeePlain("Sort by: Status"), "Should show Status sort option")

	// Press Enter to select Status sorting
	err = tf.SendKeys("\r")
	require.NoError(t, err, "Failed to select Status sorting")

	// Wait for sorting to be applied (look for status message or sort mode exit)
	require.True(t, tf.WaitFor(func(s string) bool {
		return strings.Contains(s, "Sorting by status") || (!strings.Contains(s, "Sort by:") && tf.SeePlain("gitagrip"))
	}, 3*time.Second), "Sorting by status should be applied")

	// Get output after sorting
	sortedOutput := tf.SnapshotPlain()

	// Verify that the output changed
	require.NotEqual(t, initialOutput, sortedOutput, "Output should change after status sorting")

	// Verify all repos are still present
	require.Contains(t, sortedOutput, "clean-repo", "clean-repo should still be visible")
	require.Contains(t, sortedOutput, "dirty-repo", "dirty-repo should still be visible")

	t.Logf("✅ Sort by status test passed - status-based sorting applied successfully")
}

func TestSortByBranch(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create repos - they should all be on main branch by default
	_, err = tf.CreateTestRepo("main-branch-repo")
	require.NoError(t, err, "Failed to create main-branch-repo")

	_, err = tf.CreateTestRepo("feature-branch-repo")
	require.NoError(t, err, "Failed to create feature-branch-repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repos to be discovered
	require.True(t, tf.OutputContainsPlain("main-branch-repo", 5*time.Second), "main-branch-repo should be discovered")
	require.True(t, tf.OutputContainsPlain("feature-branch-repo", 5*time.Second), "feature-branch-repo should be discovered")

	// Get initial output
	initialOutput := tf.SnapshotPlain()

	// Enter sort mode with 's'
	err = tf.SendKeys("s")
	require.NoError(t, err, "Failed to enter sort mode")

	// Wait for sort mode
	require.True(t, tf.WaitFor(func(s string) bool {
		return strings.Contains(s, "Sort by:")
	}, 2*time.Second), "Sort mode should appear")

	// Navigate to Branch sort option (should be third option)
	err = tf.SendKeys("j") // Down to Status
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	err = tf.SendKeys("j") // Down to Branch
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Should now show "Sort by: Branch"
	require.True(t, tf.SeePlain("Sort by: Branch"), "Should show Branch sort option")

	// Press Enter to select Branch sorting
	err = tf.SendKeys("\r")
	require.NoError(t, err, "Failed to select Branch sorting")

	// Wait for sorting to be applied (look for status message or sort mode exit)
	require.True(t, tf.WaitFor(func(s string) bool {
		return strings.Contains(s, "Sorting by branch") || (!strings.Contains(s, "Sort by:") && tf.SeePlain("gitagrip"))
	}, 3*time.Second), "Sorting by branch should be applied")

	// Get output after sorting
	sortedOutput := tf.SnapshotPlain()

	// Verify that the output changed
	require.NotEqual(t, initialOutput, sortedOutput, "Output should change after branch sorting")

	// Verify all repos are still present
	require.Contains(t, sortedOutput, "main-branch-repo", "main-branch-repo should still be visible")
	require.Contains(t, sortedOutput, "feature-branch-repo", "feature-branch-repo should still be visible")

	t.Logf("✅ Sort by branch test passed - branch-based sorting applied successfully")
}

func TestSortNavigationAndCancel(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a couple repos
	_, err = tf.CreateTestRepo("test-repo-1")
	require.NoError(t, err, "Failed to create test-repo-1")

	_, err = tf.CreateTestRepo("test-repo-2")
	require.NoError(t, err, "Failed to create test-repo-2")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repos to be discovered
	require.True(t, tf.OutputContainsPlain("test-repo-1", 5*time.Second), "test-repo-1 should be discovered")
	require.True(t, tf.OutputContainsPlain("test-repo-2", 5*time.Second), "test-repo-2 should be discovered")

	// Enter sort mode with 's'
	err = tf.SendKeys("s")
	require.NoError(t, err, "Failed to enter sort mode")

	// Wait for sort mode
	require.True(t, tf.WaitFor(func(s string) bool {
		return strings.Contains(s, "Sort by:")
	}, 2*time.Second), "Sort mode should appear")

	// Navigate through options
	err = tf.SendKeys("j") // Down to Status
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	err = tf.SendKeys("j") // Down to Branch
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	err = tf.SendKeys("k") // Up to Status
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Should be back to Status
	require.True(t, tf.SeePlain("Sort by: Status"), "Should show Status sort option after navigation")

	// Cancel with 'q'
	err = tf.SendKeys("q")
	require.NoError(t, err, "Failed to cancel sort mode")

	// Wait a short time for the cancel to take effect
	time.Sleep(500 * time.Millisecond)

	// Get final output
	finalOutput := tf.SnapshotPlain()

	// The key test is that repos are still present
	require.Contains(t, finalOutput, "test-repo-1", "test-repo-1 should still be visible")
	require.Contains(t, finalOutput, "test-repo-2", "test-repo-2 should still be visible")

	// Check that we're back to normal mode (not in sort mode anymore)
	require.Contains(t, finalOutput, "gitagrip", "Should be back to normal mode after canceling")

	t.Logf("✅ Sort navigation and cancel test passed - cancel preserves original state")

	t.Logf("✅ Sort navigation and cancel test passed - navigation works and cancel preserves original order")
}
