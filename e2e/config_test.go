//go:build e2e && unix

package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigFileCreation(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	_, err = tf.CreateTestRepo("config-test-repo")
	require.NoError(t, err, "Failed to create config test repo")

	configPath := filepath.Join(workspace, ".gitagrip.toml")

	// Ensure no config exists initially
	require.NoError(t, os.Remove(configPath)) // Ignore error if file doesn't exist

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Exit gracefully
	tf.Quit()

	// Wait for app to exit (process should terminate)
	done := make(chan error, 1)
	go func() { done <- tf.cmd.Wait() }()
	select {
	case <-done:
		// App exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("app did not exit after quit")
	}

	// Check if config file was created
	_, err = os.Stat(configPath)
	require.NoError(t, err, "Config file should be created")

	if err == nil {
		configContent, err := os.ReadFile(configPath)
		require.NoError(t, err, "Should be able to read config file")

		configStr := string(configContent)
		require.Contains(t, configStr, "version = 1", "Config should contain version")
		require.Contains(t, configStr, workspace, "Config should contain workspace path")

		t.Logf("Config file created: %d chars, version=1, workspace included", len(configStr))
	}
}

func TestConfigFilePersistence(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create initial config
	configPath := filepath.Join(workspace, ".gitagrip.toml")
	initialConfig := `version = 1
base_dir = "` + workspace + `"
groups = {}
group_order = []`
	require.NoError(t, os.WriteFile(configPath, []byte(initialConfig), 0644))

	_, err = tf.CreateTestRepo("persistence-test-repo")
	require.NoError(t, err, "Failed to create persistence test repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Exit gracefully
	tf.Quit()

	// Wait for app to exit
	done := make(chan error, 1)
	go func() { done <- tf.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("app did not exit after quit")
	}

	// Verify config still exists and is valid
	_, err = os.Stat(configPath)
	require.NoError(t, err, "Config file should still exist")

	configContent, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read config file")
	require.Contains(t, string(configContent), "version = 1", "Config should be preserved")
}
