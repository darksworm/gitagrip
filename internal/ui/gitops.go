package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/noborus/ov/oviewer"
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
	return true
}

// configureVimKeyBindings adds vim-like key bindings to the oviewer config
func configureVimKeyBindings(config *oviewer.Config) {
	// Clear existing key bindings to avoid conflicts
	config.Keybind = make(map[string][]string)

	// Basic movement
	config.Keybind["down"] = append(config.Keybind["down"], "j")
	config.Keybind["up"] = append(config.Keybind["up"], "k")
	config.Keybind["left"] = append(config.Keybind["left"], "h")
	config.Keybind["right"] = append(config.Keybind["right"], "l")

	// Page movement (vim-style)
	config.Keybind["page_down"] = append(config.Keybind["page_down"], "ctrl+f")
	config.Keybind["page_up"] = append(config.Keybind["page_up"], "ctrl+b")
	config.Keybind["page_half_down"] = append(config.Keybind["page_half_down"], "ctrl+d")
	config.Keybind["page_half_up"] = append(config.Keybind["page_half_up"], "ctrl+u")

	// Jump to position
	config.Keybind["top"] = append(config.Keybind["top"], "g", "g")
	config.Keybind["bottom"] = append(config.Keybind["bottom"], "G")

	// Line navigation
	config.Keybind["begin_left"] = append(config.Keybind["begin_left"], "0", "^")
	config.Keybind["end_right"] = append(config.Keybind["end_right"], "$")

	// Word navigation - using existing half_left/half_right for word movement
	config.Keybind["half_left"] = append(config.Keybind["half_left"], "b")
	config.Keybind["half_right"] = append(config.Keybind["half_right"], "w")

	// Search
	config.Keybind["search"] = append(config.Keybind["search"], "/")
	config.Keybind["backsearch"] = append(config.Keybind["backsearch"], "?")
	config.Keybind["next_search"] = append(config.Keybind["next_search"], "n")
	config.Keybind["next_backsearch"] = append(config.Keybind["next_backsearch"], "N")

	// Quit
	config.Keybind["exit"] = append(config.Keybind["exit"], "q", "ctrl+c")
}

// ShowGitLogInPager shows git log using ov pager
func (g *GitOps) ShowGitLogInPager(repoPath string) error {
	if g.program == nil {
		return fmt.Errorf("program not set")
	}

	// Release terminal control to run ov
	if err := g.program.ReleaseTerminal(); err != nil {
		return err
	}

	// Ensure terminal is restored even if ov fails
	defer func() {
		// Clear screen to prevent flash of previous content
		fmt.Print("\x1b[2J\x1b[H") // Clear screen and move cursor to top-left
		// Small delay to ensure ov has fully exited before restoring terminal
		time.Sleep(150 * time.Millisecond)
		_ = g.program.RestoreTerminal() // Ignore error as we're in defer context
	}()

	// Create git log command
	gitCmd := exec.Command("git", "log", "--oneline", "-20", "--color=always", "--decorate")
	gitCmd.Dir = repoPath

	// Get stdout pipe
	stdout, err := gitCmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Start the command
	if err := gitCmd.Start(); err != nil {
		return err
	}

	// Create oviewer root from the stdout reader
	root, err := oviewer.NewRoot(stdout)
	if err != nil {
		return err
	}

	// Configure ov to not write on exit (to avoid messing with our screen)
	config := oviewer.NewConfig()
	config.IsWriteOnExit = false
	config.IsWriteOriginal = false

	// Add vim-like navigation
	configureVimKeyBindings(&config)

	root.SetConfig(config)

	// Run the oviewer (this will take over the terminal)
	return root.Run()
}

// ShowGitDiffInPager shows git diff using ov pager
func (g *GitOps) ShowGitDiffInPager(repoPath string) error {
	if g.program == nil {
		return fmt.Errorf("program not set")
	}

	// Release terminal control to run ov
	if err := g.program.ReleaseTerminal(); err != nil {
		return err
	}

	// Ensure terminal is restored even if ov fails
	defer func() {
		// Clear screen to prevent flash of previous content
		fmt.Print("\x1b[2J\x1b[H") // Clear screen and move cursor to top-left
		// Small delay to ensure ov has fully exited before restoring terminal
		time.Sleep(150 * time.Millisecond)
		_ = g.program.RestoreTerminal() // Ignore error as we're in defer context
	}()

	// Create git diff command
	gitCmd := exec.Command("git", "diff", "--color=always")
	gitCmd.Dir = repoPath

	// Get stdout pipe
	stdout, err := gitCmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Start the command
	if err := gitCmd.Start(); err != nil {
		return err
	}

	// Create oviewer root from the stdout reader
	root, err := oviewer.NewRoot(stdout)
	if err != nil {
		return err
	}

	// Configure ov to not write on exit (to avoid messing with our screen)
	config := oviewer.NewConfig()
	config.IsWriteOnExit = false
	config.IsWriteOriginal = false

	// Add vim-like navigation
	configureVimKeyBindings(&config)

	root.SetConfig(config)

	// Run the oviewer (this will take over the terminal)
	return root.Run()
}

// ShowHelpInPager shows help content using ov pager
func (g *GitOps) ShowHelpInPager(helpContent string) error {
	if g.program == nil {
		return fmt.Errorf("program not set")
	}

	// Release terminal control to run ov
	if err := g.program.ReleaseTerminal(); err != nil {
		return err
	}

	// Ensure terminal is restored even if ov fails
	defer func() {
		// Clear screen to prevent flash of previous content
		fmt.Print("\x1b[2J\x1b[H") // Clear screen and move cursor to top-left
		// Small delay to ensure ov has fully exited before restoring terminal
		time.Sleep(150 * time.Millisecond)
		_ = g.program.RestoreTerminal() // Ignore error as we're in defer context
	}()

	// Create a reader from the help content string
	reader := strings.NewReader(helpContent)

	// Create oviewer root from the reader
	root, err := oviewer.NewRoot(reader)
	if err != nil {
		return err
	}

	// Configure ov to not write on exit (to avoid messing with our screen)
	config := oviewer.NewConfig()
	config.IsWriteOnExit = false
	config.IsWriteOriginal = false

	// Add vim-like navigation
	configureVimKeyBindings(&config)

	root.SetConfig(config)

	// Run the oviewer (this will take over the terminal)
	return root.Run()
}
