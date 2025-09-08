package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PopupRenderer handles popup/modal rendering
type PopupRenderer struct {
	styles *Styles
}

// NewPopupRenderer creates a new popup renderer
func NewPopupRenderer(styles *Styles) *PopupRenderer {
	return &PopupRenderer{
		styles: styles,
	}
}

// RenderPopupOverlay renders a popup overlay on top of main content
func (pr *PopupRenderer) RenderPopupOverlay(mainContent, popupContent string, height, width int, popupStyle lipgloss.Style) string {
	// Create a centered modal-style popup
	// First, measure the popup content
	lines := strings.Split(popupContent, "\n")
	contentHeight := len(lines)
	contentWidth := 0
	for _, line := range lines {
		if lipgloss.Width(line) > contentWidth {
			contentWidth = lipgloss.Width(line)
		}
	}

	// Ensure minimum sizes
	if contentWidth < 60 {
		contentWidth = 60
	}
	if contentHeight < 10 {
		contentHeight = 10
	}

	// Add padding for the border and internal padding
	popupWidth := contentWidth + 4   // 2 for border, 2 for horizontal padding
	popupHeight := contentHeight + 2 // 2 for border only

	// Ensure popup fits on screen with generous margins (6 chars each side for safety)
	maxPopupWidth := width - 12 // 6 chars margin on each side
	if maxPopupWidth < 60 {
		maxPopupWidth = 60 // Minimum width
	}
	if popupWidth > maxPopupWidth {
		popupWidth = maxPopupWidth
	}
	if popupHeight > height-4 {
		popupHeight = height - 4
	}

	// Apply the style with the calculated dimensions
	styledPopup := popupStyle.
		Width(popupWidth).
		Height(popupHeight).
		MaxWidth(maxPopupWidth).
		MaxHeight(height - 4).
		Render(popupContent)

	// Center the popup safely
	centeredPopup := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, styledPopup)

	// Create a semi-transparent overlay effect by dimming the background
	mainLines := strings.Split(mainContent, "\n")
	overlayLines := strings.Split(centeredPopup, "\n")

	// Ensure we have enough lines
	for len(overlayLines) < len(mainLines) {
		overlayLines = append(overlayLines, "")
	}

	result := overlayLines

	return strings.Join(result, "\n")
}
