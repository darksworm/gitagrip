//go:build e2e && unix

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpCommand(t *testing.T) {
	t.Parallel()

	// Ensure the test binary exists (it should be built by TestMain)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skip("Test binary not found - TestMain may not have run yet")
	}

	// Test help command by running it directly (not through PTY since it exits quickly)
	cmd := exec.Command(binPath, "--help")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Help command should run without error")

	output := string(out)
	t.Logf("Help output length: %d chars", len(output))

	// Verify we got some meaningful output
	require.Greater(t, len(output), 50, "Help should produce substantial output")

	// Check for key help elements (be more flexible with the text)
	require.True(t,
		strings.Contains(output, "Usage") ||
			strings.Contains(output, "usage") ||
			strings.Contains(output, "help"),
		"Help should contain usage or help information")

	require.True(t,
		strings.Contains(output, "Directory") ||
			strings.Contains(output, "directory") ||
			strings.Contains(output, "-d"),
		"Help should contain directory option information")

	t.Logf("Help command test passed - output contains expected content")
}
