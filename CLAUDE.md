# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

YARG is a fast Rust + Ratatui TUI application for discovering, grouping, and inspecting multiple Git repositories (read-only). This is currently a planning repository with detailed specifications but no implementation yet.

## Project Status

This repository contains planning documents but no actual Rust code yet. The project is designed to be implemented in phases following the milestone structure outlined in `plan.md`.

## Tech Stack & Dependencies

When implementing YARG, use these crates:
- **TUI**: ratatui, crossterm
- **Git**: git2 (libgit2)  
- **FS scan**: walkdir (and ignore optional)
- **Concurrency**: crossbeam-channel (or std mpsc) + rayon (optional)
- **Config**: serde, toml, directories (XDG paths), serde_with
- **Errors/logging**: anyhow, thiserror, tracing, tracing-subscriber

## Architecture

The application follows a modular structure:

### Core Modules
- **Config Module**: TOML configuration handling with serde (reads from `~/.config/yarg/yarg.toml`)
- **Discovery Module**: Filesystem scanning for Git repositories using walkdir
- **Git Operations Module**: Git status, fetch, and log operations via git2
- **UI Module**: Ratatui-based terminal UI with keyboard navigation
- **App State & Event Loop**: Central state management with background worker threads

### Configuration Schema
```toml
version = 1
base_dir = "/path/to/repos"

[ui]
show_ahead_behind = true
autosave_on_exit = true

[groups.Work]
repos = ["/path/to/work-repo1", "/path/to/work-repo2"]

[groups.Personal] 
repos = ["/path/to/personal-repo"]
```

## Key Features (Planned)

1. **Recursive repository discovery** - Auto-detect Git repos in base directory
2. **Repository status overview** - Show branch, dirty/clean state, ahead/behind counts
3. **Read-only Git operations** - Fetch updates, view commit logs
4. **Repository grouping** - Manual and automatic grouping by directory structure
5. **TOML configuration persistence** - Save repo lists and custom groups
6. **Non-blocking TUI** - Background operations with event-driven updates

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
- **Non-blocking UI**: All I/O operations run on background threads
- **Event-driven architecture**: Worker threads communicate via channels
- **Clean code**: Small focused modules, strong typing
- **Cross-platform**: Terminal-based for portability
- **Performance**: Rust + concurrency for handling many repos efficiently

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
- Add tracing spans for debugging slow operations
- Handle errors gracefully (network failures, missing repos, etc.)
- Follow Rust best practices: strong typing, Result propagation, immutable data where possible

## Non-Goals (v1)

- No write operations (commit/push/merge)
- No network credentials management  
- No special submodule handling beyond status
- No PR integration or advanced Git workflows

## Testing Strategy

- Unit tests for core logic (grouping, config round-trip, status parsing)
- Integration tests with temporary Git repos created via git2
- Performance testing to ensure UI stays responsive with many repos
- before making substantial code changes refer to the guidelines and philosophy of @STRATEGY.md and @DEVELOPMENT.md to ensure we do it right!