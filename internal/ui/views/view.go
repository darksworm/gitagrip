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
	Width            int
	Height           int
	Repositories     map[string]*domain.Repository
	Groups           map[string]*domain.Group
	OrderedGroups    []string
	SelectedIndex    int
	SelectedRepos    map[string]bool
	RefreshingRepos  map[string]bool
	FetchingRepos    map[string]bool
	PullingRepos     map[string]bool
	ExpandedGroups   map[string]bool
	Scanning         bool
	StatusMessage    string
	ShowHelp         bool
	HelpScrollOffset int
	ShowLog          bool
	LogContent       string
	ShowInfo         bool
	InfoContent      string
	ViewportOffset   int
	ViewportHeight   int
	SearchQuery      string
	FilterQuery      string
	IsFiltered       bool
	ShowAheadBehind  bool
	HelpModel        help.Model
	DeleteTarget     string
	TextInput        string
	InputMode        string
	UngroupedRepos   []string
	SortOptionIndex  int
	LoadingState     string
	LoadingCount     int
}

// Renderer handles all view rendering
type Renderer struct {
	styles      *Styles
	repoRender  *RepositoryRenderer
	groupRender *GroupRenderer
	popupRender *PopupRenderer
}

// NewRenderer creates a new renderer
func NewRenderer(showAheadBehind bool) *Renderer {
	styles := NewStyles()
	return &Renderer{
		styles:      styles,
		repoRender:  NewRepositoryRenderer(styles, showAheadBehind),
		groupRender: NewGroupRenderer(styles),
		popupRender: NewPopupRenderer(styles),
	}
}

