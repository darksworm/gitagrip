package logic

import (
	"strings"

	"gitagrip/internal/domain"
)

// SearchFilter handles search and filter operations
type SearchFilter struct {
	repositories map[string]*domain.Repository
}

// NewSearchFilter creates a new search filter
func NewSearchFilter(repositories map[string]*domain.Repository) *SearchFilter {
	return &SearchFilter{
		repositories: repositories,
	}
}

// MatchesFilter checks if a repo matches the given filter query
func (sf *SearchFilter) MatchesFilter(repo *domain.Repository, groupName string, filterQuery string) bool {
	if filterQuery == "" {
		return true
	}
	
	query := strings.ToLower(filterQuery)
	
	// Check if it's a status filter
	if strings.HasPrefix(query, "status:") {
		statusFilter := strings.TrimPrefix(query, "status:")
		return sf.MatchesStatusFilter(repo, statusFilter)
	}
	
	// Regular filter - check name, path, branch, group
	return strings.Contains(strings.ToLower(repo.Name), query) ||
		strings.Contains(strings.ToLower(repo.Path), query) ||
		strings.Contains(strings.ToLower(repo.Status.Branch), query) ||
		(groupName != "" && strings.Contains(strings.ToLower(groupName), query))
}

// MatchesGroupFilter checks if a group name matches the filter
func (sf *SearchFilter) MatchesGroupFilter(groupName string, filterQuery string) bool {
	if filterQuery == "" {
		return true
	}
	
	// Status filters don't match group names
	if strings.HasPrefix(filterQuery, "status:") {
		return false
	}
	
	query := strings.ToLower(filterQuery)
	return strings.Contains(strings.ToLower(groupName), query)
}

// MatchesStatusFilter checks if a repo matches the given status filter
func (sf *SearchFilter) MatchesStatusFilter(repo *domain.Repository, filter string) bool {
	switch filter {
	case "dirty":
		return repo.Status.IsDirty
	case "clean":
		return !repo.Status.IsDirty && !repo.Status.HasUntracked
	case "untracked":
		return repo.Status.HasUntracked
	case "ahead":
		return repo.Status.AheadCount > 0
	case "behind":
		return repo.Status.BehindCount > 0
	case "diverged":
		return repo.Status.AheadCount > 0 && repo.Status.BehindCount > 0
	case "stashed", "stash":
		return repo.Status.StashCount > 0
	case "error":
		return repo.Status.Error != ""
	default:
		// Check if it's a branch name
		return strings.Contains(strings.ToLower(repo.Status.Branch), filter)
	}
}

// SearchResult represents a search match
type SearchResult struct {
	Index int
	Type  SearchResultType
}

// SearchResultType indicates what type of match was found
type SearchResultType int

const (
	ResultTypeGroup SearchResultType = iota
	ResultTypeRepo
)

// PerformSearch searches for items matching the query
func (sf *SearchFilter) PerformSearch(query string, orderedGroups []string, groups map[string]*domain.Group, expandedGroups map[string]bool, ungroupedRepoPaths []string) []SearchResult {
	var results []SearchResult
	lowerQuery := strings.ToLower(query)
	currentIndex := 0
	
	// Check if it's a status filter
	isStatusFilter := false
	statusFilter := ""
	if strings.HasPrefix(lowerQuery, "status:") {
		isStatusFilter = true
		statusFilter = strings.TrimPrefix(lowerQuery, "status:")
	}
	
	// Search in groups first
	for _, groupName := range orderedGroups {
		// Check group name (only for non-status searches)
		if !isStatusFilter && strings.Contains(strings.ToLower(groupName), lowerQuery) {
			results = append(results, SearchResult{Index: currentIndex, Type: ResultTypeGroup})
		}
		currentIndex++
		
		// Check repos in group if expanded
		if expandedGroups[groupName] {
			group := groups[groupName]
			for _, repoPath := range group.Repos {
				if repo, ok := sf.repositories[repoPath]; ok {
					if isStatusFilter {
						// Filter by status
						if sf.MatchesStatusFilter(repo, statusFilter) {
							results = append(results, SearchResult{Index: currentIndex, Type: ResultTypeRepo})
						}
					} else {
						// Regular search
						if strings.Contains(strings.ToLower(repo.Name), lowerQuery) ||
							strings.Contains(strings.ToLower(repo.Path), lowerQuery) ||
							strings.Contains(strings.ToLower(repo.Status.Branch), lowerQuery) {
							results = append(results, SearchResult{Index: currentIndex, Type: ResultTypeRepo})
						}
					}
				}
				currentIndex++
			}
		}
	}
	
	// Search in ungrouped repos
	for _, repoPath := range ungroupedRepoPaths {
		if repo, ok := sf.repositories[repoPath]; ok {
			if isStatusFilter {
				// Filter by status
				if sf.MatchesStatusFilter(repo, statusFilter) {
					results = append(results, SearchResult{Index: currentIndex, Type: ResultTypeRepo})
				}
			} else {
				// Regular search
				if strings.Contains(strings.ToLower(repo.Name), lowerQuery) ||
					strings.Contains(strings.ToLower(repo.Path), lowerQuery) ||
					strings.Contains(strings.ToLower(repo.Status.Branch), lowerQuery) {
					results = append(results, SearchResult{Index: currentIndex, Type: ResultTypeRepo})
				}
			}
		}
		currentIndex++
	}
	
	return results
}