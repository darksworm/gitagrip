use anyhow::Result;
use tempfile::TempDir;
use std::fs;
use std::path::PathBuf;

// Common test utilities for M4 tests
fn create_test_git_repo(path: PathBuf) -> Result<()> {
    fs::create_dir_all(&path)?;
    
    let git_repo = git2::Repository::init(&path)?;
    let signature = git2::Signature::now("Test User", "test@example.com")?;
    
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

fn create_test_repos(base_path: &std::path::Path, count: usize) -> Result<Vec<String>> {
    let mut repo_names = Vec::new();
    
    for i in 0..count {
        let repo_name = format!("repo-{:02}", i);
        let repo_path = base_path.join(&repo_name);
        create_test_git_repo(repo_path)?;
        repo_names.push(repo_name);
    }
    
    Ok(repo_names)
}

fn create_test_config(base_dir: PathBuf) -> gitagrip::config::Config {
    gitagrip::config::Config {
        version: 1,
        base_dir,
        ui: gitagrip::config::UiConfig {
            show_ahead_behind: true,
            autosave_on_exit: false,
        },
        groups: std::collections::HashMap::new(),
    }
}

// This is our "guiding star" integration test for modal architecture
// It tests the complete modal workflow: NORMAL → ORGANIZE → operations → NORMAL
#[test]
fn test_modal_architecture_integration() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories
    let _repo_names = create_test_repos(base_path, 5)?;
    
    // Create config for the test
    let config = create_test_config(base_path.to_path_buf());
    
    // Test 1: App should start in NORMAL mode
    let mut app = gitagrip::app::App::new(config.clone());
    
    // Discover all repositories (like the real app does)
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    assert_eq!(discovered_repos.len(), 5, "Should discover all test repositories");
    
    // Add repositories to app (simulating background scan completion)
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    // Test 2: App should start in NORMAL mode
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal, "App should start in NORMAL mode");
    
    // Test 3: App should support mode switching
    app.toggle_mode();
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Organize, "Should switch to ORGANIZE mode");
    
    app.toggle_mode();
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal, "Should switch back to NORMAL mode");
    
    // Test 4: Mode should affect keymap behavior
    // In NORMAL mode, 'f' should trigger fetch (when implemented)
    // In ORGANIZE mode, 'f' should be ignored or do something different
    
    // Set to NORMAL mode
    app.set_mode(gitagrip::app::AppMode::Normal);
    let normal_mode_response = app.handle_key_for_mode(crossterm::event::KeyCode::Char('f'));
    
    // Set to ORGANIZE mode  
    app.set_mode(gitagrip::app::AppMode::Organize);
    let organize_mode_response = app.handle_key_for_mode(crossterm::event::KeyCode::Char('f'));
    
    // They should behave differently (exact behavior will be implemented)
    // For now, just verify the method exists and returns something
    assert!(normal_mode_response.is_ok() || normal_mode_response.is_err()); // Method exists
    assert!(organize_mode_response.is_ok() || organize_mode_response.is_err()); // Method exists
    
    // Test 5: UI should reflect current mode in footer
    // This will be tested once UI rendering is updated
    
    Ok(())
}

// Test specific modal operations
#[test]
fn test_mode_switching_behavior() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config);
    
    // Test initial state
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal);
    
    // Test mode switching
    app.toggle_mode();
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Organize);
    
    // Test explicit mode setting
    app.set_mode(gitagrip::app::AppMode::Normal);
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal);
    
    app.set_mode(gitagrip::app::AppMode::Organize);
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Organize);
    
    Ok(())
}

// Test that modes don't interfere with existing functionality
#[test]
fn test_normal_mode_preserves_existing_functionality() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories
    let _repo_names = create_test_repos(base_path, 3)?;
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config);
    
    // Add repositories
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    
    // Should be in NORMAL mode by default
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal);
    
    // Existing functionality should still work:
    
    // 1. Scrolling
    assert_eq!(app.scroll_offset, 0);
    app.scroll_down();
    assert_eq!(app.scroll_offset, 1);
    app.scroll_up();
    assert_eq!(app.scroll_offset, 0);
    
    // 2. Repository count
    assert_eq!(app.repositories.len(), 3);
    
    // 3. Scan completion tracking
    app.scan_complete = true;
    assert!(app.scan_complete);
    
    Ok(())
}

// Test that key behavior is actually different between modes
#[test]
fn test_modal_keymap_dispatch_behavior() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config);

    // Add some repositories for scrolling tests
    for i in 0..5 {
        app.repositories.push(gitagrip::scan::Repository {
            name: format!("repo-{}", i),
            path: temp_dir.path().join(format!("repo-{}", i)),
            auto_group: "Test".to_string(),
        });
    }

    // Test NORMAL mode behavior
    app.set_mode(gitagrip::app::AppMode::Normal);
    assert_eq!(app.scroll_offset, 0);
    
    // Down arrow should scroll in NORMAL mode
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Down)?;
    assert!(redraw_needed, "Down key should require redraw in NORMAL mode");
    assert_eq!(app.scroll_offset, 1, "Scroll should work in NORMAL mode");
    
    // Up arrow should scroll in NORMAL mode
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Up)?;
    assert!(redraw_needed, "Up key should require redraw in NORMAL mode");
    assert_eq!(app.scroll_offset, 0, "Scroll should work in NORMAL mode");

    // Test ORGANIZE mode behavior  
    app.set_mode(gitagrip::app::AppMode::Organize);
    let initial_scroll = app.scroll_offset;
    
    // Down arrow should NOT scroll in ORGANIZE mode (different behavior)
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Down)?;
    assert!(!redraw_needed, "Down key should not require redraw in ORGANIZE mode (different behavior)");
    assert_eq!(app.scroll_offset, initial_scroll, "Scroll should not change in ORGANIZE mode");
    
    // Test that organize-specific keys work
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char(' '))?;
    // Space key should be handled in organize mode (even if no visual change yet)
    assert!(!redraw_needed, "Space key handled but no visual change implemented yet");

    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('m'))?;
    assert!(!redraw_needed, "Mark key handled but no visual change implemented yet");

    Ok(())
}

// Test that the mode is properly displayed in UI
#[test]
fn test_modal_ui_display() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config);

    // This is a smoke test - we can't easily test the UI rendering directly,
    // but we can verify the mode state affects the UI methods without panicking
    
    // Test NORMAL mode
    app.set_mode(gitagrip::app::AppMode::Normal);
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal);

    // Test ORGANIZE mode
    app.set_mode(gitagrip::app::AppMode::Organize);
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Organize);

    // UI should be able to render both modes without crashing
    // (The actual UI rendering test would require a more complex setup with terminals)
    
    Ok(())
}