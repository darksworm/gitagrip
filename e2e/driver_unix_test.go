//go:build e2e && unix

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
)

const ringSize = 1 << 20     // 1 MiB of scrollback
var binPath = "gitagrip_e2e" // unified binary path

// Key constants for better readability
const (
	KeyEnter = "\r"
	KeyCtrlC = "\x03"
	KeySpace = " "
	KeyDown  = "j"
	KeyQuit  = "q"
	KeyDiff  = "D"
	KeyFetch = "f"
	KeyPull  = "p"
)

// ANSI escape sequence regex for normalization - covers CSI, OSC, charset, keypad modes
var ansiRe = regexp.MustCompile(
	`(?:\x1b\[[0-9;?]*[ -/]*[@-~])|` + // CSI sequences
		`(?:\x1b\][^\x07]*\x07)|` + // OSC sequences
		`(?:\x1b[\(\)][A-Za-z])|` + // charset sequences
		`(?:\x1b=|\x1b>)|` + // keypad mode sequences
		`\r`, // carriage returns
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
	cmdArgs := append([]string{binPath}, args...)
	tf.cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// Set per-process environment variables
	tf.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"LC_ALL=C",
		"LANG=C",
		"HOME="+tf.workspace,          // isolate $HOME
		"GIT_CONFIG_GLOBAL=/dev/null", // ignore user ~/.gitconfig
		"GITAGRIP_E2E_TEST=1",
	)

	// Start the command with a PTY
	ptyFile, tty, err := pty.Open()
	if err != nil {
		return fmt.Errorf("failed to open pty: %w", err)
	}

	tf.pty = ptyFile
	tf.tty = tty
	tf.cmd.Stdout = tty
	tf.cmd.Stdin = tty
	tf.cmd.Stderr = tty

	// Set terminal size
	ws := struct {
		Row uint16
		Col uint16
		X   uint16
		Y   uint16
	}{40, 120, 0, 0}
	syscall.Syscall(syscall.SYS_IOCTL, ptyFile.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))

	if err := tf.cmd.Start(); err != nil {
		ptyFile.Close()
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
	tf.t.Helper()
	_, err := tf.pty.Write([]byte(keys))
	return err
}

// SendEnter sends an Enter key
func (tf *TUITestFramework) SendEnter() error {
	tf.t.Helper()
	return tf.SendKeys(KeyEnter)
}

// SendCtrlC sends Ctrl+C to terminate the application
func (tf *TUITestFramework) SendCtrlC() error {
	tf.t.Helper()
	return tf.SendKeys(KeyCtrlC)
}

// PressQuit sends 'q' to quit the application
func (tf *TUITestFramework) PressQuit() error {
	tf.t.Helper()
	return tf.SendKeys(KeyQuit)
}

// OpenDiffPager sends 'D' to open the git diff pager
func (tf *TUITestFramework) OpenDiffPager() error {
	return tf.SendKeys(KeyDiff)
}

// Fetch sends 'f' to trigger fetch operation
func (tf *TUITestFramework) Fetch() error {
	return tf.SendKeys(KeyFetch)
}

// Pull sends 'p' to trigger pull operation
func (tf *TUITestFramework) Pull() error {
	return tf.SendKeys(KeyPull)
}

// WaitForStatusMessage waits for a specific status message to appear
func (tf *TUITestFramework) WaitForStatusMessage(message string, timeout time.Duration) bool {
	return tf.WaitFor(func(s string) bool {
		return strings.Contains(s, message)
	}, timeout)
}

// PageDown sends space to page down in pager
func (tf *TUITestFramework) PageDown() error {
	tf.t.Helper()
	return tf.SendKeys(KeySpace)
}

// OpenDiffPager sends 'D' to open the git diff pager
func (tf *TUITestFramework) OpenPager() error {
	tf.t.Helper()
	return tf.SendKeys(KeyDiff)
}

// Page sends space to page down in pager
func (tf *TUITestFramework) Page() error {
	tf.t.Helper()
	return tf.SendKeys(KeySpace)
}

// Select sends space to select items
func (tf *TUITestFramework) Select() error {
	tf.t.Helper()
	return tf.SendKeys(KeySpace)
}

