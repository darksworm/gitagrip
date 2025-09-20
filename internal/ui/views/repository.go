package views

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss/v2"

    "gitagrip/internal/domain"
)

// RepositoryRenderer handles rendering of repository items
type RepositoryRenderer struct {
	styles          *Styles
	showAheadBehind bool
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
	searchQuery string, isRepoSelected bool, width int) string {
	if repo == nil {
		return ""
	}

	// Background color for selection
	bgColor := ""
	if isSelected && isRepoSelected && isMultiSelect {
		// Cursor on selected item - use distinct color
		bgColor = "33" // Blue background for cursor on selected item
	} else if isSelected {
		// Cursor on unselected item
		bgColor = "238"
	} else if isRepoSelected && isMultiSelect {
		// Selected item without cursor
		bgColor = "240"
	}

	// Get status components
	status := r.getStatusIcon(repo, isFetching, isRefreshing, isPulling)
	branchName := r.formatBranchName(repo.Status.Branch)

	// Apply styles
	statusStyle := r.getStatusStyle(repo, isFetching, isRefreshing)
	if bgColor != "" {
		statusStyle = statusStyle.Background(lipgloss.Color(bgColor))
	}

	// Branch styling
	branchColor := GetBranchColor(repo.Status.Branch)
	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(branchColor))

	// Make main/master branches bold
	if repo.Status.Branch == "main" || repo.Status.Branch == "master" {
		branchStyle = branchStyle.Bold(true)
	}

	// Apply background color if selected
	if bgColor != "" {
		branchStyle = branchStyle.Background(lipgloss.Color(bgColor))
	}
	coloredBranch := branchStyle.Render(branchName)

	// Build the repository line
	var parts []string

	// Indentation
	if indent > 0 {
		indentText := strings.Repeat("  ", indent)
		if bgColor != "" {
			indentStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
			parts = append(parts, indentStyle.Render(indentText))
		} else {
			parts = append(parts, indentText)
		}
	}

	// No checkbox needed - we use background color to indicate selection

	// Status icon
	if status != "" {
		parts = append(parts, statusStyle.Render(status))
		if bgColor != "" {
			spacerStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
			parts = append(parts, spacerStyle.Render(" "))
		} else {
			parts = append(parts, " ")
		}
	}

	// Repository name (with search highlighting if applicable)
	repoName := repo.DisplayName
	if repoName == "" {
		repoName = repo.Name // Fallback to Name if DisplayName not set
	}
	nameStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
	if searchQuery != "" && strings.Contains(strings.ToLower(repoName), strings.ToLower(searchQuery)) {
		highlightStyle := nameStyle
		highlightStyle = highlightStyle.Foreground(lipgloss.Color("226"))
		repoName = r.highlightMatch(repoName, searchQuery, highlightStyle, nameStyle)
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

	// Join the parts
	line := strings.Join(parts, "")

	// Pad the line to full width with background color if selected
	if bgColor != "" && width > 0 {
		// Calculate the current line length without ANSI codes
		lineLen := lipgloss.Width(line)
		if lineLen < width {
			padding := strings.Repeat(" ", width-lineLen)
			paddingStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
			line = line + paddingStyle.Render(padding)
		}
	}

	return line
}

// getStatusIcon returns the appropriate status icon for a repository
func (r *RepositoryRenderer) getStatusIcon(repo *domain.Repository, isFetching, isRefreshing, isPulling bool) string {
	if isFetching {
		return "⟳"
	}
	if isRefreshing || isPulling {
		return "⟳"
	}
	// Check for command errors (red danger sign)
	if repo.HasError {
		return "⚠"
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
	// Check for command errors (red danger style)
	if repo.HasError {
		return r.styles.StatusError
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
