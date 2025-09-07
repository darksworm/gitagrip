package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"gitagrip/internal/domain"
	"gitagrip/internal/eventbus"
)

// DiscoveryService finds git repositories in the filesystem
type DiscoveryService interface {
	StartScan(ctx context.Context, roots []string) error
	StopScan()
}

// discoveryService is the concrete implementation
type discoveryService struct {
	bus         eventbus.EventBus
	mu          sync.Mutex
	isScanning  bool
	cancelFunc  context.CancelFunc
	wg          sync.WaitGroup
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(bus eventbus.EventBus) DiscoveryService {
	ds := &discoveryService{
		bus: bus,
	}
	
	// Subscribe to scan requests
	bus.Subscribe(eventbus.EventScanRequested, func(e eventbus.DomainEvent) {
		if event, ok := e.(eventbus.ScanRequestedEvent); ok {
			go ds.StartScan(context.Background(), event.Paths)
		}
	})
	
	return ds
}

// StartScan starts scanning for git repositories
func (ds *discoveryService) StartScan(ctx context.Context, roots []string) error {
	ds.mu.Lock()
	if ds.isScanning {
		ds.mu.Unlock()
		return fmt.Errorf("scan already in progress")
	}
	ds.isScanning = true
	
	// Create cancellable context
	scanCtx, cancel := context.WithCancel(ctx)
	ds.cancelFunc = cancel
	ds.mu.Unlock()
	
	// Publish scan started event
	ds.bus.Publish(eventbus.ScanStartedEvent{Paths: roots})
	
	// Track repositories found
	reposFound := 0
	
	// Scan in background
	ds.wg.Add(1)
	go func() {
		defer ds.wg.Done()
		defer func() {
			ds.mu.Lock()
			ds.isScanning = false
			ds.cancelFunc = nil
			ds.mu.Unlock()
			
			// Publish scan completed event
			ds.bus.Publish(eventbus.ScanCompletedEvent{ReposFound: reposFound})
		}()
		
		for _, root := range roots {
			select {
			case <-scanCtx.Done():
				return
			default:
				count := ds.scanDirectory(scanCtx, root)
				reposFound += count
			}
		}
	}()
	
	return nil
}

// StopScan stops any ongoing scan
func (ds *discoveryService) StopScan() {
	ds.mu.Lock()
	if ds.cancelFunc != nil {
		ds.cancelFunc()
	}
	ds.mu.Unlock()
	
	ds.wg.Wait()
}

// scanDirectory recursively scans a directory for git repositories
func (ds *discoveryService) scanDirectory(ctx context.Context, root string) int {
	reposFound := 0
	maxDepth := 5 // Maximum depth to scan
	
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Skip on error
		if err != nil {
			log.Printf("Error walking path %s: %v", path, err)
			return nil // Continue walking
		}
		
		// Skip if not a directory
		if !d.IsDir() {
			return nil
		}
		
		// Check depth limit
		relPath, _ := filepath.Rel(root, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			return filepath.SkipDir
		}
		
		// Skip common non-repository directories to speed up scanning
		dirName := d.Name()
		if dirName == "node_modules" || dirName == ".npm" || 
		   dirName == "vendor" || dirName == ".cache" ||
		   dirName == "dist" || dirName == "build" ||
		   dirName == "target" || dirName == ".gradle" ||
		   dirName == "__pycache__" || dirName == ".pytest_cache" ||
		   dirName == ".tox" || dirName == "venv" ||
		   dirName == ".venv" || dirName == "env" ||
		   strings.HasPrefix(dirName, ".") && dirName != ".git" {
			return filepath.SkipDir
		}
		
		// Check if this is a .git directory
		if dirName == ".git" {
			// Found a git repository - the parent is the repo root
			repoPath := filepath.Dir(path)
			repoName := filepath.Base(repoPath)
			
			// Create repository info with minimal status
			repo := domain.Repository{
				Path:        repoPath,
				Name:        repoName,
				DisplayName: repoName, // Initially same as Name, will be updated if duplicates found
				Group:       "", // Will be determined by group manager
				Status: domain.RepoStatus{
					Branch: "⋯", // Loading indicator, will be updated by git service
				},
			}
			
			// Publish discovery event immediately
			ds.bus.Publish(eventbus.RepoDiscoveredEvent{Repo: repo})
			reposFound++
			
			// Don't descend into .git directory
			return fs.SkipDir
		}
		
		// Skip hidden directories (except .git which we handle above)
		if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return fs.SkipDir
		}
		
		// Skip common non-repo directories
		skipDirs := []string{"node_modules", "target", "build", "dist", "vendor", "__pycache__"}
		for _, skipDir := range skipDirs {
			if d.Name() == skipDir {
				return fs.SkipDir
			}
		}
		
		return nil
	})
	
	if err != nil && err != context.Canceled {
		log.Printf("Error scanning directory %s: %v", root, err)
		ds.bus.Publish(eventbus.ErrorEvent{
			Message: fmt.Sprintf("Failed to scan %s", root),
			Err:     err,
		})
	}
	
	return reposFound
}