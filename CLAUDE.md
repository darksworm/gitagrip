# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Memory Bank via MCP

I'm an expert engineer whose memory resets between sessions. I rely ENTIRELY on my Memory Bank, accessed via MCP tools, and MUST read ALL memory bank files before EVERY task.

### Key Commands

1. "follow your custom instructions"
   - Triggers Pre-Flight Validation (*a)
   - Follows Memory Bank Access Pattern (*f)
   - Executes appropriate Mode flow (Plan/Act)

2. "initialize memory bank"
   - Follows Pre-Flight Validation (*a)
   - Creates new project if needed
   - Establishes core files structure (*f)

3. "update memory bank"
   - Triggers Documentation Updates (*d)
   - Performs full file re-read
   - Updates based on current state

### Memory Bank Structure

**Core Files:**
- `projectbrief.md` - Core requirements/goals
- `productContext.md` - Problem context/solutions
- `systemPatterns.md` - Architecture/patterns
- `techContext.md` - Tech stack/setup
- `activeContext.md` - Current focus/decisions
- `progress.md` - Status/roadmap
- `.clinerules` - Project-specific patterns and rules

**Access Pattern:**
- Always read in hierarchical order
- Update in reverse order (progress → active → others)
- .clinerules accessed throughout process
- Custom files integrated based on project needs

### Documentation Update Triggers
- ≥25% code impact changes
- New pattern discovery
- User request "update memory bank"
- Context ambiguity detected

## Project Overview

GitaGrip (formerly YARG) is a fast TUI application for discovering, grouping, and inspecting multiple Git repositories (read-only). The project has been rewritten in Go following an event-driven architecture.

## Project Status

The Go implementation is now active with core functionality implemented. The application uses an event-driven architecture with services communicating via a central event bus.

## Tech Stack & Dependencies

**Go Implementation:**
- **TUI**: Bubble Tea (github.com/charmbracelet/bubbletea)
- **Git**: Command-line git operations via exec.Command
- **FS scan**: filepath.WalkDir for repository discovery
- **Concurrency**: Goroutines with context-based cancellation
- **Config**: JSON configuration in ~/.config/gitagrip/
- **Styling**: Lipgloss for terminal styling
- **Architecture**: Event-driven with central event bus

## Architecture

The application follows a modular structure:

### Core Modules
- **Config Module**: JSON configuration handling (reads from `~/.config/gitagrip/config.json`)
- **Discovery Module**: Filesystem scanning for Git repositories using filepath.WalkDir
- **Git Operations Module**: Git status, fetch, and log operations via exec.Command
- **UI Module**: Bubble Tea-based terminal UI with keyboard navigation
- **Event Bus**: Central event-driven communication between services
- **Domain Models**: Repository, Group, and Status representations

### Configuration Schema
```json
{
  "version": 1,
  "base_dir": "/path/to/repos",
  "ui": {
    "show_ahead_behind": true,
    "autosave_on_exit": true
  },
  "groups": {
    "Work": ["/path/to/work-repo1", "/path/to/work-repo2"],
    "Personal": ["/path/to/personal-repo"]
  }
}
```

## Key Features (Implemented)

1. **Recursive repository discovery** - Auto-detect Git repos in base directory ✅
2. **Repository status overview** - Show branch, dirty/clean state, ahead/behind counts ✅
3. **Read-only Git operations** - Status checking (fetch and log planned)
4. **Repository grouping** - Manual grouping with expand/collapse UI ✅
5. **JSON configuration persistence** - Save repo lists and custom groups ✅
6. **Non-blocking TUI** - Background operations with event-driven updates ✅

## Implementation Milestones

Follow the milestone structure from `plan.md`:
- **M0**: Bootstrap - Basic TUI with clean exit
- **M1**: Config loading and CLI args
- **M2**: Repository discovery with background scanning  
- **M3**: Git status aggregation
- **M4**: Grouping and list UI
- **M5**: Fetch and refresh operations
- **M6**: Commit log popup
- **M7**: State persistence and polish

## Key Design Principles

- **Read-only operations**: No write ops (commits/pushes/merges) in v1
- **Non-blocking UI**: All I/O operations run on background goroutines
- **Event-driven architecture**: Services communicate via central event bus
- **Clean code**: Small focused modules, strong typing, interfaces
- **Cross-platform**: Terminal-based for portability
- **Performance**: Go + concurrency for handling many repos efficiently

## Key Controls (Planned)
- `↑↓` or `j/k`: Navigate repos
- `←→` or `h/l`: Collapse/expand groups  
- `r`: Rescan statuses (fast)
- `F`: Full rescan (re-discovery)
- `f`: Fetch selected repo
- `l`: Open log popup
- `?`: Help screen
- `q`: Quit

## Development Guidelines

- Use semantic commits
- Write tests for pure logic (grouping, config, git status parsing)
- Add logging for debugging operations
- Handle errors gracefully (network failures, missing repos, etc.)
- Follow Go best practices: interfaces, error handling, context usage
- Use goroutines with proper synchronization

## Non-Goals (v1)

- No write operations (commit/push/merge)
- No network credentials management  
- No special submodule handling beyond status
- No PR integration or advanced Git workflows

## Testing Strategy

- Unit tests for core logic (grouping, config round-trip, status parsing)
- Integration tests with temporary Git repos created via exec.Command
- Performance testing to ensure UI stays responsive with many repos
- Event bus testing for proper message flow
- Before making substantial code changes refer to the guidelines and philosophy of @STRATEGY.md and @DEVELOPMENT.md to ensure we do it right!

## Go Implementation Structure

```
cmd/gitagrip/         - Application entry point
internal/
  ├── config/         - Configuration management
  ├── domain/         - Domain models and events
  ├── discovery/      - Repository discovery service
  ├── eventbus/       - Event-driven communication
  ├── git/            - Git operations service
  ├── groups/         - Group management service
  └── ui/             - Bubble Tea UI implementation
```