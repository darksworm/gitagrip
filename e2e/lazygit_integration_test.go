//go:build e2e && unix

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestLazygitSpawn verifies that pressing Enter on a repository spawns the lazygit process
// We stub the lazygit binary using an environment override so the test does not require lazygit installed.
func TestLazygitSpawn(t *testing.T) {
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create workspace and a simple repo
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "create workspace")
	_, err = tf.CreateTestRepo("lg-repo", WithCommit(true))
	require.NoError(t, err, "create repo")

	// Create a stub lazygit binary that prints to stdout and sleeps briefly
	stubPath := filepath.Join(workspace, "lazygit-stub.sh")
	stubScript := "#!/bin/sh\n" +
		"echo LAZYGIT_MOCK START in $(pwd)\n" +
		"sleep 0.5\n" +
		"echo LAZYGIT_MOCK END\n"
	require.NoError(t, os.WriteFile(stubPath, []byte(stubScript), 0755))

	// Override the binary for this test process only
	t.Setenv("GITAGRIP_LAZYGIT_BIN", stubPath)

	// Start the app scanning the workspace
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "start app")

	// Wait for ready signal and title
	require.True(t, tf.Ready(), "app should report ready")
	require.True(t, tf.SeePlain("gitagrip"), "title should be visible")
	require.True(t, tf.SeePlain("lg-repo"), "repo should be visible")

	// Move from header to the repo row if needed
	_ = tf.Down()

	// Capture initial output, then press Enter to launch lazygit
	initial := tf.Snapshot()
	err = tf.SendEnter()
	require.NoError(t, err, "send enter")

	// The TUI should hand off the terminal, changing the output
	require.True(t, tf.WaitFor(func(s string) bool { return s != initial }, 2*time.Second), "should see output change when launching lazygit")

	// After it exits, we should be back at the main TUI
	require.True(t, tf.SeePlain("gitagrip"), "should return to main TUI after lazygit exits")
}
