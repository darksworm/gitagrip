package query

import (
	"sort"
	"strings"
	
	"gitagrip/internal/domain"
	"gitagrip/internal/logic"
)

// Service handles complex queries about repositories and groups
type Service struct {
	// Dependencies
	repoStore  logic.RepositoryStore
	groupStore logic.GroupStore
	
	// State caches
	orderedGroups  []string
	orderedRepos   []string
	ungroupedRepos []string
	expandedGroups map[string]bool
}

// NewService creates a new query service
func NewService(repoStore logic.RepositoryStore, groupStore logic.GroupStore) *Service {
	return &Service{
		repoStore:      repoStore,
		groupStore:     groupStore,
		expandedGroups: make(map[string]bool),
	}
}

// SetOrderedGroups updates the ordered groups list
func (s *Service) SetOrderedGroups(groups []string) {
	s.orderedGroups = groups
}

// SetOrderedRepos updates the ordered repos list
func (s *Service) SetOrderedRepos(repos []string) {
	s.orderedRepos = repos
	s.updateUngroupedRepos()
}

// SetExpandedGroups updates which groups are expanded
func (s *Service) SetExpandedGroups(expanded map[string]bool) {
	s.expandedGroups = expanded
}

// GetMaxIndex returns the maximum selectable index
func (s *Service) GetMaxIndex() int {
	count := 0
	
	// Count groups and their repos if expanded
	for _, groupName := range s.orderedGroups {
		count++ // Group header
		
		if s.expandedGroups[groupName] {
			group := s.groupStore.GetGroup(groupName)
			if group != nil {
				count += len(group.Repos)
			}
		}
	}
	
	// Count ungrouped repos
	count += len(s.ungroupedRepos)
	
	// Return max index (count - 1)
	if count > 0 {
		return count - 1
	}
	return 0
}

// GetIndexInfo returns information about what's at a specific index
func (s *Service) GetIndexInfo(index int) *IndexInfo {
	currentIdx := 0
	
	// Check groups
	for _, groupName := range s.orderedGroups {
		if currentIdx == index {
			// This is a group header
			return &IndexInfo{
				Type:      IndexTypeGroup,
				GroupName: groupName,
			}
		}
		currentIdx++
		
		// Check repos in group if expanded
		if s.expandedGroups[groupName] {
			group := s.groupStore.GetGroup(groupName)
			if group != nil {
				for _, repoPath := range group.Repos {
					if currentIdx == index {
						repo := s.repoStore.GetRepository(repoPath)
						return &IndexInfo{
							Type:       IndexTypeRepository,
							GroupName:  groupName,
							Repository: repo,
							Path:       repoPath,
						}
					}
					currentIdx++
				}
			}
		}
	}
	
	// Check ungrouped repos
	for _, repoPath := range s.ungroupedRepos {
		if currentIdx == index {
			repo := s.repoStore.GetRepository(repoPath)
			return &IndexInfo{
				Type:       IndexTypeRepository,
				Repository: repo,
				Path:       repoPath,
			}
		}
		currentIdx++
	}
	
	return nil
}

// GetRepositoryAtIndex returns the repository at a specific index (nil if index is a group)
func (s *Service) GetRepositoryAtIndex(index int) *domain.Repository {
	info := s.GetIndexInfo(index)
	if info != nil && info.Type == IndexTypeRepository {
		return info.Repository
	}
	return nil
}

// GetRepositoryPathAtIndex returns the repository path at a specific index
func (s *Service) GetRepositoryPathAtIndex(index int) string {
	info := s.GetIndexInfo(index)
	if info != nil && info.Type == IndexTypeRepository {
		return info.Path
	}
	return ""
}

// GetUngroupedRepos returns repositories not in any group
func (s *Service) GetUngroupedRepos() []string {
	return s.ungroupedRepos
}

// GetVisibleRepositoryPaths returns all visible repository paths in order
func (s *Service) GetVisibleRepositoryPaths() []string {
	var paths []string
	
	// Add repos from expanded groups
	for _, groupName := range s.orderedGroups {
		if s.expandedGroups[groupName] {
			group := s.groupStore.GetGroup(groupName)
			if group != nil {
				paths = append(paths, group.Repos...)
			}
		}
	}
	
	// Add ungrouped repos
	paths = append(paths, s.ungroupedRepos...)
	
	return paths
}

// GetAllRepositoryPaths returns all repository paths regardless of visibility
func (s *Service) GetAllRepositoryPaths() []string {
	var paths []string
	
	// Add all repos from groups
	for _, groupName := range s.orderedGroups {
		group := s.groupStore.GetGroup(groupName)
		if group != nil {
			paths = append(paths, group.Repos...)
		}
	}
	
	// Add ungrouped repos
	paths = append(paths, s.ungroupedRepos...)
	
	return paths
}

