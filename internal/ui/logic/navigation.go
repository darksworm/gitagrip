package logic

import (
	"gitagrip/internal/domain"
)

// Navigator handles navigation and viewport management
type Navigator struct {
	selectedIndex      int
	viewportOffset     int
	viewportHeight     int
	expandedGroups     map[string]bool
	orderedGroups      []string
	groups             map[string]*domain.Group
	repositories       map[string]*domain.Repository
	ungroupedRepoCount int
}

// NewNavigator creates a new navigator
func NewNavigator() *Navigator {
	return &Navigator{
		expandedGroups: make(map[string]bool),
	}
}

// UpdateState updates the navigator's state
func (n *Navigator) UpdateState(selectedIndex, viewportOffset, viewportHeight int,
	expandedGroups map[string]bool, orderedGroups []string,
	groups map[string]*domain.Group, repositories map[string]*domain.Repository,
	ungroupedRepoCount int) {
	n.selectedIndex = selectedIndex
	n.viewportOffset = viewportOffset
	n.viewportHeight = viewportHeight
	n.expandedGroups = expandedGroups
	n.orderedGroups = orderedGroups
	n.groups = groups
	n.repositories = repositories
	n.ungroupedRepoCount = ungroupedRepoCount
}

// GetSelectedIndex returns the current selected index
func (n *Navigator) GetSelectedIndex() int {
	return n.selectedIndex
}

// GetViewportOffset returns the current viewport offset
func (n *Navigator) GetViewportOffset() int {
	return n.viewportOffset
}

// SetSelectedIndex sets the selected index and ensures it's visible
func (n *Navigator) SetSelectedIndex(index int) (int, int) {
	n.selectedIndex = index
	n.ensureSelectedVisible()
	return n.selectedIndex, n.viewportOffset
}

// GetMaxIndex returns the maximum selectable index
func (n *Navigator) GetMaxIndex(ungroupedReposCount int) int {
	count := len(n.orderedGroups) + ungroupedReposCount
	for groupName, group := range n.groups {
		if n.expandedGroups[groupName] {
			count += len(group.Repos)
		}
	}
	
	// Add gaps after each group (except hidden at the end)
	nonHiddenGroups := 0
	for _, groupName := range n.orderedGroups {
		if groupName != "_Hidden" {
			nonHiddenGroups++
		}
	}
	count += nonHiddenGroups // One gap per non-hidden group
	
	return count - 1
}

// EnsureSelectedVisible adjusts the viewport to keep the selected item visible
func (n *Navigator) ensureSelectedVisible() {
	// Calculate total items
	totalItems := n.calculateTotalItems()
	
	// If selected item is above viewport, scroll up
	if n.selectedIndex < n.viewportOffset {
		n.viewportOffset = n.selectedIndex
	}
	
	// Determine if we'll have scroll indicators
	needsTopIndicator := n.viewportOffset > 0
	needsBottomIndicator := n.viewportOffset + n.viewportHeight < totalItems
	
	// Special case: if we're showing items but can't fit them all even without bottom indicator,
	// we still need the bottom indicator
	if !needsBottomIndicator && needsTopIndicator {
		remainingItems := totalItems - n.viewportOffset
		availableSpace := n.viewportHeight - 1 // -1 for top indicator
		if remainingItems > availableSpace {
			needsBottomIndicator = true
		}
	}
	
	// Calculate effective visible area
	effectiveHeight := n.viewportHeight
	if needsTopIndicator {
		effectiveHeight--
	}
	if needsBottomIndicator {
		effectiveHeight--
	}
	
	// Ensure we have at least 1 line for content
	if effectiveHeight < 1 {
		effectiveHeight = 1
	}
	
	// If selected item is below effective viewport, scroll down
	if n.selectedIndex >= n.viewportOffset + effectiveHeight {
		// Calculate where to position the viewport
		// We need to make sure the selected item is visible
		newOffset := n.selectedIndex - effectiveHeight + 1
		
		// For the last few items, allow scrolling to show them without context
		// This ensures we can see all items at the bottom
		maxPossibleOffset := totalItems - effectiveHeight
		if maxPossibleOffset < 0 {
			maxPossibleOffset = 0
		}
		
		// If we're near the bottom, adjust to show all remaining items
		if newOffset > maxPossibleOffset {
			newOffset = maxPossibleOffset
		}
		
		// Don't scroll past the beginning
		if newOffset < 0 {
			newOffset = 0
		}
		
		n.viewportOffset = newOffset
	}
	
	// Final validation: ensure viewport doesn't exceed bounds
	// The maximum offset should ensure we can still fill the viewport
	maxOffset := totalItems - effectiveHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if n.viewportOffset > maxOffset {
		n.viewportOffset = maxOffset
	}
	if n.viewportOffset < 0 {
		n.viewportOffset = 0
	}
}

