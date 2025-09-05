use anyhow::Result;
use tempfile::TempDir;
use std::fs;
use std::path::PathBuf;

// Common test utilities for M4 Phase 3 tests
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

// GUIDING STAR TEST: Complete Group Management Workflow
// This test defines the complete user experience for organizing repositories into custom groups
#[test]
fn test_complete_group_management_workflow() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories in different directories for auto-grouping
    let work_dir = base_path.join("work");
    let personal_dir = base_path.join("personal");
    let scripts_dir = base_path.join("scripts");
    
    fs::create_dir_all(&work_dir)?;
    fs::create_dir_all(&personal_dir)?;
    fs::create_dir_all(&scripts_dir)?;
    
    create_test_git_repo(work_dir.join("project-a"))?;
    create_test_git_repo(work_dir.join("project-b"))?;
    create_test_git_repo(personal_dir.join("dotfiles"))?;
    create_test_git_repo(personal_dir.join("blog"))?;
    create_test_git_repo(scripts_dir.join("automation"))?;
    create_test_git_repo(scripts_dir.join("tools"))?;
    
    // Create config with some existing manual groups
    let mut config = create_test_config(base_path.to_path_buf());
    let mut existing_groups = std::collections::HashMap::new();
    existing_groups.insert("Archive".to_string(), gitagrip::config::GroupConfig {
        repos: vec![],
    });
    config.groups = existing_groups;
    
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories (like the real app does)
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    // Phase 3 Test 1: Enter ORGANIZE mode and verify groups are displayed
    assert_eq!(app.current_mode(), gitagrip::app::AppMode::Normal);
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    // Should be able to see available groups (auto + manual)
    let available_groups = app.get_available_groups();
    assert!(available_groups.contains(&"Auto: work".to_string()), "Should see auto work group");
    assert!(available_groups.contains(&"Auto: personal".to_string()), "Should see auto personal group");
    assert!(available_groups.contains(&"Auto: scripts".to_string()), "Should see auto scripts group");
    assert!(available_groups.contains(&"Archive".to_string()), "Should see existing manual group");
    
    // Phase 3 Test 2: Navigate between groups (not just repositories)
    // Initially in repository navigation mode
    assert_eq!(app.get_navigation_mode(), gitagrip::app::NavigationMode::Repository);
    
    // Switch to group navigation mode with Tab key
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Tab)?;
    assert!(redraw_needed, "Switching navigation mode should trigger redraw");
    assert_eq!(app.get_navigation_mode(), gitagrip::app::NavigationMode::Group);
    
    // Navigate between groups with arrow keys
    let initial_group = app.get_current_target_group();
    app.handle_mode_specific_key(crossterm::event::KeyCode::Down)?;
    let new_group = app.get_current_target_group();
    assert_ne!(initial_group, new_group, "Should navigate to different group");
    
    // Phase 3 Test 3: Create a new custom group
    // Navigate to group navigation mode and create new group
    app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
    let groups_before = app.get_available_groups().len();
    
    // Press 'n' to create new group
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('n'))?;
    assert!(redraw_needed, "Creating new group should trigger redraw");
    
    // Should be in text input mode for group name
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::GroupName);
    
    // Type the new group name
    app.handle_text_input("Critical Projects")?;
    
    // Press Enter to confirm
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Enter)?;
    assert!(redraw_needed, "Confirming group creation should trigger redraw");
    
    // Verify new group was created
    let groups_after = app.get_available_groups();
    assert_eq!(groups_after.len(), groups_before + 1, "Should have one more group");
    assert!(groups_after.contains(&"Critical Projects".to_string()), "Should contain new group");
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::None, "Should exit input mode");
    
    // Phase 3 Test 4: Select repositories and move to custom group
    // Switch back to repository navigation
    app.set_navigation_mode(gitagrip::app::NavigationMode::Repository);
    
    // Select specific work repositories (project-a and project-b)
    // Navigate and select first work repository
    app.navigate_to_repository_by_name("project-a")?;
    app.handle_mode_specific_key(crossterm::event::KeyCode::Char(' '))?; // Select
    assert!(app.is_repository_selected_by_name("project-a"), "project-a should be selected");
    
    // Navigate and select second work repository
    app.navigate_to_repository_by_name("project-b")?;
    app.handle_mode_specific_key(crossterm::event::KeyCode::Char(' '))?; // Select
    assert!(app.is_repository_selected_by_name("project-b"), "project-b should be selected");
    
    // Verify both are marked automatically (Space behavior)
    let marked_repos = app.get_marked_repository_names();
    assert_eq!(marked_repos.len(), 2, "Should have 2 marked repositories");
    assert!(marked_repos.contains(&"project-a".to_string()), "project-a should be marked");
    assert!(marked_repos.contains(&"project-b".to_string()), "project-b should be marked");
    
    // Phase 3 Test 5: Navigate to target group and paste
    // Switch to group navigation and navigate to "Critical Projects"
    app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
    app.navigate_to_group("Critical Projects")?;
    assert_eq!(app.get_current_target_group(), "Critical Projects", "Should be targeting Critical Projects group");
    
    // Press 'p' to paste/move repositories
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('p'))?;
    assert!(redraw_needed, "Pasting should trigger redraw");
    
    // Verify repositories moved to the target group
    let critical_group_repos = app.get_repositories_in_group("Critical Projects");
    assert_eq!(critical_group_repos.len(), 2, "Critical Projects should contain 2 repositories");
    
    let repo_names: Vec<String> = critical_group_repos.iter()
        .map(|repo| repo.name.clone())
        .collect();
    assert!(repo_names.contains(&"project-a".to_string()), "Should contain project-a");
    assert!(repo_names.contains(&"project-b".to_string()), "Should contain project-b");
    
    // Original auto group should no longer contain these repositories
    let work_group_repos = app.get_repositories_in_group("Auto: work");
    assert_eq!(work_group_repos.len(), 0, "Auto: work group should now be empty");
    
    // Selection and marking should be cleared
    assert_eq!(app.get_selected_repository_names().len(), 0, "Selection should be cleared");
    assert_eq!(app.get_marked_repository_names().len(), 0, "Marking should be cleared");
    
    // Phase 3 Test 6: Rename an existing group
    // Navigate to the "Archive" group
    app.navigate_to_group("Archive")?;
    assert_eq!(app.get_current_target_group(), "Archive");
    
    // Press 'r' to rename group
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('r'))?;
    assert!(redraw_needed, "Renaming group should trigger redraw");
    
    // Should be in text input mode with current name
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::GroupName);
    assert_eq!(app.get_current_input_text(), "Archive", "Should show current group name");
    
    // Clear and type new name
    app.clear_input();
    app.handle_text_input("Legacy Projects")?;
    
    // Confirm rename
    app.handle_mode_specific_key(crossterm::event::KeyCode::Enter)?;
    
    // Verify group was renamed
    let groups_after_rename = app.get_available_groups();
    assert!(!groups_after_rename.contains(&"Archive".to_string()), "Old name should be gone");
    assert!(groups_after_rename.contains(&"Legacy Projects".to_string()), "New name should exist");
    assert_eq!(app.get_current_target_group(), "Legacy Projects", "Current target should update");
    
    // Phase 3 Test 7: Delete an empty group
    // Navigate to the empty "Legacy Projects" group
    app.navigate_to_group("Legacy Projects")?;
    let repos_in_target = app.get_repositories_in_group("Legacy Projects");
    assert_eq!(repos_in_target.len(), 0, "Group should be empty before deletion");
    
    // Press 'd' to delete group
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('d'))?;
    assert!(redraw_needed, "Deleting group should trigger redraw");
    
    // Verify group was deleted
    let final_groups = app.get_available_groups();
    assert!(!final_groups.contains(&"Legacy Projects".to_string()), "Deleted group should be gone");
    
    // Should automatically switch to another group
    let current_target_after_delete = app.get_current_target_group();
    assert_ne!(current_target_after_delete, "Legacy Projects", "Should switch to different group");
    
    // Phase 3 Test 8: Cannot delete group with repositories
    // Navigate to "Critical Projects" which has repositories
    app.navigate_to_group("Critical Projects")?;
    let repos_in_critical = app.get_repositories_in_group("Critical Projects");
    assert!(repos_in_critical.len() > 0, "Critical Projects should have repositories");
    
    // Try to delete - should fail
    let groups_before_failed_delete = app.get_available_groups().len();
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('d'))?;
    // Should show error message or do nothing
    let groups_after_failed_delete = app.get_available_groups().len();
    assert_eq!(groups_before_failed_delete, groups_after_failed_delete, 
               "Should not delete group with repositories");
    
    Ok(())
}