// IsGroupExpanded checks if a group is expanded
func (s *Service) IsGroupExpanded(groupName string) bool {
	return s.expandedGroups[groupName]
}

// GetIndexForRepository finds the index of a repository
func (s *Service) GetIndexForRepository(targetPath string) int {
	currentIdx := 0
	
	// Check groups
	for _, groupName := range s.orderedGroups {
		currentIdx++ // Group header
		
		// Check repos in group if expanded
		if s.expandedGroups[groupName] {
			group := s.groupStore.GetGroup(groupName)
			if group != nil {
				for _, repoPath := range group.Repos {
					if repoPath == targetPath {
						return currentIdx
					}
					currentIdx++
				}
			}
		}
	}
	
	// Check ungrouped repos
	for _, repoPath := range s.ungroupedRepos {
		if repoPath == targetPath {
			return currentIdx
		}
		currentIdx++
	}
	
	return -1 // Not found
}

// Internal methods
func (s *Service) updateUngroupedRepos() {
	grouped := make(map[string]bool)
	
	// Mark all grouped repos
	for _, group := range s.groupStore.GetAllGroups() {
		for _, repoPath := range group.Repos {
			grouped[repoPath] = true
		}
	}
	
	// Find ungrouped repos
	s.ungroupedRepos = nil
	for _, repoPath := range s.orderedRepos {
		if !grouped[repoPath] {
			s.ungroupedRepos = append(s.ungroupedRepos, repoPath)
		}
	}
}

// Search-related queries
func (s *Service) GetRepositoriesMatching(query string) []IndexInfo {
	var matches []IndexInfo
	lowerQuery := strings.ToLower(query)
	currentIdx := 0
	
	// Search through groups
	for _, groupName := range s.orderedGroups {
		// Check group name
		if strings.Contains(strings.ToLower(groupName), lowerQuery) {
			matches = append(matches, IndexInfo{
				Type:      IndexTypeGroup,
				GroupName: groupName,
			})
		}
		currentIdx++
		
		// Check repos in group if expanded
		if s.expandedGroups[groupName] {
			group := s.groupStore.GetGroup(groupName)
			if group != nil {
				for _, repoPath := range group.Repos {
					repo := s.repoStore.GetRepository(repoPath)
					if repo != nil && strings.Contains(strings.ToLower(repo.Name), lowerQuery) {
						matches = append(matches, IndexInfo{
							Type:       IndexTypeRepository,
							GroupName:  groupName,
							Repository: repo,
							Path:       repoPath,
						})
					}
					currentIdx++
				}
			}
		}
	}
	
	// Search ungrouped repos
	for _, repoPath := range s.ungroupedRepos {
		repo := s.repoStore.GetRepository(repoPath)
		if repo != nil && strings.Contains(strings.ToLower(repo.Name), lowerQuery) {
			matches = append(matches, IndexInfo{
				Type:       IndexTypeRepository,
				Repository: repo,
				Path:       repoPath,
			})
		}
		currentIdx++
	}
	
	return matches
}

// SortRepositories sorts repository paths based on the given mode
func (s *Service) SortRepositories(paths []string, sortMode logic.SortMode) {
	switch sortMode {
	case logic.SortByName:
		sort.Slice(paths, func(i, j int) bool {
			repoI := s.repoStore.GetRepository(paths[i])
			repoJ := s.repoStore.GetRepository(paths[j])
			if repoI == nil || repoJ == nil {
				return repoI == nil
			}
			return strings.ToLower(repoI.Name) < strings.ToLower(repoJ.Name)
		})
		
	case logic.SortByStatus:
		sort.Slice(paths, func(i, j int) bool {
			repoI := s.repoStore.GetRepository(paths[i])
			repoJ := s.repoStore.GetRepository(paths[j])
			if repoI == nil || repoJ == nil {
				return repoI == nil
			}
			return getStatusPriority(repoI.Status) > getStatusPriority(repoJ.Status)
		})
		
	case logic.SortByBranch:
		sort.Slice(paths, func(i, j int) bool {
			repoI := s.repoStore.GetRepository(paths[i])
			repoJ := s.repoStore.GetRepository(paths[j])
			if repoI == nil || repoJ == nil {
				return repoI == nil
			}
			return strings.ToLower(repoI.Status.Branch) < strings.ToLower(repoJ.Status.Branch)
		})
		
	case logic.SortByPath:
		sort.Strings(paths)
	}
}

// Helper function for status priority
func getStatusPriority(status domain.RepoStatus) int {
	if status.Error != "" {
		return 4
	}
	if status.IsDirty || status.HasUntracked {
		return 3
	}
	if status.AheadCount > 0 || status.BehindCount > 0 {
		return 2
	}
	return 1
}