// calculateTotalItems calculates the total number of visible items
func (n *Navigator) calculateTotalItems() int {
	totalItems := 0
	// Groups first
	for _, groupName := range n.orderedGroups {
		totalItems++ // Group header
		if n.expandedGroups[groupName] {
			group := n.groups[groupName]
			totalItems += len(group.Repos)
		}
		// Account for gap after group (except hidden at the end)
		if groupName != "_Hidden" {
			totalItems++ // Gap after group
		}
	}
	// Add ungrouped repos
	totalItems += n.ungroupedRepoCount
	return totalItems
}

// CalculateTotalItemsWithUngrouped calculates total items including ungrouped repos
func (n *Navigator) CalculateTotalItemsWithUngrouped(ungroupedReposCount int) int {
	// This is now redundant since calculateTotalItems includes ungrouped count
	return n.calculateTotalItems()
}

// JumpToGroupBoundary jumps to the beginning or end of the current group
func (n *Navigator) JumpToGroupBoundary(toBeginning bool, ungroupedRepoPaths []string) (needsCrossGroupJump bool, fromGroup string) {
	currentIndex := 0
	
	for _, groupName := range n.orderedGroups {
		groupHeaderIndex := currentIndex
		
		// Check if we're on the group header
		if currentIndex == n.selectedIndex {
			// On header - jump to first or last repo in group
			if n.expandedGroups[groupName] {
				group := n.groups[groupName]
				if len(group.Repos) > 0 {
					if toBeginning {
						n.selectedIndex = groupHeaderIndex + 1
					} else {
						n.selectedIndex = groupHeaderIndex + len(group.Repos)
					}
					n.ensureSelectedVisible()
					return false, ""
				}
			}
			return false, "" // Group is collapsed or empty
		}
		currentIndex++
		
		// Check repos in group if expanded
		if n.expandedGroups[groupName] {
			group := n.groups[groupName]
			groupFirstRepoIndex := currentIndex
			groupLastRepoIndex := currentIndex + len(group.Repos) - 1
			
			// Check if we're inside this group
			for i := 0; i < len(group.Repos); i++ {
				if currentIndex == n.selectedIndex {
					if toBeginning {
						// Check if we're already at the first repo in the group
						if currentIndex == groupFirstRepoIndex {
							// Need to jump to previous group
							return true, groupName
						}
						// Jump to first repo in group
						n.selectedIndex = groupFirstRepoIndex
						n.ensureSelectedVisible()
						return false, ""
					} else {
						// Check if we're already at the last repo in the group
						if currentIndex == groupLastRepoIndex {
							// Need to jump to next group
							return true, groupName
						}
						// Jump to last repo in group
						n.selectedIndex = groupLastRepoIndex
						n.ensureSelectedVisible()
						return false, ""
					}
				}
				currentIndex++
			}
		}
		
		// Account for gap after group (except hidden at the end)
		if groupName != "_Hidden" {
			currentIndex++ // Gap after group
		}
	}
	
	// Check ungrouped repos
	if len(ungroupedRepoPaths) > 0 {
		ungroupedStartIndex := currentIndex
		ungroupedEndIndex := currentIndex + len(ungroupedRepoPaths) - 1
		
		// Check if we're in ungrouped section
		for i := 0; i < len(ungroupedRepoPaths); i++ {
			if currentIndex == n.selectedIndex {
				if toBeginning {
					n.selectedIndex = ungroupedStartIndex
				} else {
					n.selectedIndex = ungroupedEndIndex
				}
				n.ensureSelectedVisible()
				return false, ""
			}
			currentIndex++
		}
	}
	
	return false, ""
}

// GetCurrentIndexForGroup finds the current display index for a group
func (n *Navigator) GetCurrentIndexForGroup(targetGroupName string) int {
	currentIndex := 0
	
	for _, name := range n.orderedGroups {
		if name == targetGroupName {
			return currentIndex
		}
		currentIndex++
		
		if n.expandedGroups[name] {
			group := n.groups[name]
			currentIndex += len(group.Repos)
		}
		
		// Account for gap after group (except hidden at the end)
		if name != "_Hidden" {
			currentIndex++ // Gap after group
		}
	}
	
	return -1
}

