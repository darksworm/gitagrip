use anyhow::Result;
use tempfile::TempDir;
use std::fs;
use std::path::PathBuf;

// Common test utilities
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

// This is our "guiding star" integration test for scrollable UI
// It tests the complete flow: many repositories -> scrollable display -> navigation
#[test]
fn test_scrollable_repository_list_integration() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create many repositories to exceed screen height (25 repos)
    let _repo_names = create_test_repos(base_path, 25)?;
    
    // Create config for the test
    let config = create_test_config(base_path.to_path_buf());
    
    // Test 1: App should handle many repositories without crashing
    let mut app = gitagrip::app::App::new(config.clone(), None);
    
    // Discover all repositories (like the real app does)
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    assert_eq!(discovered_repos.len(), 25, "Should discover all test repositories");
    
    // Add repositories to app (simulating background scan completion)
    for repo in discovered_repos {
        app.add_repository(repo);
    }
    app.scan_complete = true;
    
    // Test 2: App should support scrolling state
    assert_eq!(app.repositories.len(), 25, "App should hold all repositories");
    
    // Test 3: App should have scroll state
    assert_eq!(app.scroll_offset, 0, "App should have scroll_offset field starting at 0");
    
    // Test 4: App should support scroll operations
    app.scroll_down();
    assert_eq!(app.scroll_offset, 1, "scroll_down should increment scroll_offset");
    
    app.scroll_up();
    assert_eq!(app.scroll_offset, 0, "scroll_up should decrement scroll_offset");
    
    // Test 5: UI should be able to render a windowed view
    let mock_terminal_height = 20usize;
    let mock_content_height = mock_terminal_height - 6; // Minus title, footer, borders
    let visible_repo_count = mock_content_height.saturating_sub(2); // Minus group headers
    
    assert!(app.repositories.len() > visible_repo_count, 
            "Should have more repos than can fit in viewport");
    
    // Test 6: Scroll functionality should be functional
    app.scroll_down();
    assert_eq!(app.scroll_offset, 1, "Should be able to scroll down");
    
    app.scroll_up();
    assert_eq!(app.scroll_offset, 0, "Should be able to scroll back up");
    
    Ok(())
}

#[test]
fn test_scroll_bounds_checking() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Test scrolling with no repositories
    assert_eq!(app.scroll_offset, 0);
    app.scroll_up(); // Should not crash or go negative
    assert_eq!(app.scroll_offset, 0, "Should stay at 0 when scrolling up from empty");
    
    // Add some repositories
    let discovered_repos = create_test_repos(temp_dir.path(), 3)?;
    for name in &discovered_repos {
        app.add_repository(gitagrip::scan::Repository {
            name: name.clone(),
            path: temp_dir.path().join(name),
            auto_group: "Test".to_string(),
        });
    }
    
    // Test scroll bounds
    assert_eq!(app.repositories.len(), 3);
    
    // Scroll to end
    app.scroll_down(); // offset = 1
    app.scroll_down(); // offset = 2
    app.scroll_down(); // Should not go beyond repo count - 1
    assert_eq!(app.scroll_offset, 2, "Should not scroll beyond last repository");
    
    // Scroll back to beginning
    app.scroll_up(); // offset = 1  
    app.scroll_up(); // offset = 0
    app.scroll_up(); // Should not go below 0
    assert_eq!(app.scroll_offset, 0, "Should not scroll below 0");
    
    Ok(())
}

#[test]
fn test_scroll_with_empty_repository_list() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // No repositories added
    assert_eq!(app.repositories.len(), 0);
    assert_eq!(app.scroll_offset, 0);
    
    // Should handle scrolling gracefully
    app.scroll_down();
    assert_eq!(app.scroll_offset, 0, "Should not scroll with empty repository list");
    
    app.scroll_up();
    assert_eq!(app.scroll_offset, 0, "Should not scroll with empty repository list");
    
    Ok(())
}