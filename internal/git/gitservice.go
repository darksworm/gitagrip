package git

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
)

// GitService handles git repository operations
type GitService interface {
	RefreshRepo(ctx context.Context, repoPath string) (domain.RepoStatus, error)
	RefreshAll(ctx context.Context, repos []domain.Repository)
	StartBackgroundRefresh(ctx context.Context, interval time.Duration)
}

// gitService is the concrete implementation
type gitService struct {
	bus        eventbus.EventBus
	mu         sync.Mutex
	knownRepos map[string]bool
	workerPool chan struct{} // Semaphore for limiting concurrent git operations
}

// NewGitService creates a new git service
func NewGitService(bus eventbus.EventBus) GitService {
	gs := &gitService{
		bus:        bus,
		knownRepos: make(map[string]bool),
		workerPool: make(chan struct{}, 5), // Limit to 5 concurrent git operations
	}

	// Subscribe to repo discovery events
	bus.Subscribe(eventbus.EventRepoDiscovered, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.RepoDiscoveredEvent); ok {
			gs.mu.Lock()
			gs.knownRepos[event.Repo.Path] = true
			gs.mu.Unlock()

			// Get initial status
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				gs.RefreshRepo(ctx, event.Repo.Path)
			}()
		}
	})

	// Subscribe to status refresh requests
	bus.Subscribe(eventbus.EventStatusRefreshRequested, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.StatusRefreshRequestedEvent); ok {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if len(event.RepoPaths) == 0 {
					// Refresh all known repos
					gs.mu.Lock()
					repos := make([]domain.Repository, 0, len(gs.knownRepos))
					for path := range gs.knownRepos {
						repos = append(repos, domain.Repository{Path: path})
					}
					gs.mu.Unlock()
					gs.RefreshAll(ctx, repos)
				} else {
					// Refresh specific repos
					repos := make([]domain.Repository, 0, len(event.RepoPaths))
					for _, path := range event.RepoPaths {
						repos = append(repos, domain.Repository{Path: path})
					}
					gs.RefreshAll(ctx, repos)
				}
			}()
		}
	})

	// Subscribe to fetch requests
	bus.Subscribe(eventbus.EventFetchRequested, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.FetchRequestedEvent); ok {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Longer timeout for network ops
				defer cancel()

				var repos []string
				if len(event.RepoPaths) == 0 {
					// Fetch all known repos
					gs.mu.Lock()
					for path := range gs.knownRepos {
						repos = append(repos, path)
					}
					gs.mu.Unlock()
				} else {
					repos = event.RepoPaths
				}

				// Fetch each repository
				for _, repoPath := range repos {
					err := gs.fetchRepo(ctx, repoPath)
					if err != nil {
						log.Printf("Failed to fetch %s: %v", repoPath, err)
						gs.bus.Publish(eventbus.FetchCompletedEvent{
							RepoPath: repoPath,
							Success:  false,
							Error:    err,
						})
					} else {
						gs.bus.Publish(eventbus.FetchCompletedEvent{
							RepoPath: repoPath,
							Success:  true,
							Error:    nil,
						})
						// Refresh status after successful fetch
						gs.RefreshRepo(ctx, repoPath)
					}
				}
			}()
		}
	})

	// Subscribe to pull requests
	bus.Subscribe(eventbus.EventPullRequested, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.PullRequestedEvent); ok {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Longer timeout for network ops
				defer cancel()

				var repos []string
				if len(event.RepoPaths) == 0 {
					// Pull all known repos
					gs.mu.Lock()
					for path := range gs.knownRepos {
						repos = append(repos, path)
					}
					gs.mu.Unlock()
				} else {
					repos = event.RepoPaths
				}

				// Pull each repository
				for _, repoPath := range repos {
					err := gs.pullRepo(ctx, repoPath)
					if err != nil {
						log.Printf("Failed to pull %s: %v", repoPath, err)
						gs.bus.Publish(eventbus.PullCompletedEvent{
							RepoPath: repoPath,
							Success:  false,
							Error:    err,
						})
						// Also publish error event for UI notification
						gs.bus.Publish(eventbus.ErrorEvent{
							Message: fmt.Sprintf("Pull failed for %s", filepath.Base(repoPath)),
							Err:     err,
						})
					} else {
						gs.bus.Publish(eventbus.PullCompletedEvent{
							RepoPath: repoPath,
							Success:  true,
							Error:    nil,
						})
						// Refresh status after successful pull
						gs.RefreshRepo(ctx, repoPath)
					}
				}
			}()
		}
	})

	return gs
}

// RefreshRepo refreshes the status of a single repository
func (gs *gitService) RefreshRepo(ctx context.Context, repoPath string) (domain.RepoStatus, error) {
	// Acquire worker slot
	select {
	case gs.workerPool <- struct{}{}:
		defer func() { <-gs.workerPool }()
	case <-ctx.Done():
		return domain.RepoStatus{}, ctx.Err()
	}

	status := domain.RepoStatus{}

	// Get current branch
	branch, err := gs.getCurrentBranch(ctx, repoPath)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to get branch: %v", err)
		gs.publishStatus(repoPath, status)
		return status, err
	}
	status.Branch = branch

	// Get working tree status
	isDirty, hasUntracked, err := gs.getWorkingTreeStatus(ctx, repoPath)
	if err != nil {
		log.Printf("Failed to get working tree status for %s: %v", repoPath, err)
	}
	status.IsDirty = isDirty
	status.HasUntracked = hasUntracked

	// Get ahead/behind counts
	ahead, behind, err := gs.getAheadBehind(ctx, repoPath, branch)
	if err != nil {
		log.Printf("Failed to get ahead/behind for %s: %v", repoPath, err)
	}
	status.AheadCount = ahead
	status.BehindCount = behind

	// Publish status update
	gs.publishStatus(repoPath, status)

	return status, nil
}

