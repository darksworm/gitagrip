package ui

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// GitOps handles git operations like log and diff
type GitOps struct{}

// NewGitOps creates a new GitOps instance
func NewGitOps() *GitOps {
	return &GitOps{}
}

// FetchGitLog fetches git log for a repository with branch/tag decorations
func (g *GitOps) FetchGitLog(repoPath string) (string, error) {
	// Run git log command with decorations for branch/tag info
	cmd := exec.Command("git", "log", "--oneline", "-12", "--color=always", "--decorate")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Truncate very long lines to prevent wrapping (preserve colors)
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		// Use visible width for truncation (accounting for ANSI color codes)
		visibleWidth := lipgloss.Width(line)
		if visibleWidth > 110 { // Allow more space for branch/tag decorations
			// Find a good break point by examining the string character by character
			bytes := []byte(line)
			visibleCount := 0
			byteIndex := 0

			for byteIndex < len(bytes) {
				if bytes[byteIndex] == '\x1b' { // ESC character starts ANSI sequence
					// Skip the entire ANSI sequence
					byteIndex++ // Skip ESC
					if byteIndex < len(bytes) && bytes[byteIndex] == '[' {
						byteIndex++ // Skip [
						for byteIndex < len(bytes) && bytes[byteIndex] != 'm' {
							byteIndex++
						}
						if byteIndex < len(bytes) {
							byteIndex++ // Skip m
						}
					}
				} else {
					visibleCount++
					byteIndex++

					// Break at 105 visible characters to preserve branch/tag info
					if visibleCount >= 105 {
						// Find the actual byte position for truncation
						truncateAt := byteIndex
						lines[i] = line[:truncateAt] + "..."
						break
					}
				}
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

// FetchGitDiff fetches git diff for a repository
func (g *GitOps) FetchGitDiff(repoPath string) (string, error) {
	// Run git diff command to show uncommitted changes
	cmd := exec.Command("git", "diff", "--color=always")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is the expected exit code 1 from git diff (indicating changes exist)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 with git diff means there are changes - this is expected
			return string(output), nil
		}
		// Any other error is a real problem
		return "", err
	}

	// No error means no changes (clean working directory)
	return string(output), nil
}

// HasUncommittedChanges checks if a repository has uncommitted changes
func (g *GitOps) HasUncommittedChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "diff", "--quiet")
	cmd.Dir = repoPath

	err := cmd.Run()
	if err != nil {
		// If git diff --quiet returns non-zero exit code, there are changes
		return true, nil
	}
	// Exit code 0 means no changes
	return false, nil
}
