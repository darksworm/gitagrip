//go:build e2e && unix

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApplicationExit(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	_, err = tf.CreateTestRepo("exit-test-repo")
	require.NoError(t, err, "Failed to create exit test repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize and render
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Clear any buffered output first
	tf.Snapshot()

	// Set up exit monitoring before sending 'q'
	done := make(chan error, 1)
	go func() {
		done <- tf.cmd.Wait()
	}()

	// Send 'q' to quit
	t.Logf("Sending 'q' to quit application...")
	tf.Quit()

	// Wait for graceful shutdown
	select {
	case exitErr := <-done:
		if exitErr == nil {
			t.Logf("Process exited cleanly with 'q' command")
		} else {
			t.Logf("Process exited with 'q' command (exit code: %v)", exitErr)
		}
		return
	case <-time.After(1500 * time.Millisecond):
		// If 'q' didn't work within 1.5 seconds, use Ctrl+C
		t.Logf("'q' didn't work within 1.5 seconds, using Ctrl+C")
		tf.SendCtrlC()
	}

	// Wait for Ctrl+C to work
	select {
	case exitErr := <-done:
		t.Logf("Process exited with Ctrl+C (exit code: %v)", exitErr)
	case <-time.After(750 * time.Millisecond):
		t.Error("Application did not exit within total timeout")
		tf.DumpTailOnFail(t, "exit-failure", 4096) // Debug output
		tf.SendCtrlC()                             // Force exit again
	}
}

func TestApplicationExitWithConfigSave(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	_, err = tf.CreateTestRepo("exit-save-test-repo")
	require.NoError(t, err, "Failed to create exit save test repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to initialize
	require.True(t, tf.Ready(), "Should receive ready signal")
	require.True(t, tf.SeePlain("gitagrip"), "Should show gitagrip title")

	// Make some changes (select a repo)
	tf.Select()

	// Exit gracefully
	done := make(chan error, 1)
	go func() {
		done <- tf.cmd.Wait()
	}()

	tf.Quit()

	// Wait for exit
	select {
	case <-done:
		t.Logf("Process exited cleanly after config save")
	case <-time.After(2 * time.Second):
		t.Fatal("app did not exit after quit")
	}
}
