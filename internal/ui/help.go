package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/noborus/ov/oviewer"
)

// helpPagerMsg contains the result of a help pager command
type helpPagerMsg struct {
	err error
}

// HelpRenderer handles help content rendering
type HelpRenderer struct{}

// NewHelpRenderer creates a new help renderer
func NewHelpRenderer() *HelpRenderer {
	return &HelpRenderer{}
}

// renderHelpContent renders the help information
func (r *HelpRenderer) renderHelpContent(height int, scrollOffset int) string {
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
	help.WriteString("\n")

	// Selection section
	help.WriteString(sectionStyle.Render("Selection"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("Space"), descStyle.Render("Toggle selection")))
	help.WriteString(fmt.Sprintf("  %s          %s\n", keyStyle.Render("a/A"), descStyle.Render("Select/deselect all")))
	help.WriteString(fmt.Sprintf("  %s          %s\n", keyStyle.Render("Esc"), descStyle.Render("Clear selection")))
	help.WriteString("\n")

	// Repository actions section
	help.WriteString(sectionStyle.Render("Repository Actions"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("r"), descStyle.Render("Refresh repository status")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("f"), descStyle.Render("Fetch from remote")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("p"), descStyle.Render("Pull from remote")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("L"), descStyle.Render("View git log")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("D"), descStyle.Render("View git diff")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("i"), descStyle.Render("Show repository info & logs")))
	help.WriteString("\n")

	// Group management section
	help.WriteString(sectionStyle.Render("Group Management"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("z"), descStyle.Render("Toggle group")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("N"), descStyle.Render("Create new group (with selection)")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("m"), descStyle.Render("Move to group")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+R"), descStyle.Render("Rename group")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+J/K"), descStyle.Render("Move group up/down")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("H"), descStyle.Render("Hide selected repositories")))
	help.WriteString("\n")

	// Search & filter section
	help.WriteString(sectionStyle.Render("Search & Filter"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("/"), descStyle.Render("Search repositories")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("n"), descStyle.Render("Next search result")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+N"), descStyle.Render("Previous search result")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("F"), descStyle.Render("Filter repositories")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("s"), descStyle.Render("Sort options")))
	help.WriteString("\n")

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
		startLine := scrollOffset
		endLine := startLine + visibleHeight
		if endLine > totalLines {
			endLine = totalLines
		}
		visibleLines := lines[startLine:endLine]

		// Add scroll indicators
		if scrollOffset > 0 {
			visibleLines[0] = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑ (more above)")
		}
		if endLine < totalLines {
			visibleLines[len(visibleLines)-1] = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↓ (more below)")
		}

		return strings.Join(visibleLines, "\n")
	}

	return content
}

// RenderHelpContentPlain generates help content with colors for pager
func (r *HelpRenderer) RenderHelpContentPlain() string {
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
	help.WriteString("\n")

	// Selection section
	help.WriteString(sectionStyle.Render("Selection"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("Space"), descStyle.Render("Toggle selection")))
	help.WriteString(fmt.Sprintf("  %s          %s\n", keyStyle.Render("a/A"), descStyle.Render("Select/deselect all")))
	help.WriteString(fmt.Sprintf("  %s          %s\n", keyStyle.Render("Esc"), descStyle.Render("Clear selection")))
	help.WriteString("\n")

	// Repository actions section
	help.WriteString(sectionStyle.Render("Repository Actions"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("r"), descStyle.Render("Refresh repository status")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("f"), descStyle.Render("Fetch from remote")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("p"), descStyle.Render("Pull from remote")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("L"), descStyle.Render("View git log")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("D"), descStyle.Render("View git diff")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("i"), descStyle.Render("Show repository info & logs")))
	help.WriteString("\n")

	// Group management section
	help.WriteString(sectionStyle.Render("Group Management"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("z"), descStyle.Render("Toggle group")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("N"), descStyle.Render("Create new group (with selection)")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("m"), descStyle.Render("Move to group")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+R"), descStyle.Render("Rename group")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+J/K"), descStyle.Render("Move group up/down")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("H"), descStyle.Render("Hide selected repositories")))
	help.WriteString("\n")

	// Search & filter section
	help.WriteString(sectionStyle.Render("Search & Filter"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("/"), descStyle.Render("Search repositories")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("n"), descStyle.Render("Next search result")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("Shift+N"), descStyle.Render("Previous search result")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("F"), descStyle.Render("Filter repositories")))
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("s"), descStyle.Render("Sort options")))
	help.WriteString("\n")

	// Filter examples (using italic style)
	filterStyle := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("241"))
	help.WriteString(filterStyle.Render("  Filter examples: status:dirty, status:clean, status:ahead"))
	help.WriteString("\n")

	// Other section
	help.WriteString(sectionStyle.Render("Other"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s            %s\n", keyStyle.Render("?"), descStyle.Render("Toggle this help")))
	help.WriteString(fmt.Sprintf("  %s            %s", keyStyle.Render("q"), descStyle.Render("Quit")))

	return help.String()
}

// HelpOps handles help operations
type HelpOps struct {
	program *tea.Program // reference to Bubble Tea program for terminal management
}

// NewHelpOps creates a new help operations instance
func NewHelpOps(program *tea.Program) *HelpOps {
	return &HelpOps{
		program: program,
	}
}

// ShowHelpInPager shows help content using ov pager
func (h *HelpOps) ShowHelpInPager(helpContent string) error {
	if h.program == nil {
		return fmt.Errorf("program not set")
	}

	// Release terminal control to run ov
	if err := h.program.ReleaseTerminal(); err != nil {
		return err
	}

	// Ensure terminal is restored even if ov fails
	defer func() {
		// Small delay to ensure ov has fully exited before restoring terminal
		time.Sleep(100 * time.Millisecond)
		_ = h.program.RestoreTerminal() // Ignore error as we're in defer context
	}()

	// Create a reader from the help content string
	reader := strings.NewReader(helpContent)

	// Create oviewer root from the reader
	root, err := oviewer.NewRoot(reader)
	if err != nil {
		return err
	}

	// Configure ov to not write on exit (to avoid messing with our screen)
	config := oviewer.NewConfig()
	config.IsWriteOnExit = false
	config.IsWriteOriginal = false

	// Add vim-like navigation
	configureVimKeyBindings(&config)

	root.SetConfig(config)

	// Run the oviewer (this will take over the terminal)
	return root.Run()
}
