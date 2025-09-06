package views

import (
	"github.com/charmbracelet/lipgloss"
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
		Confirm:          lipgloss.NewStyle().Bold(true),
		Scan:             lipgloss.NewStyle().Foreground(lipgloss.Color("33")),
		Dim:              lipgloss.NewStyle().Faint(true),
		Status:           lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1).
			MarginBottom(1),
		Filter:           lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // yellow
		LogBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(1).
			MarginBottom(1).
			Width(80).
			Height(20).
			BorderForeground(lipgloss.Color("241")),
		InfoBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(1).
			MarginBottom(1).
			Width(60).
			Height(10).
			BorderForeground(lipgloss.Color("241")),
		Help:             lipgloss.NewStyle().Faint(true),
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
		return "78" // green
	case "develop", "dev":
		return "33" // blue
	default:
		if branchName == "" || branchName == "HEAD" {
			return "203" // red (detached HEAD or error)
		}
		return "214" // yellow for feature branches
	}
}