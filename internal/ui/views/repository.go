package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	
	"gitagrip/internal/domain"
)

// RepositoryRenderer handles rendering of repository items
type RepositoryRenderer struct {
	styles           *Styles
	showAheadBehind  bool
}

// NewRepositoryRenderer creates a new repository renderer
func NewRepositoryRenderer(styles *Styles, showAheadBehind bool) *RepositoryRenderer {
	return &RepositoryRenderer{
		styles:          styles,
		showAheadBehind: showAheadBehind,
	}
}

// RenderRepository renders a repository item
func (r *RepositoryRenderer) RenderRepository(repo *domain.Repository, isSelected bool, indent int, 
	isMultiSelect bool, isFetching bool, isRefreshing bool, isPulling bool,
	searchQuery string, isRepoSelected bool) string {
	if repo == nil {
		return ""
	}
	
	// Background color for selection
	bgColor := ""
	if isSelected {
		bgColor = "238"
	}
	
	// Get status components
	status := r.getStatusIcon(repo, isFetching, isRefreshing, isPulling)
	branchName := r.formatBranchName(repo.Status.Branch)
	
	// Apply styles
	statusStyle := r.getStatusStyle(repo, isFetching, isRefreshing)
	if isSelected {
		statusStyle = statusStyle.Background(lipgloss.Color(bgColor))
	}
	
	// Branch styling
	branchColor := GetBranchColor(repo.Status.Branch)
	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(branchColor))
	if repo.Status.IsDirty || repo.Status.HasUntracked {
		branchStyle = branchStyle.Bold(true)
	}
	if isSelected {
		branchStyle = branchStyle.Background(lipgloss.Color(bgColor))
	}
	coloredBranch := branchStyle.Render(branchName)
	
	// Build the repository line
	var parts []string
	
	// Indentation
	if indent > 0 {
		parts = append(parts, strings.Repeat("  ", indent))
	}
	
	// Multi-select indicator
	if isMultiSelect {
		selectionIndicator := "[ ]"
		if isRepoSelected {
			selectionIndicator = "[x]"
		}
		selectionStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
		parts = append(parts, selectionStyle.Render(selectionIndicator))
		parts = append(parts, " ")
	}
	
	// Status icon
	if status != "" {
		parts = append(parts, statusStyle.Render(status))
		parts = append(parts, " ")
	}
	
	// Repository name (with search highlighting if applicable)
	repoName := repo.Name
	nameStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
	if searchQuery != "" && strings.Contains(strings.ToLower(repoName), strings.ToLower(searchQuery)) {
		repoName = r.highlightMatch(repoName, searchQuery, 
			nameStyle.Copy().Foreground(lipgloss.Color("226")), nameStyle)
	}
	parts = append(parts, nameStyle.Render(repoName))
	
	// Branch and status info
	parenStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
	parts = append(parts, parenStyle.Render(" ("))
	parts = append(parts, coloredBranch)
	
	// Ahead/behind info
	if r.showAheadBehind {
		aheadBehind := r.getAheadBehindText(repo.Status.AheadCount, repo.Status.BehindCount)
		if aheadBehind != "" {
			parts = append(parts, parenStyle.Render(" "))
			aheadBehindWithBg := lipgloss.NewStyle().Background(lipgloss.Color(bgColor)).Render(aheadBehind)
			parts = append(parts, aheadBehindWithBg)
		}
	}
	
	parts = append(parts, parenStyle.Render(")"))
	
	// Stash count
	if repo.Status.StashCount > 0 {
		stashText := fmt.Sprintf(" [%d stashed]", repo.Status.StashCount)
		stashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Background(lipgloss.Color(bgColor))
		parts = append(parts, stashStyle.Render(stashText))
	}
	
	return strings.Join(parts, "")
}

// getStatusIcon returns the appropriate status icon for a repository
func (r *RepositoryRenderer) getStatusIcon(repo *domain.Repository, isFetching, isRefreshing, isPulling bool) string {
	if isFetching {
		return "⟳"
	}
	if isRefreshing || isPulling {
		return "⟳"
	}
	if repo.Status.Error != "" {
		return "✗"
	}
	if repo.Status.IsDirty || repo.Status.HasUntracked {
		return "●"
	}
	return "✓"
}

// getStatusStyle returns the appropriate style for a repository status
func (r *RepositoryRenderer) getStatusStyle(repo *domain.Repository, isFetching, isRefreshing bool) lipgloss.Style {
	if isFetching {
		return r.styles.StatusFetching
	}
	if isRefreshing {
		return r.styles.StatusRefreshing
	}
	if repo.Status.Error != "" {
		return r.styles.StatusError
	}
	if repo.Status.IsDirty || repo.Status.HasUntracked {
		return r.styles.StatusWarning
	}
	return r.styles.StatusSuccess
}

// formatBranchName formats a branch name for display
func (r *RepositoryRenderer) formatBranchName(branch string) string {
	if branch == "" {
		return "no branch"
	}
	// Truncate long branch names
	if len(branch) > 30 {
		return branch[:27] + "..."
	}
	return branch
}

// getAheadBehindText formats ahead/behind counts
func (r *RepositoryRenderer) getAheadBehindText(ahead, behind int) string {
	if ahead > 0 && behind > 0 {
		return fmt.Sprintf("↑%d ↓%d", ahead, behind)
	} else if ahead > 0 {
		return fmt.Sprintf("↑%d", ahead)
	} else if behind > 0 {
		return fmt.Sprintf("↓%d", behind)
	}
	return ""
}

// highlightMatch highlights matching text within a string
func (r *RepositoryRenderer) highlightMatch(text, query string, highlightStyle, normalStyle lipgloss.Style) string {
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	
	index := strings.Index(lowerText, lowerQuery)
	if index == -1 {
		return normalStyle.Render(text)
	}
	
	// Split the text into parts
	before := text[:index]
	match := text[index : index+len(query)]
	after := text[index+len(query):]
	
	// Render with appropriate styles
	var result []string
	if before != "" {
		result = append(result, normalStyle.Render(before))
	}
	result = append(result, highlightStyle.Render(match))
	if after != "" {
		result = append(result, normalStyle.Render(after))
	}
	
	return strings.Join(result, "")
}