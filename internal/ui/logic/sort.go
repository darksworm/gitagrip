package logic

import (
	"sort"
	"strings"

	"gitagrip/internal/domain"
)

// SortMode represents different sort modes
type SortMode int

const (
	SortByName SortMode = iota
	SortByStatus
	SortByBranch
	SortByGroup
)

// RepositorySorter handles repository sorting logic
type RepositorySorter struct {
	repositories map[string]*domain.Repository
}

// NewRepositorySorter creates a new repository sorter
func NewRepositorySorter(repositories map[string]*domain.Repository) *RepositorySorter {
	return &RepositorySorter{
		repositories: repositories,
	}
}

// SortRepositories sorts a slice of repository paths according to the given sort mode
func (s *RepositorySorter) SortRepositories(repoPaths []string, mode SortMode) {
	switch mode {
	case SortByName:
		s.sortByName(repoPaths)
	case SortByStatus:
		s.sortByStatus(repoPaths)
	case SortByBranch:
		s.sortByBranch(repoPaths)
	case SortByGroup:
		// Group sort doesn't affect repository order within groups
		s.sortByName(repoPaths)
	default:
		// Default to alphabetical by path
		sort.Strings(repoPaths)
	}
}

// sortByName sorts repositories alphabetically by name
func (s *RepositorySorter) sortByName(repoPaths []string) {
	sort.Slice(repoPaths, func(i, j int) bool {
		repoI, okI := s.repositories[repoPaths[i]]
		repoJ, okJ := s.repositories[repoPaths[j]]
		if !okI || !okJ {
			return !okI
		}
		return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
	})
}

// sortByStatus sorts repositories by status priority
func (s *RepositorySorter) sortByStatus(repoPaths []string) {
	sort.Slice(repoPaths, func(i, j int) bool {
		repoI, okI := s.repositories[repoPaths[i]]
		repoJ, okJ := s.repositories[repoPaths[j]]
		if !okI || !okJ {
			return !okI
		}
		// Order: error, dirty, clean
		statusI := GetStatusPriority(repoI)
		statusJ := GetStatusPriority(repoJ)
		if statusI != statusJ {
			return statusI > statusJ // Higher priority first
		}
		return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
	})
}

// sortByBranch sorts repositories by branch name
func (s *RepositorySorter) sortByBranch(repoPaths []string) {
	sort.Slice(repoPaths, func(i, j int) bool {
		repoI, okI := s.repositories[repoPaths[i]]
		repoJ, okJ := s.repositories[repoPaths[j]]
		if !okI || !okJ {
			return !okI
		}
		branchI := strings.ToLower(repoI.Status.Branch)
		branchJ := strings.ToLower(repoJ.Status.Branch)
		if branchI != branchJ {
			// Put main/master first
			if branchI == "main" || branchI == "master" {
				return true
			}
			if branchJ == "main" || branchJ == "master" {
				return false
			}
			return branchI < branchJ
		}
		return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
	})
}

// GetStatusPriority returns a priority value for sorting by status
func GetStatusPriority(repo *domain.Repository) int {
	if repo.Status.Error != "" {
		return 3 // Highest priority - errors
	}
	if repo.Status.IsDirty || repo.Status.HasUntracked {
		return 2 // Medium priority - dirty/untracked
	}
	if repo.Status.AheadCount > 0 || repo.Status.BehindCount > 0 {
		return 1 // Low priority - ahead/behind
	}
	return 0 // Lowest priority - clean
}

// ShouldSortGroups returns true if groups should be sorted alphabetically
func ShouldSortGroups(mode SortMode) bool {
	return mode == SortByGroup
}