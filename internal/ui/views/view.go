package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"

	"gitagrip/internal/domain"
	"gitagrip/internal/ui/input/modes"
)

// ViewState contains all the state needed for rendering
type ViewState struct {
	Width           int
	Height          int
	Repositories    map[string]*domain.Repository
	Groups          map[string]*domain.Group
	OrderedGroups   []string
	SelectedIndex   int
	SelectedRepos   map[string]bool
	RefreshingRepos map[string]bool
	FetchingRepos   map[string]bool
	PullingRepos    map[string]bool
	ExpandedGroups  map[string]bool
	Scanning        bool
	StatusMessage   string
	ShowHelp        bool
	ShowLog         bool
	LogContent      string
	ShowInfo        bool
	InfoContent     string
	ViewportOffset  int
	ViewportHeight  int
	SearchQuery     string
	FilterQuery     string
	IsFiltered      bool
	ShowAheadBehind bool
	HelpModel       help.Model
	DeleteTarget    string
	TextInput       string
	InputMode       string
	UngroupedRepos  []string
	SortOptionIndex int
	LoadingState    string
	LoadingCount    int
}

// Renderer handles all view rendering
type Renderer struct {
	styles      *Styles
	repoRender  *RepositoryRenderer
	groupRender *GroupRenderer
}

// NewRenderer creates a new renderer
func NewRenderer(showAheadBehind bool) *Renderer {
	styles := NewStyles()
	return &Renderer{
		styles:      styles,
		repoRender:  NewRepositoryRenderer(styles, showAheadBehind),
		groupRender: NewGroupRenderer(styles),
	}
}

// Render produces the complete view
func (r *Renderer) Render(state ViewState) string {
	content := &strings.Builder{}

	// Title
	content.WriteString(r.styles.Title.Render("gitagrip"))
	content.WriteString("\n")

	// Delete confirmation
	if state.DeleteTarget != "" {
		content.WriteString(r.styles.Confirm.Render(fmt.Sprintf("Delete group '%s'? (y/n): ", state.DeleteTarget)))
		content.WriteString("\n")
	} else if state.InputMode != "" {
		if state.InputMode == "sort" {
			content.WriteString(r.renderSortOptions(state))
		} else {
			content.WriteString(state.TextInput)
		}
		content.WriteString("\n")
		content.WriteString("\n")
	}

	// Main content
	if state.Scanning && len(state.Repositories) == 0 {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := int(time.Now().UnixMilli()/80) % len(spinner)
		content.WriteString(r.styles.Scan.Render(fmt.Sprintf("%s Scanning for repositories...", spinner[frame])))
	} else if len(state.Repositories) == 0 {
		content.WriteString(r.styles.Dim.Render("No repositories found. Press F for full scan."))
	} else {
		content.WriteString(r.renderRepositoryList(state))
	}
	content.WriteString("\n")

	// Status bar
	r.renderStatusBar(content, state)

	// Log popup
	if state.ShowLog && state.LogContent != "" {
		r.renderLogPopup(content, state.LogContent)
	}

	// Info popup
	if state.ShowInfo && state.InfoContent != "" {
		r.renderInfoPopup(content, state.InfoContent)
	}

	// Help
	if state.ShowHelp {
		content.WriteString("\n")
		content.WriteString(state.HelpModel.View(nil))
		content.WriteString("\n")
	} else {
		content.WriteString("\n")
		content.WriteString(r.styles.Help.Render("Press ? for help"))
	}

	// Apply main container style
	mainStyle := r.styles.Main.MaxHeight(state.Height)
	return mainStyle.Render(content.String())
}