// RefreshAll refreshes the status of all repositories
func (gs *gitService) RefreshAll(ctx context.Context, repos []domain.Repository) {
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(r domain.Repository) {
			defer wg.Done()
			gs.RefreshRepo(ctx, r.Path)
		}(repo)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All refreshes completed
	case <-ctx.Done():
		// Context cancelled
	}
}

// StartBackgroundRefresh starts periodic refresh of repository statuses
func (gs *gitService) StartBackgroundRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gs.mu.Lock()
			repos := make([]domain.Repository, 0, len(gs.knownRepos))
			for path := range gs.knownRepos {
				repos = append(repos, domain.Repository{Path: path})
			}
			gs.mu.Unlock()

			if len(repos) > 0 {
				refreshCtx, cancel := context.WithTimeout(ctx, interval)
				gs.RefreshAll(refreshCtx, repos)
				cancel()
			}

		case <-ctx.Done():
			return
		}
	}
}

// getCurrentBranch gets the current branch name
func (gs *gitService) getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(string(output))
	if branch == "HEAD" {
		// Detached HEAD state - try to get commit hash
		cmd = exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
		cmd.Dir = repoPath
		output, err = cmd.Output()
		if err != nil {
			return "detached", nil
		}
		branch = "detached@" + strings.TrimSpace(string(output))
	}

	return branch, nil
}

// getWorkingTreeStatus checks if the working tree is dirty or has untracked files
func (gs *gitService) getWorkingTreeStatus(ctx context.Context, repoPath string) (isDirty bool, hasUntracked bool, err error) {
	// Use git status --porcelain for machine-readable output
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return false, false, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		// First two characters indicate the status
		status := line[:2]

		// Check for modifications
		if status[0] != ' ' && status[0] != '?' || status[1] != ' ' && status[1] != '?' {
			isDirty = true
		}

		// Check for untracked files
		if status[0] == '?' || status[1] == '?' {
			hasUntracked = true
		}
	}

	return isDirty, hasUntracked, nil
}

// getAheadBehind gets the ahead/behind counts relative to the upstream branch
func (gs *gitService) getAheadBehind(ctx context.Context, repoPath string, branch string) (ahead int, behind int, err error) {
	// First check if there's an upstream branch
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", branch+"@{u}")
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	upstream, err := cmd.Output()
	if err != nil {
		// No upstream branch
		return 0, 0, nil
	}

	upstreamBranch := strings.TrimSpace(string(upstream))
	if upstreamBranch == "" {
		return 0, 0, nil
	}

	// Get ahead/behind counts
	cmd = exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", upstreamBranch+"..."+branch)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	// Parse output (format: "behind<tab>ahead")
	parts := strings.Split(strings.TrimSpace(string(output)), "\t")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected git rev-list output: %s", output)
	}

	behind, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	ahead, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return ahead, behind, nil
}

// fetchRepo performs a git fetch operation on the repository
func (gs *gitService) fetchRepo(ctx context.Context, repoPath string) error {
	startTime := time.Now()

	// Acquire worker slot
	select {
	case gs.workerPool <- struct{}{}:
		defer func() { <-gs.workerPool }()
	case <-ctx.Done():
		return ctx.Err()
	}

	// Run git fetch
	cmd := exec.CommandContext(ctx, "git", "fetch", "--all", "--prune")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime).Milliseconds()

	// Emit command log event
	if err != nil {
		gs.bus.Publish(eventbus.CommandExecutedEvent{
			RepoPath: repoPath,
			Command:  "fetch",
			Success:  false,
			Output:   string(output),
			Error:    err.Error(),
			Duration: duration,
		})
		return fmt.Errorf("git fetch failed: %v\nOutput: %s", err, output)
	}

	gs.bus.Publish(eventbus.CommandExecutedEvent{
		RepoPath: repoPath,
		Command:  "fetch",
		Success:  true,
		Output:   string(output),
		Error:    "",
		Duration: duration,
	})

	log.Printf("Fetched %s successfully", repoPath)
	return nil
}

// pullRepo performs a git pull operation on the repository
func (gs *gitService) pullRepo(ctx context.Context, repoPath string) error {
	startTime := time.Now()

	// Acquire worker slot
	select {
	case gs.workerPool <- struct{}{}:
		defer func() { <-gs.workerPool }()
	case <-ctx.Done():
		return ctx.Err()
	}

	// Run git pull
	cmd := exec.CommandContext(ctx, "git", "pull", "--rebase")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime).Milliseconds()

	// Emit command log event
	if err != nil {
		gs.bus.Publish(eventbus.CommandExecutedEvent{
			RepoPath: repoPath,
			Command:  "pull",
			Success:  false,
			Output:   string(output),
			Error:    err.Error(),
			Duration: duration,
		})
		return fmt.Errorf("git pull failed: %v\nOutput: %s", err, output)
	}

	gs.bus.Publish(eventbus.CommandExecutedEvent{
		RepoPath: repoPath,
		Command:  "pull",
		Success:  true,
		Output:   string(output),
		Error:    "",
		Duration: duration,
	})

	log.Printf("Pulled %s successfully", repoPath)
	return nil
}

// publishStatus publishes a status update event
func (gs *gitService) publishStatus(repoPath string, status domain.RepoStatus) {
	gs.bus.Publish(eventbus.StatusUpdatedEvent{
		RepoPath: repoPath,
		Status:   status,
	})
}
