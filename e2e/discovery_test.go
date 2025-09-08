//go:build e2e && unix

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRepositoryDiscovery(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace with repositories
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	_, err = tf.CreateTestRepo("frontend-app")
	require.NoError(t, err, "Failed to create frontend-app repo")

	_, err = tf.CreateTestRepo("backend-api")
	require.NoError(t, err, "Failed to create backend-api repo")

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to signal ready
	require.True(t, tf.Ready(), "Should receive ready signal")

	// Wait for TUI to initialize and show content
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Get current buffered output
	output := tf.SnapshotPlain()

	// Only log summary with key indicators
	containsGitagrip := strings.Contains(output, "gitagrip")
	containsFrontend := strings.Contains(output, "frontend-app")
	containsBackend := strings.Contains(output, "backend-api")
	t.Logf("TUI output: %d chars, shows gitagrip=%v, frontend=%v, backend=%v",
		len(output), containsGitagrip, containsFrontend, containsBackend)

	// The app should have started without crashing
	require.Greater(t, len(output), 50, "Should produce substantial output indicating TUI is running")
	require.True(t, containsGitagrip, "Should show gitagrip title")
}

func TestRepositoryDiscoveryWithGroups(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace with grouped repositories
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create repos in a group structure
	for i := 1; i <= 3; i++ {
		_, err = tf.CreateTestRepo(fmt.Sprintf("api/service-%d", i))
		require.NoError(t, err, "Failed to create service repo")
	}

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Verify repos are discovered
	output := tf.SnapshotPlain()
	for i := 1; i <= 3; i++ {
		require.Contains(t, output, fmt.Sprintf("service-%d", i), "Should contain service repo")
	}
}
