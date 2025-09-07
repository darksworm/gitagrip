package input

import (
	"gitagrip/internal/ui/repositories"
	"gitagrip/internal/ui/state"
	"gitagrip/internal/ui/logic"
)

// ModelContext implements the Context interface for the input handler
type ModelContext struct {
	State         *state.AppState
	Store         repositories.RepositoryStore
	Navigator     *logic.Navigator
	CurrentSort   logic.SortMode
}

// CurrentIndex returns the current selected index
func (c *ModelContext) CurrentIndex() int {
	return c.State.SelectedIndex
}

// TotalItems returns the total number of visible items
func (c *ModelContext) TotalItems() int {
	totalRepos := 0
	for _, group := range c.State.OrderedGroups {
		totalRepos++ // Count the group itself
		if c.State.ExpandedGroups[group] {
			groupData := c.State.Groups[group]
			if groupData != nil {
				totalRepos += len(groupData.Repos)
			}
		}
	}
	// Add ungrouped repos if they exist
	ungrouped := c.State.Groups["Ungrouped"]
	if ungrouped != nil && len(ungrouped.Repos) > 0 && c.State.ExpandedGroups["Ungrouped"] {
		totalRepos += len(ungrouped.Repos)
	}
	return totalRepos
}

// HasSelection returns true if any items are selected
func (c *ModelContext) HasSelection() bool {
	return len(c.State.SelectedRepos) > 0
}

// SelectedCount returns the number of selected items
func (c *ModelContext) SelectedCount() int {
	return len(c.State.SelectedRepos)
}

// CurrentRepositoryPath returns the repo path at the current index
func (c *ModelContext) CurrentRepositoryPath() string {
	return c.GetRepoPathAtIndex(c.CurrentIndex())
}

// GetRepoPathAtIndex returns the repo path at the given index
func (c *ModelContext) GetRepoPathAtIndex(index int) string {
	currentIdx := 0
	
	// Check grouped repos
	for i, groupName := range c.State.OrderedGroups {
		if currentIdx == index {
			// On a group header
			return ""
		}
		currentIdx++
		
		if c.State.ExpandedGroups[groupName] {
			group := c.State.Groups[groupName]
			if group != nil {
				for _, repoPath := range group.Repos {
					if currentIdx == index {
						return repoPath
					}
					currentIdx++
				}
			}
		}
		
		// Add gap after group unless it's the hidden group at the end
		isLastGroup := i == len(c.State.OrderedGroups)-1
		isHiddenGroup := groupName == "_Hidden"
		if !isHiddenGroup || !isLastGroup {
			if currentIdx == index {
				// On a gap
				return ""
			}
			currentIdx++ // Gap after group
		}
	}
	
	// Check ungrouped repos
	ungrouped := c.State.Groups["Ungrouped"]
	if ungrouped != nil && len(ungrouped.Repos) > 0 && c.State.ExpandedGroups["Ungrouped"] {
		if currentIdx == index {
			// On ungrouped header
			return ""
		}
		currentIdx++
		
		for _, repoPath := range ungrouped.Repos {
			if currentIdx == index {
				return repoPath
			}
			currentIdx++
		}
	}
	
	return ""
}

// IsOnGroup returns true if the current selection is on a group header
func (c *ModelContext) IsOnGroup() bool {
	currentIdx := 0
	targetIdx := c.CurrentIndex()
	
	// Check grouped repos
	for i, groupName := range c.State.OrderedGroups {
		if currentIdx == targetIdx {
			return true // On a group header
		}
		currentIdx++
		
		if c.State.ExpandedGroups[groupName] {
			group := c.State.Groups[groupName]
			if group != nil {
				currentIdx += len(group.Repos)
			}
		}
		
		// Add gap after group unless it's the hidden group at the end
		isLastGroup := i == len(c.State.OrderedGroups)-1
		isHiddenGroup := groupName == "_Hidden"
		if !isHiddenGroup || !isLastGroup {
			currentIdx++ // Gap after group
		}
	}
	
	// Check ungrouped header
	ungrouped := c.State.Groups["Ungrouped"]
	if ungrouped != nil && len(ungrouped.Repos) > 0 && c.State.ExpandedGroups["Ungrouped"] {
		if currentIdx == targetIdx {
			return true // On ungrouped header
		}
	}
	
	return false
}

// CurrentGroupName returns the name of the group at the current index
func (c *ModelContext) CurrentGroupName() string {
	currentIdx := 0
	targetIdx := c.CurrentIndex()
	
	// Check grouped repos
	for i, groupName := range c.State.OrderedGroups {
		if currentIdx == targetIdx {
			return groupName // On a group header
		}
		currentIdx++
		
		if c.State.ExpandedGroups[groupName] {
			group := c.State.Groups[groupName]
			if group != nil {
				currentIdx += len(group.Repos)
			}
		}
		
		// Add gap after group unless it's the hidden group at the end
		isLastGroup := i == len(c.State.OrderedGroups)-1
		isHiddenGroup := groupName == "_Hidden"
		if !isHiddenGroup || !isLastGroup {
			currentIdx++ // Gap after group
		}
	}
	
	// Check ungrouped header
	ungrouped := c.State.Groups["Ungrouped"]
	if ungrouped != nil && len(ungrouped.Repos) > 0 && c.State.ExpandedGroups["Ungrouped"] {
		if currentIdx == targetIdx {
			return "Ungrouped"
		}
	}
	
	return ""
}

// SearchQuery returns the current search query
func (c *ModelContext) SearchQuery() string {
	return c.State.SearchQuery
}

// GetCurrentSort returns the current sort mode
func (c *ModelContext) GetCurrentSort() string {
	switch c.CurrentSort {
	case logic.SortByName:
		return "name"
	case logic.SortByStatus:
		return "status"
	case logic.SortByBranch:
		return "branch"
	case logic.SortByGroup:
		return "group"
	default:
		return "name"
	}
}