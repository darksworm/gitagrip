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

// GUIDING STAR TEST: Simplified File Manager-Like Organization
// This test defines the much simpler user workflow we want to achieve
#[test]
fn test_simplified_organize_workflow() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories in different directories 
    let work_dir = base_path.join("work");
    let personal_dir = base_path.join("personal");
    
    fs::create_dir_all(&work_dir)?;
    fs::create_dir_all(&personal_dir)?;
    
    create_test_git_repo(work_dir.join("frontend"))?;
    create_test_git_repo(work_dir.join("backend"))?;
    create_test_git_repo(work_dir.join("mobile"))?;
    create_test_git_repo(personal_dir.join("dotfiles"))?;
    create_test_git_repo(personal_dir.join("blog"))?;
    
    let mut config = create_test_config(base_path.to_path_buf());
    
    // Add a manual "Legacy" group
    config.groups.insert("Legacy".to_string(), gitagrip::config::GroupConfig {
        repos: vec![],
    });
    
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories (exactly like app.run does)
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("=== SIMPLIFIED ORGANIZE WORKFLOW TEST ===");
    
    // Test 1: Start in NORMAL mode, switch to single ORGANIZE mode
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal, "Should start in NORMAL mode");
    
    app.toggle_mode();
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Organize, "Should switch to ORGANIZE mode");
    
    // Test 2: Single navigation - arrow keys navigate through EVERYTHING
    // (repositories, group headers, everything in one unified list)
    let initial_cursor = app.get_cursor_position();
    println!("Initial cursor position: {}", initial_cursor);
    
    // Navigate down through the interface
    app.handle_organize_key(crossterm::event::KeyCode::Down)?;
    let cursor_after_down = app.get_cursor_position();
    assert_ne!(initial_cursor, cursor_after_down, "Cursor should move when pressing down");
    
    // Test 3: Select multiple repositories with Space
    // Navigate to first work repository and select it
    app.navigate_to_item_containing("frontend")?;
    let frontend_cursor = app.get_cursor_position();
    println!("Frontend repository cursor: {}", frontend_cursor);
    
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
    assert!(app.is_item_selected(frontend_cursor), "Frontend should be selected");
    
    // Navigate to second work repository and select it  
    app.navigate_to_item_containing("backend")?;
    let backend_cursor = app.get_cursor_position();
    
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
    assert!(app.is_item_selected(backend_cursor), "Backend should be selected");
    assert!(app.is_item_selected(frontend_cursor), "Frontend should still be selected");
    
    let selected_repos = app.get_selected_repository_names();
    assert_eq!(selected_repos.len(), 2, "Should have 2 selected repositories");
    assert!(selected_repos.contains(&"frontend".to_string()), "Should contain frontend");
    assert!(selected_repos.contains(&"backend".to_string()), "Should contain backend");
    
    println!("Selected repositories: {:?}", selected_repos);
    
    // Test 4: Create new group from selected repositories (n key)
    app.handle_organize_key(crossterm::event::KeyCode::Char('n'))?;
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::GroupName, "Should be in group name input mode");
    
    // Type new group name
    app.handle_text_input("Production")?;
    assert_eq!(app.get_current_input_text(), "Production", "Should show typed text");
    
    // Confirm group creation
    app.handle_organize_key(crossterm::event::KeyCode::Enter)?;
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::None, "Should exit input mode");
    
    // Verify new group was created with selected repositories
    let production_repos = app.get_repositories_in_group("Production");
    assert_eq!(production_repos.len(), 2, "Production group should have 2 repositories");
    
    let production_names: Vec<String> = production_repos.iter().map(|r| r.name.clone()).collect();
    assert!(production_names.contains(&"frontend".to_string()), "Should contain frontend");
    assert!(production_names.contains(&"backend".to_string()), "Should contain backend");
    
    // Selection should be cleared after group creation
    assert_eq!(app.get_selected_repository_names().len(), 0, "Selection should be cleared");
    
    // Test 5: Cut repositories from groups (x key)
    // Navigate to a repository in the Production group
    app.navigate_to_item_containing("frontend")?;
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?; // Select it
    
    // Cut it (removes from Production group, goes back to its auto group)
    app.handle_organize_key(crossterm::event::KeyCode::Char('x'))?;
    
    // Verify repository moved back to its auto group (Auto: work)
    let production_after_cut = app.get_repositories_in_group("Production");
    assert_eq!(production_after_cut.len(), 1, "Production should have 1 repo after cut");
    
    let auto_work_repos = app.get_repositories_in_group("Auto: work");
    let auto_work_names: Vec<String> = auto_work_repos.iter().map(|r| r.name.clone()).collect();
    assert!(auto_work_names.contains(&"frontend".to_string()), "Frontend should be in Auto: work after cut");
    
    println!("After cut - Production: {}, Auto: work contains frontend: {}", 
             production_after_cut.len(), 
             auto_work_names.contains(&"frontend".to_string()));
    
    // Test 6: Move repositories between groups (m key)
    // Select a repository from Auto: work group
    app.navigate_to_item_containing("frontend")?;
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
    
    // Navigate cursor to Legacy group (not select, just position cursor there)
    app.navigate_to_group_header("Legacy")?;
    
    // Move selected repositories to the group where cursor is positioned
    app.handle_organize_key(crossterm::event::KeyCode::Char('m'))?;
    
    // Verify move
    let legacy_repos = app.get_repositories_in_group("Legacy");
    assert_eq!(legacy_repos.len(), 1, "Legacy should have 1 repo after move");
    assert_eq!(legacy_repos[0].name, "frontend", "Should be frontend in Legacy");
    
    // Test 7: Delete empty groups (d key)
    // Production should now be empty (backend was the only one left, let's move it too)
    app.navigate_to_item_containing("backend")?;
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
    app.navigate_to_group_header("Legacy")?;
    app.handle_organize_key(crossterm::event::KeyCode::Char('m'))?;
    
    // Now Production should be empty
    let production_empty = app.get_repositories_in_group("Production");
    assert_eq!(production_empty.len(), 0, "Production should be empty");
    
    // Navigate to Production group and delete it
    app.navigate_to_group_header("Production")?;
    app.handle_organize_key(crossterm::event::KeyCode::Char('d'))?;
    
    // Verify group was deleted
    let available_groups = app.get_available_groups();
    assert!(!available_groups.contains(&"Production".to_string()), "Production group should be deleted");
    
    println!("Available groups after deletion: {:?}", available_groups);
    
    // Test 8: Try to delete non-empty group (should fail)
    app.navigate_to_group_header("Legacy")?;
    let groups_before = app.get_available_groups().len();
    app.handle_organize_key(crossterm::event::KeyCode::Char('d'))?;
    let groups_after = app.get_available_groups().len();
    assert_eq!(groups_before, groups_after, "Should not delete non-empty group");
    
    println!("✅ Simplified organize workflow test complete!");
    Ok(())
}

// Test edge cases for the simplified navigation
#[test]
fn test_simplified_navigation_edge_cases() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    // Test navigation with no repositories
    let initial_cursor = app.get_cursor_position();
    app.handle_organize_key(crossterm::event::KeyCode::Down)?;
    // Should handle gracefully, not crash
    
    // Test selection with empty list
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
    assert_eq!(app.get_selected_repository_names().len(), 0, "No repositories to select");
    
    Ok(())
}

// Test that the simplified interface shows clear visual feedback
#[test]
fn test_simplified_ui_feedback() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    create_test_git_repo(base_path.join("test-repo"))?;
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    // Test that UI shows current mode
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Organize);
    
    // Test that selection is tracked
    app.navigate_to_item_containing("test-repo")?;
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
    assert_eq!(app.get_selected_repository_names().len(), 1);
    
    Ok(())
}