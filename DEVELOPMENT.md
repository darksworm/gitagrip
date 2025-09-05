# GitaGrip Development Guide

## Overview

This document captures practical lessons learned from building GitaGrip, including mistakes we made and how to avoid them in future development. Use this alongside `STRATEGY.md` for a complete development approach.

## Environment Setup

### Prerequisites

```bash
# Ensure Rust/Cargo is properly configured
source ~/.cargo/env  # Add to your shell profile to avoid repetition

# Verify installation
cargo --version
rustc --version
```

### Build and Test

```bash
# Development build
cargo build

# Run all tests
cargo test

# Run specific test
cargo test test_name

# Check for warnings/dead code
cargo check

# Release build
cargo build --release
```

## Lessons Learned

### 1. Dead Code Accumulation ⚠️

**Problem**: We accumulated unused functions and structs over time.

Current dead code (as of M3):
- `compute_statuses_parallel()` function (git.rs:130)
- `RepositoryDisplayInfo` struct (app.rs:7)  
- `prepare_repository_display_with_status()` method
- `render_repository_list_with_status()` method
- `create_test_ui_frame()` method

**Lesson**: Clean up dead code immediately after refactoring, don't let it accumulate.

**Fix**: 
```bash
# Regular cleanup
cargo check --quiet  # Shows dead code warnings
# Remove unused code when warnings appear
```

### 2. UI Architecture Evolution 🔄

**Problem**: The colored branch feature required rewriting 70+ lines of UI code because we didn't design for extensibility.

**Evolution**:
1. **M2**: Plain text UI rendering
2. **M3**: Complete rewrite to support rich text (Spans, Lines)

**Lesson**: Design UI architecture for extensibility early, especially display/rendering logic.

**Better approach**:
```rust
// ✅ Good: Separate display logic from app logic
trait Renderer {
    fn render_repository(&self, repo: &Repository, status: &RepoStatus) -> Line;
}

struct PlainTextRenderer;
struct RichTextRenderer;  // Can add colors, styles, etc.
```

### 3. Naming and Branding Changes 📝

**Problem**: Renamed entire tool late (YARG → GitaGrip), requiring changes to 72+ references across multiple files.

**Files affected**:
- Cargo.toml (package name, binary name)
- src/app.rs (UI title)
- src/config.rs (config paths)
- src/main.rs (log messages)
- tests/integration_test.rs (all 72 references)

**Lesson**: Establish naming/branding early, or make it configurable from the start.

**Better approach**:
```rust
// ✅ Good: Centralized branding
const APP_NAME: &str = "GitaGrip";
const CONFIG_DIR: &str = "gitagrip";

// Or make it configurable
struct Branding {
    name: &'static str,
    config_dir: &'static str,
    // ...
}
```

### 4. Test File Organization 📚

**Problem**: Single integration test file grew to 1200+ lines with 15 tests.

**Current structure**:
```
tests/integration_test.rs  (1200+ lines)
├── M1 tests (config, CLI)
├── M2 tests (repository discovery)  
├── M3 tests (git status)
└── Edge case tests
```

**Lesson**: Organize tests by milestone/feature for better maintainability.

**Better structure**:
```
tests/
├── m1_config_cli_tests.rs
├── m2_repository_discovery_tests.rs  
├── m3_git_status_tests.rs
└── integration/
    ├── mod.rs  (common test utilities)
    └── end_to_end_tests.rs
```

### 5. Performance Regression Catching 🐛

**Success Story**: Our integration tests caught a critical performance regression.

**Issue**: App hung on "scanning for repositories..." due to competing channel receivers.

**Why our approach worked**:
- Integration test `test_scanning_completes_with_real_repos()` ran the real app
- Test mirrored exact behavior of main.rs  
- Caught issue that unit tests would have missed

**Key pattern**:
```rust
// Comments ensure test fidelity
// Process repository scan events (exactly like app.run does)
while let Ok(event) = scan_receiver.try_recv() {
    match event {
        // Mirror the exact same logic as main.rs
    }
}
```

### 6. Refactoring Strategy 🔧

**Problem**: Built multiple approaches instead of designing for change.

**Example**: Repository display logic
1. **V1**: Simple HashMap with string formatting
2. **V2**: RepositoryDisplayInfo struct (now unused)  
3. **V3**: Rich text with Spans and Lines

**Lesson**: When you find yourself rewriting large chunks, step back and design an extensible architecture.

**Better approach**:
- Start with simple implementation
- When adding second approach, extract abstraction
- Design for the third implementation

## Development Best Practices

### 1. Immediate Cleanup

```bash
# After any refactoring
cargo check --quiet    # Check for dead code
git status             # See what changed
# Clean up unused code before committing
```

### 2. Test Fidelity

```rust
// ✅ Good: Clear comments about mirroring real behavior
// Process events exactly like app.run does
while let Ok(event) = receiver.try_recv() {
    // Copy exact logic from main.rs
}

// ❌ Bad: Test logic that doesn't match app
let mock_events = vec![...];  // Doesn't match real behavior
```

### 3. Architecture Evolution

1. **First implementation**: Simple, working solution
2. **Second implementation**: Extract abstraction when adding variation  
3. **Third implementation**: Should fit cleanly into abstraction

### 4. Quality Gates

Before every commit:
- [ ] All tests pass (`cargo test`)
- [ ] No dead code warnings (`cargo check`)
- [ ] Manual testing of user-facing features
- [ ] Code review of changes

## Common Patterns

### Setting Up Integration Tests

```rust
use tempfile::TempDir;
use std::fs;

#[test]  
fn test_feature_integration() -> Result<()> {
    // 1. Create real test environment
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // 2. Create real git repositories
    create_test_git_repo(base_path.join("repo1"))?;
    
    // 3. Create real config
    let config = gitagrip::config::Config {
        base_dir: base_path.to_path_buf(),
        // ...
    };
    
    // 4. Run real app components
    let mut app = gitagrip::app::App::new(config);
    
    // 5. Test complete workflow
    // ...
    
    Ok(())
}
```

### Creating Test Git Repositories

```rust
fn create_test_git_repo(path: PathBuf) -> Result<()> {
    fs::create_dir_all(&path)?;
    
    let git_repo = git2::Repository::init(&path)?;
    let signature = git2::Signature::now("Test User", "test@example.com")?;
    
    // Create initial commit
    let tree_id = {
        let mut index = git_repo.index()?;
        let tree_id = index.write_tree()?;
        tree_id
    };
    let tree = git_repo.find_tree(tree_id)?;
    git_repo.commit(
        Some("HEAD"),
        &signature,
        &signature,
        "Initial commit",
        &tree,
        &[],
    )?;
    
    Ok(())
}
```

## Future Improvements

1. **Clean up current dead code** (see section 1)
2. **Split integration tests** by milestone
3. **Add development environment documentation**
4. **Consider extracting UI rendering abstraction**
5. **Centralize branding/naming configuration**

## Conclusion

These lessons learned from building GitaGrip should guide future development. The key is to learn from each iteration and apply improvements immediately, not accumulate technical debt.

Remember: **Perfect is the enemy of good, but good enough should still be maintainable.**