// renderRepositoryList renders the list of repositories with groups
func (r *Renderer) renderRepositoryList(state ViewState) string {
	var lines []string
	currentIndex := 0

	// Track which items are visible
	visibleLines := make([]string, 0)

	// Groups first
	for _, groupName := range state.OrderedGroups {
		group := state.Groups[groupName]
		isSelected := currentIndex == state.SelectedIndex
		isExpanded := state.ExpandedGroups[groupName]

		// Check if group is in viewport
		if currentIndex >= state.ViewportOffset {
			repoCount := 0
			if isExpanded {
				// Count visible repos in group
				for _, repoPath := range group.Repos {
					if repo, ok := state.Repositories[repoPath]; ok {
						if r.matchesFilter(repo, groupName, state.FilterQuery) {
							repoCount++
						}
					}
				}
			} else {
				repoCount = len(group.Repos)
			}

			header := r.groupRender.RenderGroupHeader(group, isExpanded, isSelected, state.SearchQuery, repoCount, state.Width)
			visibleLines = append(visibleLines, header)
		}
		currentIndex++

		// Render repos in group if expanded
		if isExpanded {
			for _, repoPath := range group.Repos {
				repo, ok := state.Repositories[repoPath]
				if !ok || (state.IsFiltered && !r.matchesFilter(repo, groupName, state.FilterQuery)) {
					continue
				}

				isRepoSelected := currentIndex == state.SelectedIndex
				if currentIndex >= state.ViewportOffset {
					line := r.repoRender.RenderRepository(
						repo, isRepoSelected, 1,
						len(state.SelectedRepos) > 0,
						state.FetchingRepos[repoPath],
						state.RefreshingRepos[repoPath],
						state.PullingRepos[repoPath],
						state.SearchQuery,
						state.SelectedRepos[repoPath],
						state.Width,
					)
					visibleLines = append(visibleLines, line)
				}
				currentIndex++
			}
		}
		
		// Add gap after group (except for hidden group at the end)
		if groupName != "_Hidden" || currentIndex < state.SelectedIndex {
			if currentIndex >= state.ViewportOffset && len(visibleLines) > 0 {
				visibleLines = append(visibleLines, "") // Empty line for gap
			}
			currentIndex++ // Count the gap in index
		}
	}

	// Ungrouped repos
	for _, repoPath := range state.UngroupedRepos {
		repo, ok := state.Repositories[repoPath]
		if !ok || (state.IsFiltered && !r.matchesFilter(repo, "", state.FilterQuery)) {
			continue
		}

		isRepoSelected := currentIndex == state.SelectedIndex
		if currentIndex >= state.ViewportOffset {
			line := r.repoRender.RenderRepository(
				repo, isRepoSelected, 0,
				len(state.SelectedRepos) > 0,
				state.FetchingRepos[repoPath],
				state.RefreshingRepos[repoPath],
				state.PullingRepos[repoPath],
				state.SearchQuery,
				state.SelectedRepos[repoPath],
				state.Width,
			)
			visibleLines = append(visibleLines, line)
		}
		currentIndex++
	}

	// Calculate effective height
	effectiveHeight := state.ViewportHeight
	needsTopIndicator := state.ViewportOffset > 0
	needsBottomIndicator := len(visibleLines) > effectiveHeight || currentIndex > state.ViewportOffset+state.ViewportHeight

	if needsTopIndicator {
		effectiveHeight--
	}
	if needsBottomIndicator {
		effectiveHeight--
	}

	// Add scroll indicators
	if needsTopIndicator {
		lines = append(lines, r.styles.Scroll.Render(fmt.Sprintf("↑ %d more above ↑", state.ViewportOffset)))
	}

	// Add visible lines (up to effective height)
	for i := 0; i < effectiveHeight && i < len(visibleLines); i++ {
		lines = append(lines, visibleLines[i])
	}

	// Add bottom scroll indicator
	if needsBottomIndicator {
		// Calculate how many items are below the current viewport
		// currentIndex is the total number of items
		// state.ViewportOffset + effectiveHeight is what we're showing
		itemsBelow := currentIndex - (state.ViewportOffset + effectiveHeight)
		if itemsBelow < 0 {
			itemsBelow = 0
		}
		lines = append(lines, r.styles.Scroll.Render(fmt.Sprintf("↓ %d more below ↓", itemsBelow)))
	}

	return strings.Join(lines, "\n")
}

// renderStatusBar renders the status bar
func (r *Renderer) renderStatusBar(content *strings.Builder, state ViewState) {
	statusParts := []string{
		fmt.Sprintf("%d repos", len(state.Repositories)),
	}

	if len(state.Groups) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d groups", len(state.Groups)))
	}

	if len(state.RefreshingRepos) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d refreshing", len(state.RefreshingRepos)))
	}

	if len(state.FetchingRepos) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d fetching", len(state.FetchingRepos)))
	}

	if len(state.SelectedRepos) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d selected", len(state.SelectedRepos)))
	}

	if state.Scanning {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := int(time.Now().UnixMilli()/80) % len(spinner)
		statusParts = append(statusParts, r.styles.Scan.Render(fmt.Sprintf("%s Scanning...", spinner[frame])))
	}

	if state.FilterQuery != "" {
		filterText := fmt.Sprintf("Filter: %s", state.FilterQuery)
		statusParts = append(statusParts, r.styles.Filter.Render(filterText))
	}

	content.WriteString(r.styles.Status.Render(strings.Join(statusParts, " | ")))

	if state.StatusMessage != "" {
		content.WriteString("\n")
		content.WriteString(state.StatusMessage)
	}
}

