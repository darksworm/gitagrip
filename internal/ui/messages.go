package ui

import (
	"time"

	"gitagrip/internal/eventbus"
)

// EventMsg wraps a domain event for the UI
type EventMsg struct {
	Event eventbus.DomainEvent
}

// tickMsg is sent on a timer for animations
type tickMsg time.Time

// gitLogMsg contains the result of a git log command
type gitLogMsg struct {
	repoPath string
	content  string
	err      error
}

// gitDiffMsg contains the result of a git diff command
type gitDiffMsg struct {
	repoPath string
	content  string
	err      error
}

// gitLogPagerMsg contains the result of a git log pager command
type gitLogPagerMsg struct {
	repoPath string
	err      error
}

// gitDiffPagerMsg contains the result of a git diff pager command
type gitDiffPagerMsg struct {
	repoPath string
	err      error
}

// quitMsg signals that the application should quit
type quitMsg struct {
	saveConfig bool
}

// pauseRenderingMsg signals to pause Bubble Tea rendering
type pauseRenderingMsg struct{}

// resumeRenderingMsg signals to resume Bubble Tea rendering
type resumeRenderingMsg struct{}
