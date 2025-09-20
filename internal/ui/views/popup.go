package views

import (
    "regexp"
    "strings"

    "github.com/charmbracelet/lipgloss/v2"
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

    // Center the popup into a full-screen overlay string
    centeredPopup := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, styledPopup)

    // Desaturate base content to produce a greyscale effect under the modal
    grayBase := desaturateANSI(mainContent)

    // Compose overlay over grayscale base: prefer overlay rows when they contain visible content
    baseLines := strings.Split(grayBase, "\n")
    overlayLines := strings.Split(centeredPopup, "\n")
    // Normalize lengths
    if len(baseLines) < len(overlayLines) {
        diff := len(overlayLines) - len(baseLines)
        for i := 0; i < diff; i++ { baseLines = append(baseLines, "") }
    } else if len(overlayLines) < len(baseLines) {
        diff := len(baseLines) - len(overlayLines)
        for i := 0; i < diff; i++ { overlayLines = append(overlayLines, "") }
    }
    out := make([]string, len(baseLines))
    for i := range baseLines {
        if strings.TrimSpace(overlayLines[i]) != "" {
            out[i] = overlayLines[i]
        } else {
            out[i] = baseLines[i]
        }
    }
    return strings.Join(out, "\n")
}

// ANSI escape sequence regex to strip styles/colors
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// desaturateANSI strips ANSI color/style codes and recolors text dim gray
func desaturateANSI(s string) string {
    plain := ansiRE.ReplaceAllString(s, "")
    return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(plain)
}
