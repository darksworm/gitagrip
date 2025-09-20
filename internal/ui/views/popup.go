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
	// Render the popup with its style without forcing width/height – keep it tight
	styledPopup := popupStyle.Render(popupContent)

	// Compute modal placement using actual rendered size
	modalW := lipgloss.Width(styledPopup)
	modalH := lipgloss.Height(styledPopup)
	if modalW > width-6 { // keep a small margin
		modalW = width - 6
	}
	if modalH > height-4 {
		modalH = height - 4
	}
	x := (width - modalW) / 2
	y := (height - modalH) / 2

	// Base greyscale layer, but keep the target repository line colored
	targetName := extractTitlePlain(popupContent)
	grayBase := desaturateKeeping(mainContent, targetName)
	baseLayer := lipgloss.NewLayer(grayBase)

	// Modal layer on top (only its bounding box, not whole lines)
	modalLayer := lipgloss.NewLayer(styledPopup).X(x).Y(y).Z(1)

	// Compose layers without erasing left/right content
	canvas := lipgloss.NewCanvas(baseLayer, modalLayer)
	return canvas.Render()
}

// ANSI escape sequence regex to strip styles/colors
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// desaturateANSI strips ANSI color/style codes and recolors text dim gray
func desaturateANSI(s string) string {
	plain := ansiRE.ReplaceAllString(s, "")
	return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(plain)
}

// extractTitlePlain returns the first line of popup content without ANSI (repo name in Info modal)
func extractTitlePlain(popup string) string {
	// first line before newline
	for i := 0; i < len(popup); i++ {
		if popup[i] == '\n' {
			return ansiRE.ReplaceAllString(popup[:i], "")
		}
	}
	return ansiRE.ReplaceAllString(popup, "")
}

// desaturateKeeping turns everything greyscale except lines containing keepSubstr (plain text match)
func desaturateKeeping(s, keepSubstr string) string {
	if keepSubstr == "" {
		return desaturateANSI(s)
	}
	lines := strings.Split(s, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		plain := ansiRE.ReplaceAllString(line, "")
		if keepSubstr != "" && strings.Contains(plain, keepSubstr) {
			// keep original colored line
			out[i] = line
		} else {
			out[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(plain)
		}
	}
	return strings.Join(out, "\n")
}