// renderLogPopup renders the git log popup
func (r *Renderer) renderLogPopup(content *strings.Builder, logContent string) {
	content.WriteString("\n\n")
	content.WriteString(r.styles.LogBox.Render(logContent))
}

// renderInfoPopup renders the repository info popup
func (r *Renderer) renderInfoPopup(content *strings.Builder, infoContent string) {
	content.WriteString("\n\n")
	content.WriteString(r.styles.InfoBox.Render(infoContent))
}

// matchesFilter checks if a repo matches the filter (simplified for now)
func (r *Renderer) matchesFilter(repo *domain.Repository, groupName string, filterQuery string) bool {
	if filterQuery == "" {
		return true
	}

	query := strings.ToLower(filterQuery)

	// Check if it's a status filter
	if strings.HasPrefix(query, "status:") {
		statusFilter := strings.TrimPrefix(query, "status:")
		return r.matchesStatusFilter(repo, statusFilter)
	}

	// Regular filter
	return strings.Contains(strings.ToLower(repo.Name), query) ||
		strings.Contains(strings.ToLower(repo.Path), query) ||
		strings.Contains(strings.ToLower(repo.Status.Branch), query) ||
		(groupName != "" && strings.Contains(strings.ToLower(groupName), query))
}

// matchesStatusFilter checks status-based filters
func (r *Renderer) matchesStatusFilter(repo *domain.Repository, filter string) bool {
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

// renderSortOptions renders the sort mode selection interface
func (r *Renderer) renderSortOptions(state ViewState) string {
	// Show only the current sort option
	if state.SortOptionIndex >= 0 && state.SortOptionIndex < len(modes.SortOptions) {
		option := modes.SortOptions[state.SortOptionIndex]
		sortLine := fmt.Sprintf("Sort by: %s - %s", option.Name, option.Description)
		helpLine := r.styles.Dim.Render("↑/↓ or j/k to change • Enter to accept • Esc to cancel")
		return sortLine + "\n" + helpLine
	}
	return ""
}

// renderLoadingScreen renders the loading screen
func (r *Renderer) renderLoadingScreen(state ViewState) string {
	lines := []string{}
	
	// Use full window height for centering
	fullHeight := state.Height
	if fullHeight < 10 {
		fullHeight = 10 // Minimum height
	}
	
	// Center the content vertically
	topPadding := (fullHeight - 6) / 2 // 6 lines for content (title + spacing + loading + spacing + hint)
	if topPadding < 0 {
		topPadding = 0
	}
	
	for i := 0; i < topPadding; i++ {
		lines = append(lines, "")
	}
	
	// Spinner
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := int(time.Now().UnixMilli()/80) % len(spinner)
	
	// Loading title (centered)
	titleStyle := r.styles.Title.Copy().MarginBottom(0).AlignHorizontal(lipgloss.Center).Width(state.Width)
	lines = append(lines, titleStyle.Render("gitagrip"))
	lines = append(lines, "")
	
	// Loading state
	loadingLine := fmt.Sprintf("%s %s", spinner[frame], state.LoadingState)
	if state.LoadingCount > 0 {
		loadingLine = fmt.Sprintf("%s %s (%d found)", spinner[frame], state.LoadingState, state.LoadingCount)
	}
	loadingStyle := r.styles.Scan.Copy().AlignHorizontal(lipgloss.Center).Width(state.Width)
	lines = append(lines, loadingStyle.Render(loadingLine))
	
	// Hint
	lines = append(lines, "")
	hintStyle := r.styles.Dim.Copy().AlignHorizontal(lipgloss.Center).Width(state.Width)
	
	if state.LoadingState == "Loading repositories..." {
		lines = append(lines, hintStyle.Render("Preparing your repository view"))
	} else if state.LoadingState == "Scanning for repositories..." {
		if state.LoadingCount > 0 {
			lines = append(lines, hintStyle.Render("Organizing repositories into groups"))
		} else {
			lines = append(lines, hintStyle.Render("Looking for Git repositories..."))
		}
	} else if state.LoadingState == "Initializing..." {
		lines = append(lines, hintStyle.Render("Setting up gitagrip"))
	}
	
	// Fill the rest with empty lines to ensure full height
	currentLines := len(lines)
	for i := currentLines; i < fullHeight; i++ {
		lines = append(lines, "")
	}
	
	return strings.Join(lines, "\n")
}
