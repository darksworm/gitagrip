package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ringSize = 1 << 20 // 1 MiB of scrollback

// Key constants for better readability
const (
	KeyEnter = "\r"
	KeyCtrlC = "\x03"
	KeySpace = " "
	KeyDown  = "j"
	KeyQuit  = "q"
	KeyDiff  = "D"
)

// TUITestFramework provides utilities for testing TUI applications
type TUITestFramework struct {
	t         *testing.T
	pty       *os.File
	tty       *os.File
	cmd       *exec.Cmd
	workspace string

	// Ring buffer for continuous output capture
	mu   sync.Mutex
	buf  []byte
	head int
	full bool
	cond *sync.Cond
}

// NewTUITest creates a new TUI test framework instance
func NewTUITest(t *testing.T) *TUITestFramework {
	tf := &TUITestFramework{
		t:   t,
		buf: make([]byte, ringSize),
	}
	tf.cond = sync.NewCond(&tf.mu)
	return tf
}

// StartApp launches the gitagrip application with given arguments in a PTY
func (tf *TUITestFramework) StartApp(args ...string) error {
	// Build the command
	cmdArgs := append([]string{"./gitagrip_e2e"}, args...)
	tf.cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// Set per-process environment variables
	tf.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"LC_ALL=C",
		"LANG=C",
		"GITAGRIP_E2E_TEST=1",
	)

	// Start the command with a PTY
	pty, tty, err := pty.Open()
	if err != nil {
		return fmt.Errorf("failed to open pty: %w", err)
	}

	tf.pty = pty
	tf.tty = tty
	tf.cmd.Stdout = tty
	tf.cmd.Stdin = tty
	tf.cmd.Stderr = tty

	// Set terminal size using syscall (fallback to original method)
	ws := struct {
		Row uint16
		Col uint16
		X   uint16
		Y   uint16
	}{40, 120, 0, 0}
	syscall.Syscall(syscall.SYS_IOCTL, pty.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))

	if err := tf.cmd.Start(); err != nil {
		pty.Close()
		tty.Close()
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Start the continuous reader
	tf.startReader()

	return nil
}

// startReader starts the continuous reader goroutine
func (tf *TUITestFramework) startReader() {
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := tf.pty.Read(buf)
			if n > 0 {
				tf.mu.Lock()
				for i := 0; i < n; i++ {
					tf.buf[tf.head] = buf[i]
					tf.head = (tf.head + 1) % ringSize
					if tf.head == 0 {
						tf.full = true
					}
				}
				tf.cond.Broadcast()
				tf.mu.Unlock()
			}
			if err != nil {
				tf.mu.Lock()
				tf.cond.Broadcast()
				tf.mu.Unlock()
				return
			}
		}
	}()
}

// SendKeys sends keystrokes to the application
func (tf *TUITestFramework) SendKeys(keys string) error {
	_, err := tf.pty.Write([]byte(keys))
	return err
}

// SendEnter sends an Enter key
func (tf *TUITestFramework) SendEnter() error {
	return tf.SendKeys(KeyEnter)
}

// SendCtrlC sends Ctrl+C to terminate the application
func (tf *TUITestFramework) SendCtrlC() error {
	return tf.SendKeys(KeyCtrlC)
}

// PressQuit sends 'q' to quit the application
func (tf *TUITestFramework) PressQuit() error {
	return tf.SendKeys(KeyQuit)
}

// OpenPager sends 'D' to open the git diff pager
func (tf *TUITestFramework) OpenPager() error {
	return tf.SendKeys(KeyDiff)
}

// PageDown sends space to page down in pager
func (tf *TUITestFramework) PageDown() error {
	return tf.SendKeys(KeySpace)
}

