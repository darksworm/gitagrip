package views

import (
    "github.com/charmbracelet/lipgloss/v2"
)

// Styles contains all the style definitions for the UI
type Styles struct {
	Title            lipgloss.Style
	Confirm          lipgloss.Style
	Scan             lipgloss.Style
	Dim              lipgloss.Style
	Status           lipgloss.Style
	Filter           lipgloss.Style
	LogBox           lipgloss.Style
	InfoBox          lipgloss.Style
	Help             lipgloss.Style
	Main             lipgloss.Style
	Scroll           lipgloss.Style
	Highlight        lipgloss.Style
	HighlightBg      lipgloss.Style
	StatusError      lipgloss.Style
	StatusWarning    lipgloss.Style
	StatusLoading    lipgloss.Style
	StatusSuccess    lipgloss.Style
	StatusFetching   lipgloss.Style
	StatusRefreshing lipgloss.Style
	SelectionBg      lipgloss.Style
}

// NewStyles creates a new Styles instance with default values
func NewStyles() *Styles {
	return &Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginBottom(1),
		Confirm: lipgloss.NewStyle().Bold(true),
		Scan:    lipgloss.NewStyle().Foreground(lipgloss.Color("33")),
		Dim:     lipgloss.NewStyle().Faint(true),
		Status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1).
			MarginBottom(1),
		Filter: lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // yellow
		LogBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			BorderForeground(lipgloss.Color("244")),
		InfoBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			BorderForeground(lipgloss.Color("244")),
		Help: lipgloss.NewStyle().Faint(true),
		Main: lipgloss.NewStyle().
			Padding(1, 2).
			MaxHeight(100), // Will be dynamically adjusted
		Scroll:           lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true),
		Highlight:        lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true),
		HighlightBg:      lipgloss.NewStyle().Background(lipgloss.Color("238")),
		StatusError:      lipgloss.NewStyle().Foreground(lipgloss.Color("203")), // red
		StatusWarning:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // yellow
		StatusLoading:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")), // gray
		StatusSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("78")),  // green
		StatusFetching:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // yellow
		StatusRefreshing: lipgloss.NewStyle().Foreground(lipgloss.Color("51")),  // cyan
		SelectionBg:      lipgloss.NewStyle().Background(lipgloss.Color("238")),
	}
}

// GetBranchColor returns the appropriate color for a git branch
func GetBranchColor(branchName string) string {
	switch branchName {
	case "main", "master":
		return "78" // green - production branches
	case "develop", "dev":
		return "33" // blue - development branches
	default:
		if branchName == "" || branchName == "HEAD" {
			return "203" // red (detached HEAD or error)
		}
		// Compute a color based on branch name hash
		return computeBranchColor(branchName)
	}
}

// computeBranchColor generates a consistent color for a branch name
func computeBranchColor(branchName string) string {
	// List of good contrasting colors (avoiding too dark or too light)
	colors := []string{
		"39",  // bright blue
		"41",  // bright green
		"43",  // cyan
		"45",  // light blue
		"50",  // turquoise
		"51",  // bright cyan
		"75",  // steel blue
		"84",  // spring green
		"87",  // light cyan
		"99",  // purple
		"111", // sky blue
		"117", // light blue
		"120", // light green
		"123", // bright cyan
		"135", // violet
		"141", // purple
		"147", // light purple
		"156", // green-cyan
		"159", // pale blue
		"165", // magenta
		"171", // violet
		"177", // pink
		"183", // plum
		"189", // light purple
		"198", // hot pink
		"201", // bright pink
		"204", // orange red
		"207", // pink
		"208", // orange
		"209", // light orange
		"213", // pink-orange
		"214", // yellow-orange
		"219", // light pink
		"220", // gold
		"221", // light yellow
		"222", // peach
		"225", // light pink
		"226", // yellow
		"227", // light yellow
		"228", // pale yellow
		"229", // wheat
	}

	// Simple hash: sum of character codes
	hash := 0
	for _, ch := range branchName {
		hash += int(ch)
		hash = hash * 17 // multiply by prime for better distribution
	}

	// Use modulo to pick a color
	colorIndex := hash % len(colors)
	if colorIndex < 0 {
		colorIndex = -colorIndex
	}

	return colors[colorIndex]
}