// Enter sends enter key
func (tf *TUITestFramework) Enter() error {
	tf.t.Helper()
	return tf.SendKeys(KeyEnter)
}

// Down sends down navigation key
func (tf *TUITestFramework) Down() error {
	tf.t.Helper()
	return tf.SendKeys(KeyDown)
}

// Driver DSL helpers for readable test scripts

// Ready waits for the app to signal it's ready
func (tf *TUITestFramework) Ready() bool {
	tf.t.Helper()
	return tf.OutputContains("__READY__", 5*time.Second)
}

// SeePlain waits for specific plain text to appear (normalized output)
func (tf *TUITestFramework) SeePlain(text string) bool {
	tf.t.Helper()
	return tf.OutputContainsPlain(text, 3*time.Second)
}

// Quit sends quit command
func (tf *TUITestFramework) Quit() error {
	tf.t.Helper()
	return tf.PressQuit()
}

// OutputContains checks if the output contains specific text within a timeout
func (tf *TUITestFramework) OutputContains(text string, timeout time.Duration) bool {
	tf.t.Helper()
	return tf.WaitFor(func(s string) bool { return strings.Contains(s, text) }, timeout)
}

// OutputContainsPlain checks if the normalized output contains specific text within a timeout
func (tf *TUITestFramework) OutputContainsPlain(text string, timeout time.Duration) bool {
	tf.t.Helper()
	return tf.WaitFor(func(s string) bool {
		return strings.Contains(ansiRe.ReplaceAllString(s, ""), text)
	}, timeout)
}

// WaitFor waits for a predicate to be true in the output
func (tf *TUITestFramework) WaitFor(pred func(string) bool, timeout time.Duration) bool {
	tf.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if pred(tf.Snapshot()) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(25 * time.Millisecond) // simple, reliable polling; tests only
	}
}

// WaitForText waits for specific text to appear in the output (legacy method)
func (tf *TUITestFramework) WaitForText(expectedText string, timeout time.Duration) bool {
	tf.t.Helper()
	return tf.WaitFor(func(s string) bool { return strings.Contains(s, expectedText) }, timeout)
}

// WaitForE waits for a predicate with better error messages and failure artifacts
func (tf *TUITestFramework) WaitForE(pred func(string) bool, timeout time.Duration, failMsg string) error {
	tf.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if pred(tf.Snapshot()) {
			return nil
		}
		if time.Now().After(deadline) {
			tail := tf.SnapshotPlain()
			if len(tail) > 4096 {
				tail = tail[len(tail)-4096:]
			}
			return fmt.Errorf("%s\n--- tail ---\n%s", failMsg, tail)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// Snapshot returns the current contents of the ring buffer (thread-safe)
func (tf *TUITestFramework) Snapshot() string {
	tf.t.Helper()
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

// SnapshotPlain returns the current contents of the ring buffer with ANSI sequences removed
func (tf *TUITestFramework) SnapshotPlain() string {
	tf.t.Helper()
	return ansiRe.ReplaceAllString(tf.Snapshot(), "")
}

// DumpTailOnFail saves the last N bytes of normalized output to a file for debugging
func (tf *TUITestFramework) DumpTailOnFail(t *testing.T, name string, n int) {
	tf.t.Helper()
	s := tf.SnapshotPlain()
	if len(s) > n {
		s = s[len(s)-n:]
	}
	p := filepath.Join(t.TempDir(), name+".txt")
	_ = os.WriteFile(p, []byte(s), 0644)
	t.Logf("Saved tail to %s", p)
}

// Cleanup closes the PTY and terminates the application
func (tf *TUITestFramework) Cleanup() {
	// Close PTY first to deliver SIGHUP to child process
	if tf.pty != nil {
		_ = tf.pty.Close()
		tf.pty = nil
	}
	if tf.tty != nil {
		_ = tf.tty.Close()
		tf.tty = nil
	}
	if tf.cmd != nil && tf.cmd.Process != nil {
		_ = tf.cmd.Process.Kill()
		_, _ = tf.cmd.Process.Wait()
		tf.cmd = nil
	}
	if tf.workspace != "" {
		_ = os.RemoveAll(tf.workspace)
		tf.workspace = ""
	}
}