// Render produces the complete view
func (r *Renderer) Render(state ViewState) string {
	content := &strings.Builder{}

	// Title with loading indicator
	logo := r.styles.Title.Render("gitagrip")

	// Build loading indicators
	loadingIndicators := []string{}

	if state.Scanning {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := int(time.Now().UnixMilli()/80) % len(spinner)
		loadingIndicators = append(loadingIndicators, fmt.Sprintf("%s Scanning", spinner[frame]))
	}

	if len(state.RefreshingRepos) > 0 {
		loadingIndicators = append(loadingIndicators, fmt.Sprintf("↻ Refreshing %d", len(state.RefreshingRepos)))
	}

	if len(state.FetchingRepos) > 0 {
		loadingIndicators = append(loadingIndicators, fmt.Sprintf("↓ Fetching %d", len(state.FetchingRepos)))
	}

	if len(state.PullingRepos) > 0 {
		loadingIndicators = append(loadingIndicators, fmt.Sprintf("↓ Pulling %d", len(state.PullingRepos)))
	}

	// Build the title line with right-aligned indicators
	var titleLine string
	if len(loadingIndicators) > 0 || state.FilterQuery != "" {
		// Calculate widths
		logoWidth := lipgloss.Width(logo)

		// Build right side content
		rightContent := ""
		if len(loadingIndicators) > 0 {
			rightContent = r.styles.Dim.Render(strings.Join(loadingIndicators, " | "))
		}
		if state.FilterQuery != "" {
			filterText := r.styles.Filter.Render(fmt.Sprintf("[Filter: %s]", state.FilterQuery))
			if rightContent != "" {
				rightContent = fmt.Sprintf("%s  %s", rightContent, filterText)
			} else {
				rightContent = filterText
			}
		}

		// Calculate padding needed
		rightWidth := lipgloss.Width(rightContent)
		// Use a default width if state.Width is not set
		termWidth := state.Width
		if termWidth <= 0 {
			termWidth = 80 // Default terminal width
		}
		availableWidth := termWidth - 4 // Account for main container padding
		paddingWidth := availableWidth - logoWidth - rightWidth

		if paddingWidth > 0 {
			padding := strings.Repeat(" ", paddingWidth)
			titleLine = fmt.Sprintf("%s%s%s", logo, padding, rightContent)
		} else {
			// If not enough space, just show with minimal spacing
			titleLine = fmt.Sprintf("%s  %s", logo, rightContent)
		}
	} else {
		titleLine = logo
	}

	content.WriteString(titleLine)
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
	mainContent := ""
	if state.Scanning && len(state.Repositories) == 0 {
		// Don't show duplicate scanning message - it's already in the title
		mainContent = r.styles.Dim.Render("Looking for repositories...")
	} else if len(state.Repositories) == 0 {
		mainContent = r.styles.Dim.Render("No repositories found. Press F for full scan.")
	} else {
		mainContent = r.renderRepositoryList(state)
	}

	// Add main content
	content.WriteString(mainContent)

	// Calculate help text (shown at bottom when no popups are visible)
	helpText := ""
	if !state.ShowHelp && !state.ShowLog && !state.ShowInfo {
		helpText = r.styles.Help.Render("Press ? for help")
	}

	// If we have help text, add padding to push it to the bottom
	if helpText != "" {
		// Count current lines
		currentContent := content.String()
		currentLines := strings.Count(currentContent, "\n") + 1

		// Account for container padding (1 top, 1 bottom from Padding(1, 2))
		availableLines := state.Height - 2
		if availableLines <= 0 {
			availableLines = 22 // Default terminal height minus padding
		}

		// Help takes 1 line
		helpLines := 1

		// Calculate padding needed
		paddingNeeded := availableLines - currentLines - helpLines

		// Add padding
		if paddingNeeded > 0 {
			content.WriteString(strings.Repeat("\n", paddingNeeded))
		}

		// Add help
		content.WriteString("\n")
		content.WriteString(helpText)
	}

	// Apply main container style
	mainStyle := r.styles.Main.MaxHeight(state.Height)
	finalContent := mainStyle.Render(content.String())

	// Overlay popups on top of main content
	if state.ShowLog && state.LogContent != "" {
		return r.popupRender.RenderPopupOverlay(finalContent, state.LogContent, state.Height, state.Width, r.styles.LogBox)
	}

	if state.ShowInfo && state.InfoContent != "" {
		return r.popupRender.RenderPopupOverlay(finalContent, state.InfoContent, state.Height, state.Width, r.styles.InfoBox)
	}

	if state.ShowHelp {
		helpContent := r.renderHelpContent(state.Height, state.HelpScrollOffset)
		return r.popupRender.RenderPopupOverlay(finalContent, helpContent, state.Height, state.Width, r.styles.InfoBox)
	}

	return finalContent
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
			allReposSelected := true
			hasSelectedRepos := false

			if isExpanded {
				// Count visible repos in group and check selection
				for _, repoPath := range group.Repos {
					if repo, ok := state.Repositories[repoPath]; ok {
						if r.matchesFilter(repo, groupName, state.FilterQuery) {
							repoCount++
							if state.SelectedRepos[repoPath] {
								hasSelectedRepos = true
							} else {
								allReposSelected = false
							}
						}
					}
				}
			} else {
				// For collapsed groups, check all repos
				repoCount = len(group.Repos)
				for _, repoPath := range group.Repos {
					if state.SelectedRepos[repoPath] {
						hasSelectedRepos = true
					} else {
						allReposSelected = false
					}
				}
			}

			// Only highlight if there are repos and all are selected
			groupIsFullySelected := repoCount > 0 && allReposSelected && hasSelectedRepos

			header := r.groupRender.RenderGroupHeader(group, isExpanded, isSelected, state.SearchQuery, repoCount, state.Width, groupIsFullySelected)
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

// renderHelpContent renders the help information
func (r *Renderer) renderHelpContent(height int, scrollOffset int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var help strings.Builder

	// Title
	help.WriteString(titleStyle.Render("GitaGrip Help"))
	help.WriteString("\n")

	// Navigation section
	help.WriteString(sectionStyle.Render("Navigation"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("↑/↓, j/k"), descStyle.Render("Navigate up/down")))
	help.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("←/→, h/l"), descStyle.Render("Collapse/expand groups")))
	help.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("PgUp/PgDn"), descStyle.Render("Page up/down")))
	help.WriteString(fmt.Sprintf("  %s       %s\n", keyStyle.Render("gg/G"), descStyle.Render("Go to top/bottom")))

	// Selection section
	help.WriteString(sectionStyle.Render("Selection"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("Space"), descStyle.Render("Toggle selection")))
	help.WriteString(fmt.Sprintf("  %s          %s\n", keyStyle.Render("a/A"), descStyle.Render("Select/deselect all")))
	help.WriteString(fmt.Sprintf("  %s          %s\n", keyStyle.Render("Esc"), descStyle.Render("Clear selection")))

	// Repository actions section
	help.WriteString(sectionStyle.Render("Repository Actions"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("r"), descStyle.Render("Refresh repository status")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("f"), descStyle.Render("Fetch from remote")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("p"), descStyle.Render("Pull from remote")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("L"), descStyle.Render("View git log")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("D"), descStyle.Render("View git diff")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("i"), descStyle.Render("Show repository info & logs")))

	// Group management section
	help.WriteString(sectionStyle.Render("Group Management"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("z"), descStyle.Render("Toggle group")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("N"), descStyle.Render("Create new group (with selection)")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("m"), descStyle.Render("Move to group")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+R"), descStyle.Render("Rename group")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+J/K"), descStyle.Render("Move group up/down")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("H"), descStyle.Render("Hide selected repositories")))

	// Search & filter section
	help.WriteString(sectionStyle.Render("Search & Filter"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("/"), descStyle.Render("Search repositories")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("n"), descStyle.Render("Next search result")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+N"), descStyle.Render("Previous search result")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("F"), descStyle.Render("Filter repositories")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("s"), descStyle.Render("Sort options")))

	// Filter examples
	help.WriteString(lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("241")).Render("  Filter examples: status:dirty, status:clean, status:ahead"))
	help.WriteString("\n")

	// Other section
	help.WriteString(sectionStyle.Render("Other"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("?"), descStyle.Render("Toggle this help")))
	help.WriteString(fmt.Sprintf("  %s            %s", keyStyle.Render("q"), descStyle.Render("Quit")))

	// Split into lines for scrolling
	content := help.String()
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Calculate visible window (account for popup border and padding)
	visibleHeight := height - 4
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	// Apply scrolling
	if totalLines > visibleHeight {
		// Ensure scroll offset is valid
		maxOffset := totalLines - visibleHeight
		if scrollOffset > maxOffset {
			scrollOffset = maxOffset
		}
		if scrollOffset < 0 {
			scrollOffset = 0
		}

		// Extract visible lines
		endLine := scrollOffset + visibleHeight
		if endLine > totalLines {
			endLine = totalLines
		}
		lines = lines[scrollOffset:endLine]

		// Add scroll indicators
		if scrollOffset > 0 {
			lines[0] = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑ (more above)")
		}
		if endLine < totalLines {
			lines[len(lines)-1] = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↓ (more below)")
		}
	}

	return strings.Join(lines, "\n")
}