// Test group navigation edge cases
#[test]
fn test_group_navigation_edge_cases() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Test navigation with no groups
    app.set_mode(gitagrip::app::AppMode::Organize);
    app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
    
    // Should handle empty group list gracefully
    let available_groups = app.get_available_groups();
    assert_eq!(available_groups.len(), 0, "Should start with no groups");
    
    // Navigation keys should not crash
    let redraw1 = app.handle_mode_specific_key(crossterm::event::KeyCode::Up)?;
    let redraw2 = app.handle_mode_specific_key(crossterm::event::KeyCode::Down)?;
    // Should return false (no redraw needed) or handle gracefully
    
    Ok(())
}

// Test text input handling for group names
#[test]
fn test_group_name_input_validation() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let config = create_test_config(temp_dir.path().to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    app.set_mode(gitagrip::app::AppMode::Organize);
    app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
    
    // Start group creation
    app.handle_mode_specific_key(crossterm::event::KeyCode::Char('n'))?;
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::GroupName);
    
    // Test empty name rejection
    app.handle_mode_specific_key(crossterm::event::KeyCode::Enter)?;
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::GroupName, 
               "Should remain in input mode for empty name");
    
    // Test valid name acceptance
    app.handle_text_input("Valid Group Name")?;
    app.handle_mode_specific_key(crossterm::event::KeyCode::Enter)?;
    assert_eq!(app.get_input_mode(), gitagrip::app::InputMode::None, 
               "Should exit input mode for valid name");
    
    // Verify group was created
    let groups = app.get_available_groups();
    assert!(groups.contains(&"Valid Group Name".to_string()), "Should create group with valid name");
    
    Ok(())
}