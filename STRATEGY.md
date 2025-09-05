# GitaGrip Development Strategy

## Overview

GitaGrip is built using **Outside-In Test-Driven Development (TDD)** with a focus on user experience, quality, and maintainability. This document captures our core development philosophy and practices that have proven successful.

## Core Principles

### 1. Outside-In TDD with "Guiding Star" Integration Tests

**Philosophy**: Start with the complete user experience, then work inward.

- **Write comprehensive integration tests first** that describe the full user workflow
- These "guiding star" tests initially fail, then drive all implementation decisions
- Tests serve as living documentation of expected behavior
- Example: `test_m3_end_to_end_git_status_in_tui()` drove our entire M3 implementation

### 2. Run the Real App in Tests

**Critical Pattern**: Our integration tests don't mock - they run the actual application.

```rust
// ✅ Good: Run the real app
let mut app = gitagrip::app::App::new(config.clone());
let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();

// Process repository scan events (exactly like app.run does)
while let Ok(event) = scan_receiver.try_recv() {
    match event {
        gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
            app.repositories.push(repo);  // Same as main app
        }
        // ...
    }
}
```

**Why this matters**:
- Catches integration bugs that unit tests miss
- Ensures tests match real application behavior
- Found critical issues like competing channel receivers
- Uses real git repositories and file systems

### 3. Milestone-Based Development

Each milestone delivers **complete, usable value** to users:

- **M0: Bootstrap** ✅ - Basic TUI structure and test framework
- **M1: Config + CLI** ✅ - Configuration loading and command-line interface  
- **M2: Repository Discovery** ✅ - Find and group git repositories
- **M3: Git Status Integration** ✅ - Display git status with colored branches
- **M4: Interactive Navigation** (proposed) - User can navigate and perform operations

**Benefits**:
- Clear progress tracking
- Early user feedback opportunities
- Maintains motivation with regular deliverables
- Easy rollback if needed

### 4. User Experience First

**Performance and UX are non-negotiable**:
- Background scanning keeps UI responsive
- Immediate fix for UI jittering (stable BTreeMap ordering)
- Rich text display with colored branch names
- Clean, informative status indicators (●/✓/?/⋯)

### 5. Quality Gates

**All tests must pass before moving forward**:
- Comprehensive test coverage (53 tests as of M3)
- Integration tests catch real-world issues
- Performance regression testing
- Manual verification of user-facing features

## Testing Strategy

### Integration Test Structure

```rust
#[test]
fn test_milestone_integration() -> Result<()> {
    // 1. Set up real environment (temp dirs, git repos)
    let temp_dir = TempDir::new()?;
    create_real_git_repos(&temp_dir)?;
    
    // 2. Create real app with real config
    let app = gitagrip::app::App::new(config);
    
    // 3. Run real background processes
    std::thread::spawn(|| {
        gitagrip::scan::scan_repositories_background(base_dir, sender);
    });
    
    // 4. Process events exactly like main app does
    // (Copy event handling logic from main.rs)
    
    // 5. Assert on user-visible outcomes
    assert!(ui_content.contains("expected behavior"));
    
    Ok(())
}
```

### Test Organization

- **Integration tests** verify complete user workflows
- **Unit tests** verify individual component behavior  
- **Comments like "exactly like app.run does"** ensure test fidelity
- Test real git repositories, not mocked ones

## Architecture Principles

### 1. Event-Driven Design

- Use channels for communication between components
- Background threads for non-blocking operations
- Clear event types for different concerns (`ScanEvent`, `StatusEvent`)

### 2. Separation of Concerns

- `scan` module: Repository discovery
- `git` module: Git operations
- `app` module: Application state and UI
- `config` module: Configuration management

### 3. Error Handling

- Use `anyhow::Result` for comprehensive error context
- Graceful degradation when possible
- Clear error messages for user-facing issues

## Development Workflow

1. **Start with integration test** - Write the "guiding star" test for the milestone
2. **Run test** - It should fail initially
3. **Implement minimal solution** - Make the test pass
4. **Add unit tests** - Cover edge cases and internal logic
5. **Refactor** - Improve code quality while keeping tests green
6. **Manual testing** - Verify real-world usage
7. **Clean up** - Remove dead code, update documentation
8. **Commit** - Clear, descriptive commit messages

## Success Metrics

- All tests pass (currently 53/53)
- Real-world performance meets expectations
- Clean, maintainable codebase
- User-facing features work reliably
- Documentation stays current

## Key Lessons

1. **Integration tests that mirror real app behavior catch critical bugs**
2. **Background processing is essential for responsive UI**
3. **User experience issues should be fixed immediately**
4. **Milestone-based development maintains focus and momentum**
5. **Quality gates prevent technical debt accumulation**

---

This strategy has successfully delivered GitaGrip from concept to working TUI in a structured, maintainable way. Continue following these principles for future development.