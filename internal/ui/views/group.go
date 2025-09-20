package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"

	"gitagrip/internal/domain"
)

// GroupRenderer handles rendering of group headers
type GroupRenderer struct {
	styles *Styles
}

// NewGroupRenderer creates a new group renderer
func NewGroupRenderer(styles *Styles) *GroupRenderer {
	return &GroupRenderer{
		styles: styles,
	}
}

// RenderGroupHeader renders a group header
func (g *GroupRenderer) RenderGroupHeader(group *domain.Group, isExpanded bool, isSelected bool,
	searchQuery string, repoCount int, width int, groupIsFullySelected bool) string {

	// Determine arrow
	arrow := "▶"
	if isExpanded {
		arrow = "▼"
	}

	// Build group name with search highlighting
	groupName := group.Name
	if searchQuery != "" && strings.Contains(strings.ToLower(groupName), strings.ToLower(searchQuery)) {
		groupName = g.highlightMatch(groupName, searchQuery, g.styles.Highlight, lipgloss.NewStyle())
	}

	// Format the complete line
	line := fmt.Sprintf("%s %s (%d)", arrow, groupName, repoCount)

	// Apply background color based on selection state
	var bgColor string
	if isSelected && groupIsFullySelected {
		bgColor = "33" // Blue background for cursor on fully selected group
	} else if isSelected {
		bgColor = "238" // Darker background for cursor selection
	} else if groupIsFullySelected {
		bgColor = "240" // Lighter background when all repos are selected
	}

	// Apply background if needed
	if bgColor != "" {
		// Pad the line to full width
		if width > 0 {
			lineLen := lipgloss.Width(line)
			if lineLen < width {
				line = line + strings.Repeat(" ", width-lineLen)
			}
		}
		style := lipgloss.NewStyle().Background(lipgloss.Color(bgColor))
		return style.Render(line)
	}

	// Apply dim style for hidden group
	if group.Name == "_Hidden" {
		return g.styles.Dim.Render(line)
	}

	return line
}

// highlightMatch highlights matching text within a string
func (g *GroupRenderer) highlightMatch(text, query string, highlightStyle, normalStyle lipgloss.Style) string {
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
