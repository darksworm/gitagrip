package ui

import (
    "fmt"
    "io"
    "os"
    "os/exec"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// GitOps handles git operations like log and diff
type GitOps struct {
	program *tea.Program // reference to Bubble Tea program for terminal management
}

// NewGitOps creates a new GitOps instance
func NewGitOps() *GitOps {
	return &GitOps{}
}

// SetProgram sets the program reference for terminal management
func (g *GitOps) SetProgram(p *tea.Program) {
	g.program = p
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

// IsOvAvailable checks if the ov pager is available (always true since we use the library)
func (g *GitOps) IsOvAvailable() bool {
    // Treat pager availability as presence of `less`
    _, err := exec.LookPath("less")
    return err == nil
}

// IsLazygitAvailable checks if the lazygit binary is available
func (g *GitOps) IsLazygitAvailable() bool {
	// Allow overriding the binary path for testing via env var
	if path := os.Getenv("GITAGRIP_LAZYGIT_BIN"); path != "" {
		if _, err := exec.LookPath(path); err == nil {
			return true
		}
		// If absolute path was provided but not in PATH, it might still be executable
		if _, err := os.Stat(path); err == nil {
			return true
		}
		return false
	}
	_, err := exec.LookPath("lazygit")
	return err == nil
}

// RunLazygit launches the lazygit TUI for the given repository
func (g *GitOps) RunLazygit(repoPath string) error {
	if g.program == nil {
		return fmt.Errorf("program not set")
	}

	// Determine binary
	bin := os.Getenv("GITAGRIP_LAZYGIT_BIN")
	if bin == "" {
		bin = "lazygit"
	}

	// Release terminal control to run external program
	if err := g.program.ReleaseTerminal(); err != nil {
		return err
	}
	defer func() {
		// Clear screen to reduce visual artifacts when returning
		fmt.Print("\x1b[2J\x1b[H")
		time.Sleep(150 * time.Millisecond)
		_ = g.program.RestoreTerminal()
	}()

	// Spawn lazygit with working directory set to repo
	cmd := exec.Command(bin)
	cmd.Dir = repoPath
	// Inherit stdio so it fully takes over the terminal
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Pager integration: we use external `less -R` and no longer embed a pager

// ShowGitLogInPager shows git log using ov pager
func (g *GitOps) ShowGitLogInPager(repoPath string) error {
    if g.program == nil {
        return fmt.Errorf("program not set")
    }
    gitCmd := exec.Command("git", "log", "--oneline", "-20", "--color=always", "--decorate")
    gitCmd.Dir = repoPath
    pr, pw := io.Pipe()
    gitCmd.Stdout = pw
    gitCmd.Stderr = os.Stderr
    if err := gitCmd.Start(); err != nil {
        _ = pw.Close()
        _ = pr.Close()
        return err
    }
    pagerErrCh := make(chan error, 1)
    go func() { pagerErrCh <- g.runPager(pr) }()
    gitErr := gitCmd.Wait()
    _ = pw.Close()
    pagerErr := <-pagerErrCh
    if gitErr != nil {
        return gitErr
    }
    return pagerErr
}

// ShowGitDiffInPager shows git diff using ov pager
func (g *GitOps) ShowGitDiffInPager(repoPath string) error {
    if g.program == nil {
        return fmt.Errorf("program not set")
    }
    gitCmd := exec.Command("git", "diff", "--color=always")
    gitCmd.Dir = repoPath
    pr, pw := io.Pipe()
    gitCmd.Stdout = pw
    gitCmd.Stderr = os.Stderr
    if err := gitCmd.Start(); err != nil {
        _ = pw.Close()
        _ = pr.Close()
        return err
    }
    pagerErrCh := make(chan error, 1)
    go func() { pagerErrCh <- g.runPager(pr) }()
    gitErr := gitCmd.Wait()
    _ = pw.Close()
    pagerErr := <-pagerErrCh
    if gitErr != nil {
        if ee, ok := gitErr.(*exec.ExitError); ok && ee.ExitCode() == 1 {
            // git diff returns 1 when there are changes; treat as success
        } else {
            return gitErr
        }
    }
    return pagerErr
}

// ShowHelpInPager shows help content using ov pager
func (g *GitOps) ShowHelpInPager(helpContent string) error {
    reader := strings.NewReader(helpContent)
    return g.runPager(reader)
}

// runPager executes `less -R`, feeding content via r, handling terminal release/restore
func (g *GitOps) runPager(r io.Reader) error {
    if g.program == nil {
        return fmt.Errorf("program not set")
    }
    if _, err := exec.LookPath("less"); err != nil {
        return fmt.Errorf("less not found in PATH")
    }
    if err := g.program.ReleaseTerminal(); err != nil {
        return err
    }
    defer func() {
        fmt.Print("\x1b[2J\x1b[H")
        time.Sleep(150 * time.Millisecond)
        _ = g.program.RestoreTerminal()
    }()
    lessCmd := exec.Command("less", "-R")
    lessCmd.Stdin = r
    lessCmd.Stdout = os.Stdout
    lessCmd.Stderr = os.Stderr
    return lessCmd.Run()
}
