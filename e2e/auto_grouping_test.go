//go:build e2e && unix

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutomaticGroupCreation(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create a directory structure with multiple repos in subdirectories
	// This should trigger automatic group creation

	// Create repos in "frontend" directory (should create a "frontend" group)
	frontendDir := filepath.Join(workspace, "frontend")
	require.NoError(t, os.MkdirAll(frontendDir, 0755))

	_, err = tf.CreateTestRepo("frontend/web-app")
	require.NoError(t, err, "Failed to create frontend/web-app repo")

	_, err = tf.CreateTestRepo("frontend/mobile-app")
	require.NoError(t, err, "Failed to create frontend/mobile-app repo")

	// Create repos in "backend" directory (should create a "backend" group)
	backendDir := filepath.Join(workspace, "backend")
	require.NoError(t, os.MkdirAll(backendDir, 0755))

	_, err = tf.CreateTestRepo("backend/api-server")
	require.NoError(t, err, "Failed to create backend/api-server repo")

	_, err = tf.CreateTestRepo("backend/auth-service")
	require.NoError(t, err, "Failed to create backend/auth-service repo")

	// Create a single repo in "utils" directory (should NOT create a group since it has only 1 repo)
	utilsDir := filepath.Join(workspace, "utils")
	require.NoError(t, os.MkdirAll(utilsDir, 0755))

	_, err = tf.CreateTestRepo("utils/helpers")
	require.NoError(t, err, "Failed to create utils/helpers repo")

	// Create a repo directly in workspace root (should NOT be in any group)
	_, err = tf.CreateTestRepo("standalone-repo")
	require.NoError(t, err, "Failed to create standalone-repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize and scan to complete
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Wait for repositories to be discovered and grouped
	require.True(t, tf.OutputContainsPlain("frontend", 10), "Should show frontend group")
	require.True(t, tf.OutputContainsPlain("backend", 10), "Should show backend group")

	// Expand the frontend group (navigate to it and press right arrow)
	tf.SendKeys("j") // Move down to frontend group
	time.Sleep(100 * time.Millisecond)
	tf.SendKeys("\x1b[C") // Right arrow key (ANSI escape sequence)
	time.Sleep(100 * time.Millisecond)

	// Expand the backend group (move down to it and press right arrow)
	tf.SendKeys("j") // Move down to backend group
	time.Sleep(100 * time.Millisecond)
	tf.SendKeys("\x1b[C") // Right arrow key (ANSI escape sequence)
	time.Sleep(100 * time.Millisecond)

	// Wait for groups to expand
	time.Sleep(500 * time.Millisecond)

	// Verify the groups contain the expected repositories
	output := tf.SnapshotPlain()

	// Check that frontend group contains both repos
	require.Contains(t, output, "web-app", "Should contain web-app in frontend group")
	require.Contains(t, output, "mobile-app", "Should contain mobile-app in frontend group")

	// Check that backend group contains both repos
	require.Contains(t, output, "api-server", "Should contain api-server in backend group")
	require.Contains(t, output, "auth-service", "Should contain auth-service in backend group")

	// Check that utils repo is present but not in a group (since it has only 1 repo)
	require.Contains(t, output, "helpers", "Should contain helpers repo")

	// Check that standalone repo is present
	require.Contains(t, output, "standalone-repo", "Should contain standalone repo")

	// Verify group counts - should have 2 groups (frontend and backend)
	groupCount := 0
	if tf.OutputContainsPlain("frontend", 1) {
		groupCount++
	}
	if tf.OutputContainsPlain("backend", 1) {
		groupCount++
	}
	require.Equal(t, 2, groupCount, "Should have exactly 2 groups created automatically")

	t.Logf("Automatic grouping test passed - created %d groups from directory structure", groupCount)
}
