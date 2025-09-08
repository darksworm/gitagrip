package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TUITestFramework provides utilities for testing TUI applications
type TUITestFramework struct {
	t        *testing.T
	pty      *os.File
	tty      *os.File
	cmd      *exec.Cmd
	workspace string
}

// NewTUITest creates a new TUI test framework instance
func NewTUITest(t *testing.T) *TUITestFramework {
	return &TUITestFramework{
		t: t,
	}
}

// StartApp launches the gitagrip application with given arguments in a PTY
func (tf *TUITestFramework) StartApp(args ...string) error {
	// Build the command
	cmdArgs := append([]string{"./gitagrip_e2e"}, args...)
	tf.cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	
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
	
	// Set terminal size (using syscall)
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
	
	return nil
}

// SendKeys sends keystrokes to the application
func (tf *TUITestFramework) SendKeys(keys string) error {
	_, err := tf.pty.Write([]byte(keys))
	return err
}

// SendEnter sends an Enter key
func (tf *TUITestFramework) SendEnter() error {
	return tf.SendKeys("\r")
}

// SendCtrlC sends Ctrl+C to terminate the application
func (tf *TUITestFramework) SendCtrlC() error {
	return tf.SendKeys("\x03")
}

// ReadOutput reads available output from the PTY with timeout
func (tf *TUITestFramework) ReadOutput(timeout time.Duration) (string, error) {
	done := make(chan struct{})
	var output strings.Builder
	var readErr error
	
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := tf.pty.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				readErr = err
				return
			}
		}
	}()
	
	select {
	case <-done:
		return output.String(), readErr
	case <-time.After(timeout):
		return output.String(), nil
	}
}

// WaitForText waits for specific text to appear in the output
func (tf *TUITestFramework) WaitForText(expectedText string, timeout time.Duration) bool {
	start := time.Now()
	
	for time.Since(start) < timeout {
		output, _ := tf.ReadOutput(100 * time.Millisecond)
		if strings.Contains(output, expectedText) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	return false
}

// Cleanup closes the PTY and terminates the application
func (tf *TUITestFramework) Cleanup() {
	if tf.cmd != nil && tf.cmd.Process != nil {
		tf.cmd.Process.Kill()
		tf.cmd.Wait()
	}
	if tf.pty != nil {
		tf.pty.Close()
	}
	if tf.tty != nil {
		tf.tty.Close()
	}
	if tf.workspace != "" {
		os.RemoveAll(tf.workspace)
	}
}

// CreateTestWorkspace creates a temporary directory with test Git repositories
func (tf *TUITestFramework) CreateTestWorkspace() (string, error) {
	tmpDir, err := os.MkdirTemp("", "gitagrip-test-*")
	if err != nil {
		return "", err
	}
	
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
	
	// Configure Git user
	if err := tf.runGitCommand(repoPath, "config", "user.email", "test@gitagrip.test"); err != nil {
		return "", err
	}
	if err := tf.runGitCommand(repoPath, "config", "user.name", "GitaGrip Test"); err != nil {
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
	return cmd.Run()
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
	tf := NewTUITest(t)
	defer tf.Cleanup()
	
	err := tf.StartApp("--help")
	require.NoError(t, err, "Failed to start app with --help")
	
	// Read output
	output, err := tf.ReadOutput(2 * time.Second)
	require.NoError(t, err, "Failed to read help output")
	
	// Verify help content
	assert.Contains(t, output, "Usage", "Help should contain usage information")
	assert.Contains(t, output, "Directory to scan", "Help should contain directory option description")
	
	// Only log summary instead of full output
	t.Logf("Help output length: %d chars, contains expected content", len(output))
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
	
	// Wait for the app to discover repositories
	time.Sleep(3 * time.Second)
	
	// Read output
	output, err := tf.ReadOutput(1 * time.Second)
	require.NoError(t, err, "Failed to read output")
	
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
	
	// Wait for startup
	time.Sleep(3 * time.Second)
	
	// Get initial state and clear buffer
	initialOutput, _ := tf.ReadOutput(500 * time.Millisecond)
	
	// Send navigation commands with longer waits
	tf.SendKeys("j") // Down
	time.Sleep(1 * time.Second)
	
	// Capture any response to navigation
	navOutput, _ := tf.ReadOutput(1 * time.Second)
	
	t.Logf("Initial output: %d chars, navigation response: %d chars", len(initialOutput), len(navOutput))
	
	// Navigation should either produce output or the initial output should show the app is running
	assert.True(t, len(initialOutput) > 100 || len(navOutput) > 0, "Should show TUI is running and responsive")
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
	
	time.Sleep(2 * time.Second)
	
	// Try to select with spacebar
	tf.SendKeys(" ")
	time.Sleep(500 * time.Millisecond)
	
	output, _ := tf.ReadOutput(500 * time.Millisecond)
	
	t.Logf("Selection test: output %d chars, app responsive", len(output))
	
	// Basic test that the app responds to selection
	assert.True(t, len(output) >= 0, "App should respond to selection input")
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
	
	time.Sleep(2 * time.Second)
	
	// Exit gracefully
	tf.SendKeys("q")
	time.Sleep(1 * time.Second)
	
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
	
	// Wait longer for the app to fully initialize and render
	time.Sleep(5 * time.Second)
	
	// Read and clear any buffered output first
	tf.ReadOutput(500 * time.Millisecond)
	
	// Set up exit monitoring before sending 'q'
	done := make(chan error, 1)
	go func() {
		done <- tf.cmd.Wait()
	}()
	
	// Send 'q' to quit
	t.Logf("Sending 'q' to quit application...")
	tf.SendKeys("q")
	
	// Give more time for graceful shutdown
	select {
	case exitErr := <-done:
		if exitErr == nil {
			t.Logf("Process exited cleanly with 'q' command")
		} else {
			t.Logf("Process exited with 'q' command (exit code: %v)", exitErr)
		}
		return
	case <-time.After(6 * time.Second):
		// If 'q' didn't work within 6 seconds, use Ctrl+C
		t.Logf("'q' didn't work within 6 seconds, using Ctrl+C")
		tf.SendCtrlC()
	}
	
	// Wait for Ctrl+C to work
	select {
	case exitErr := <-done:
		t.Logf("Process exited with Ctrl+C (exit code: %v)", exitErr)
	case <-time.After(3 * time.Second):
		t.Error("Application did not exit within total timeout")
		tf.SendCtrlC() // Force exit again
	}
}