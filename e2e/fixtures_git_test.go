//go:build e2e && unix
//go:build e2e && unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RepoOption is a function that configures repository creation
type RepoOption func(*repoOptions)

type repoOptions struct {
	withCommit bool
	dirty      bool
	withRemote bool
	files      map[string]string // filename -> contents
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

// WithFiles creates the repository with specific files and contents
func WithFiles(files map[string]string) RepoOption {
	return func(opts *repoOptions) {
		opts.files = files
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

	// Create custom files if requested
	if opts.files != nil {
		for filename, content := range opts.files {
			filePath := filepath.Join(repoPath, filename)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return "", err
			}
		}
	}

	// Make dirty if requested
	if opts.dirty {
		dirtyPath := filepath.Join(repoPath, "dirty.txt")
		if err := os.WriteFile(dirtyPath, []byte("Uncommitted changes"), 0644); err != nil {
			return "", err
		}
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
		"GIT_CONFIG_GLOBAL=/dev/null", // ignore user ~/.gitconfig
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v failed: %v; out=%s", args, err, out)
	}
	return nil
}