// GetCurrentIndexForRepo finds the current display index for a repo
func (n *Navigator) GetCurrentIndexForRepo(repoPath string, ungroupedRepoPaths []string) int {
	currentIndex := 0
	
	// Check groups first
	for _, groupName := range n.orderedGroups {
		currentIndex++ // Group header
		
		if n.expandedGroups[groupName] {
			group := n.groups[groupName]
			for _, path := range group.Repos {
				if path == repoPath {
					return currentIndex
				}
				currentIndex++
			}
		}
		
		// Account for gap after group (except hidden at the end)
		if groupName != "_Hidden" {
			currentIndex++ // Gap after group
		}
	}
	
	// Check ungrouped repos
	for _, path := range ungroupedRepoPaths {
		if path == repoPath {
			return currentIndex
		}
		currentIndex++
	}
	
	return -1
}

// JumpToNextGroupEnd jumps to the last repo of the next group
func (n *Navigator) JumpToNextGroupEnd(currentGroupName string, ungroupedRepoPaths []string) bool {
	// Find current group index
	currentGroupIndex := -1
	for i, groupName := range n.orderedGroups {
		if groupName == currentGroupName {
			currentGroupIndex = i
			break
		}
	}
	
	// If we found the current group and there's a next group
	if currentGroupIndex != -1 && currentGroupIndex < len(n.orderedGroups)-1 {
		// Find the next expanded group with repos
		for i := currentGroupIndex + 1; i < len(n.orderedGroups); i++ {
			nextGroupName := n.orderedGroups[i]
			if n.expandedGroups[nextGroupName] {
				group := n.groups[nextGroupName]
				if len(group.Repos) > 0 {
					// Calculate the index of the last repo in this group
					currentIndex := 0
					for j, groupName := range n.orderedGroups {
						currentIndex++ // Group header
						if j < i {
							// Count all repos in previous groups
							if n.expandedGroups[groupName] {
								g := n.groups[groupName]
								currentIndex += len(g.Repos)
							}
							// Account for gap after group (except hidden at the end)
							if groupName != "_Hidden" {
								currentIndex++ // Gap after group
							}
						} else if j == i {
							// We're at the target group, add repos to get to last one
							currentIndex += len(group.Repos) - 1
							n.selectedIndex = currentIndex
							n.ensureSelectedVisible()
							return true
						}
					}
				}
			}
		}
	}
	
	// If no next group found, check ungrouped repos
	if len(ungroupedRepoPaths) > 0 {
		// Jump to last ungrouped repo
		currentIndex := 0
		// Count all groups and their repos
		for _, groupName := range n.orderedGroups {
			currentIndex++ // Group header
			if n.expandedGroups[groupName] {
				group := n.groups[groupName]
				currentIndex += len(group.Repos)
			}
			// Account for gap after group (except hidden at the end)
			if groupName != "_Hidden" {
				currentIndex++ // Gap after group
			}
		}
		// Add ungrouped repos
		currentIndex += len(ungroupedRepoPaths) - 1
		n.selectedIndex = currentIndex
		n.ensureSelectedVisible()
		return true
	}
	
	return false
}

// JumpToPreviousGroupStart jumps to the first repo of the previous group
func (n *Navigator) JumpToPreviousGroupStart(currentGroupName string) bool {
	// Find current group index
	currentGroupIndex := -1
	for i, groupName := range n.orderedGroups {
		if groupName == currentGroupName {
			currentGroupIndex = i
			break
		}
	}
	
	// If we found the current group and there's a previous group
	if currentGroupIndex > 0 {
		// Find the previous expanded group with repos
		for i := currentGroupIndex - 1; i >= 0; i-- {
			prevGroupName := n.orderedGroups[i]
			if n.expandedGroups[prevGroupName] {
				group := n.groups[prevGroupName]
				if len(group.Repos) > 0 {
					// Calculate the index of the first repo in this group
					currentIndex := 0
					for j, groupName := range n.orderedGroups {
						currentIndex++ // Group header
						if j < i {
							// Count all repos in previous groups
							if n.expandedGroups[groupName] {
								g := n.groups[groupName]
								currentIndex += len(g.Repos)
							}
							// Account for gap after group (except hidden at the end)
							if groupName != "_Hidden" {
								currentIndex++ // Gap after group
							}
						} else if j == i {
							// We're at the target group, we're already at first repo
							n.selectedIndex = currentIndex
							n.ensureSelectedVisible()
							return true
						}
					}
				}
			}
		}
	}
	
	// If no previous group found, stay where we are
	return false
}