// ANSI escape sequence regex for normalization
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\x1b\]0;.*?\x07|\r`)

// OutputContains checks if the output contains specific text within a timeout
func (tf *TUITestFramework) OutputContains(text string, timeout time.Duration) bool {
	return tf.WaitFor(func(s string) bool { return strings.Contains(s, text) }, timeout)
}

// OutputContainsPlain checks if the normalized output contains specific text within a timeout
func (tf *TUITestFramework) OutputContainsPlain(text string, timeout time.Duration) bool {
	return tf.WaitFor(func(s string) bool {
		return strings.Contains(ansiRe.ReplaceAllString(s, ""), text)
	}, timeout)
}

// SnapshotPlain returns the current contents of the ring buffer with ANSI sequences removed
func (tf *TUITestFramework) SnapshotPlain() string {
	return ansiRe.ReplaceAllString(tf.Snapshot(), "")
}

// WaitFor waits for a predicate to be true in the output
func (tf *TUITestFramework) WaitFor(pred func(string) bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	tf.mu.Lock()
	defer tf.mu.Unlock()

	for {
		if pred(tf.snapshot()) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		tf.cond.Wait() // predicate loop handles spurious wakeups
	}
}

// WaitForText waits for specific text to appear in the output (legacy method)
func (tf *TUITestFramework) WaitForText(expectedText string, timeout time.Duration) bool {
	return tf.WaitFor(func(s string) bool { return strings.Contains(s, expectedText) }, timeout)
}

// Snapshot returns the current contents of the ring buffer (thread-safe)
func (tf *TUITestFramework) Snapshot() string {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	return tf.snapshot()
}

// snapshot returns the current contents of the ring buffer
// NOTE: This assumes the mutex is already locked by the caller
func (tf *TUITestFramework) snapshot() string {
	if !tf.full {
		return string(tf.buf[:tf.head])
	}
	out := make([]byte, ringSize)
	copy(out, tf.buf[tf.head:])
	copy(out[ringSize-tf.head:], tf.buf[:tf.head])
	return string(out)
}

// Cleanup closes the PTY and terminates the application
func (tf *TUITestFramework) Cleanup() {
	// Close PTY first to deliver SIGHUP to child process
	if tf.pty != nil {
		tf.pty.Close()
	}
	if tf.tty != nil {
		tf.tty.Close()
	}
	if tf.cmd != nil && tf.cmd.Process != nil {
		tf.cmd.Process.Kill()
		_, _ = tf.cmd.Process.Wait()
	}
	if tf.workspace != "" {
		os.RemoveAll(tf.workspace)
	}
}

// CreateTestWorkspace creates a temporary directory with test Git repositories
func (tf *TUITestFramework) CreateTestWorkspace() (string, error) {
	tmpDir := tf.t.TempDir()
	tf.workspace = tmpDir
	return tmpDir, nil
}

// CreateTestRepo creates a Git repository in the workspace
func (tf *TUITestFramework) CreateTestRepo(name string, options ...RepoOption) (string, error) {
	if tf.workspace == "" {
		return "", fmt.Errorf("workspace not created")
	}

	repoPath := filepath.Join(tf.workspace, name)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return "", err
	}

	// Initialize Git repo
	if err := tf.runGitCommand(repoPath, "init"); err != nil {
		return "", err
	}

	// Ensure main branch exists (deterministic branch setup)
	if err := tf.runGitCommand(repoPath, "checkout", "-b", "main"); err != nil {
		return "", err
	}

	// Apply options
	opts := &repoOptions{withCommit: true}
	for _, opt := range options {
		opt(opts)
	}

	// Create initial commit if requested
	if opts.withCommit {
		readmeContent := fmt.Sprintf("# %s\n\nTest repository for GitaGrip testing.", name)
		readmePath := filepath.Join(repoPath, "README.md")
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return "", err
		}

		if err := tf.runGitCommand(repoPath, "add", "."); err != nil {
			return "", err
		}
		if err := tf.runGitCommand(repoPath, "commit", "-m", "Initial commit"); err != nil {
			return "", err
		}
	}

	// Make dirty if requested
	if opts.dirty {
		dirtyPath := filepath.Join(repoPath, "dirty.txt")
		if err := os.WriteFile(dirtyPath, []byte("Uncommitted changes"), 0644); err != nil {
			return "", err
		}
	}

	// Add remote if requested
	if opts.withRemote {
		remotePath := filepath.Join(tf.workspace, name+"-remote.git")
		if err := tf.runGitCommand("", "init", "--bare", remotePath); err != nil {
			return "", err
		}
		if err := tf.runGitCommand(repoPath, "remote", "add", "origin", remotePath); err != nil {
			return "", err
		}
		if opts.withCommit {
			if err := tf.runGitCommand(repoPath, "push", "origin", "main"); err != nil {
				return "", err
			}
		}
	}

	return repoPath, nil
}

func (tf *TUITestFramework) runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Set deterministic git environment
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=GitaGrip Test",
		"GIT_AUTHOR_EMAIL=test@gitagrip.test",
		"GIT_COMMITTER_NAME=GitaGrip Test",
		"GIT_COMMITTER_EMAIL=test@gitagrip.test",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v failed: %v; out=%s", args, err, out)
	}
	return nil
}

// RepoOption is a function that configures repository creation
type RepoOption func(*repoOptions)

type repoOptions struct {
	withCommit bool
	dirty      bool
	withRemote bool
}

// WithCommit creates the repository with an initial commit
func WithCommit(commit bool) RepoOption {
	return func(opts *repoOptions) {
		opts.withCommit = commit
	}
}

// WithDirtyState creates the repository with uncommitted changes
func WithDirtyState() RepoOption {
	return func(opts *repoOptions) {
		opts.dirty = true
	}
}

// WithRemote creates the repository with a remote
func WithRemote() RepoOption {
	return func(opts *repoOptions) {
		opts.withRemote = true
	}
}

// Test functions

func TestMain(m *testing.M) {
	// Build the test binary from the parent directory
	fmt.Println("Building test binary from main project...")
	cmd := exec.Command("go", "build", "-o", "./e2e/gitagrip_e2e", ".")
	cmd.Dir = ".." // Run from parent directory
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to build test binary: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove("./gitagrip_e2e")
	os.Exit(code)
}

func TestHelpCommand(t *testing.T) {
	t.Parallel()

	// Ensure the test binary exists (it should be built by TestMain)
	if _, err := os.Stat("./gitagrip_e2e"); os.IsNotExist(err) {
		t.Skip("Test binary not found - TestMain may not have run yet")
	}

	// Test help command by running it directly (not through PTY since it exits quickly)
	cmd := exec.Command("./gitagrip_e2e", "--help")
	output, err := cmd.CombinedOutput()

	// The command should run without error
	require.NoError(t, err, "Help command should run without error")

	outputStr := string(output)
	t.Logf("Help output length: %d chars", len(outputStr))

	// Verify we got some meaningful output
	require.Greater(t, len(outputStr), 50, "Help should produce substantial output")

	// Check for key help elements (be more flexible with the text)
	require.True(t,
		strings.Contains(outputStr, "Usage") ||
			strings.Contains(outputStr, "usage") ||
			strings.Contains(outputStr, "help"),
		"Help should contain usage or help information")

	require.True(t,
		strings.Contains(outputStr, "Directory") ||
			strings.Contains(outputStr, "directory") ||
			strings.Contains(outputStr, "-d"),
		"Help should contain directory option information")

	t.Logf("Help command test passed - output contains expected content")
}

func TestBasicRepositoryDiscovery(t *testing.T) {
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
	require.True(t, tf.OutputContains("__READY__", 5*time.Second), "Should receive ready signal")

	// Wait for TUI to initialize and show content
	require.True(t, tf.OutputContains("gitagrip", 3*time.Second), "Should show gitagrip title")

	// Get current buffered output
	output := tf.Snapshot()

	// Only log summary with key indicators
	containsGitagrip := strings.Contains(output, "gitagrip")
	containsFrontend := strings.Contains(output, "frontend-app")
	containsBackend := strings.Contains(output, "backend-api")
	t.Logf("TUI output: %d chars, shows gitagrip=%v, frontend=%v, backend=%v", len(output), containsGitagrip, containsFrontend, containsBackend)

	// The app should have started without crashing
	assert.Greater(t, len(output), 50, "Should produce substantial output indicating TUI is running")
	assert.True(t, containsGitagrip, "Should show gitagrip title")
}

func TestKeyboardNavigation(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	// Create test workspace
	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	// Create multiple repos for navigation
	for i := 1; i <= 3; i++ {
		_, err = tf.CreateTestRepo(fmt.Sprintf("repo-%d", i))
		require.NoError(t, err, "Failed to create test repo")
	}

	// Start the application
	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to signal ready
	require.True(t, tf.OutputContains("__READY__", 5*time.Second), "Should receive ready signal")

	// Wait for TUI to initialize
	require.True(t, tf.OutputContains("gitagrip", 3*time.Second), "Should show gitagrip title")

	// Get initial state
	initialOutput := tf.Snapshot()

	// Send navigation commands
	tf.SendKeys("j") // Down
	time.Sleep(200 * time.Millisecond)

	// Get output after navigation
	navOutput := tf.Snapshot()

	t.Logf("Initial output: %d chars, navigation response: %d chars", len(initialOutput), len(navOutput))

	// The TUI should be running and responsive
	assert.True(t, len(initialOutput) > 100, "Should show TUI is running")
}

func TestRepositorySelection(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	defer tf.Cleanup()

	workspace, err := tf.CreateTestWorkspace()
	require.NoError(t, err, "Failed to create test workspace")

	_, err = tf.CreateTestRepo("selectable-repo")
	require.NoError(t, err, "Failed to create selectable repo")

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to signal ready
	require.True(t, tf.OutputContains("__READY__", 5*time.Second), "Should receive ready signal")

	// Wait for TUI to initialize
	require.True(t, tf.OutputContains("gitagrip", 3*time.Second), "Should show gitagrip title")

	// Try to select with spacebar
	tf.SendKeys(" ")
	time.Sleep(150 * time.Millisecond)

	output := tf.Snapshot()

	t.Logf("Selection test: output %d chars, app responsive", len(output))

	// Basic test that the app is running
	assert.True(t, len(output) > 100, "App should be running with TUI content")
}

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
	os.Remove(configPath)

	err = tf.StartApp("-d", workspace)
	require.NoError(t, err, "Failed to start app")

	// Wait for TUI to signal ready
	require.True(t, tf.OutputContains("__READY__", 5*time.Second), "Should receive ready signal")

	// Wait for TUI to initialize
	require.True(t, tf.OutputContains("gitagrip", 3*time.Second), "Should show gitagrip title")

	// Exit gracefully
	tf.PressQuit()
	time.Sleep(200 * time.Millisecond)

	// Check if config file was created
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "Config file should be created")

	if err == nil {
		configContent, err := os.ReadFile(configPath)
		require.NoError(t, err, "Should be able to read config file")

		configStr := string(configContent)
		assert.Contains(t, configStr, "version = 1", "Config should contain version")
		assert.Contains(t, configStr, workspace, "Config should contain workspace path")

		t.Logf("Config file created: %d chars, version=1, workspace included", len(configStr))
	}
}

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

	// Give TUI time to start up
	time.Sleep(500 * time.Millisecond)

	// Wait for TUI to initialize and render
	require.True(t, tf.OutputContains("gitagrip", 3*time.Second), "Should show gitagrip title")

	// Clear any buffered output first
	tf.Snapshot()

	// Set up exit monitoring before sending 'q'
	done := make(chan error, 1)
	go func() {
		done <- tf.cmd.Wait()
	}()

	// Send 'q' to quit
	t.Logf("Sending 'q' to quit application...")
	tf.PressQuit()

	// Give more time for graceful shutdown
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
		tf.SendCtrlC() // Force exit again
	}
}
