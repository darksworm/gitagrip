use anyhow::Result;
use tempfile::TempDir;
use std::fs;
use std::path::PathBuf;

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

fn create_config_with_groups(base_dir: PathBuf) -> gitagrip::config::Config {
    let mut groups = std::collections::HashMap::new();
    
    // Create some initial groups
    groups.insert("Important".to_string(), gitagrip::config::GroupConfig {
        repos: vec![base_dir.join("critical-app")],
    });
    
    groups.insert("Archive".to_string(), gitagrip::config::GroupConfig {
        repos: vec![],  // Empty group to test moving into
    });
    
    gitagrip::config::Config {
        version: 1,
        base_dir,
        ui: gitagrip::config::UiConfig {
            show_ahead_behind: true,
            autosave_on_exit: false,
        },
        groups,
    }
}

#[test]
fn test_move_functionality_complete_workflow() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    println!("=== TESTING MOVE FUNCTIONALITY ===");
    
    // Create test repositories
    create_test_git_repo(base_path.join("critical-app"))?;      // Will be in Important group
    create_test_git_repo(base_path.join("old-project"))?;      // Will be in Ungrouped
    create_test_git_repo(base_path.join("legacy-tool"))?;      // Will be in Ungrouped
    
    let config = create_config_with_groups(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("\nInitial state:");
    let available_groups = app.get_available_groups();
    for group_name in &available_groups {
        let repos = app.get_repositories_in_group(group_name);
        println!("  {}: {} repos", group_name, repos.len());
        for repo in repos {
            println!("    - {}", repo.name);
        }
    }
    
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    println!("\n=== STEP 1: Select repositories to move ===");
    
    // Find and select repositories in the Ungrouped section
    // Display layout:
    // Important:
    //   [0] critical-app
    // Ungrouped:
    //   [1] legacy-tool  
    //   [2] old-project
    
    // Select legacy-tool (display position 1)
    app.current_selection = 1;
    let storage_index_1 = app.display_to_storage_index(1);
    app.toggle_repository_selection(storage_index_1);
    
    // Select old-project (display position 2)  
    app.current_selection = 2;
    let storage_index_2 = app.display_to_storage_index(2);
    app.toggle_repository_selection(storage_index_2);
    
    println!("Selected {} repositories", app.selected_repositories.len());
    for &storage_idx in &app.selected_repositories {
        if let Some(repo) = app.repositories.get(storage_idx) {
            println!("  - {} (Storage[{}])", repo.name, storage_idx);
        }
    }
    
    println!("\n=== STEP 2: Navigate to target group ===");
    
    // Navigate to Archive group (position 0, since Important comes first alphabetically)
    // But first, let's see what the display layout actually looks like:
    let mut display_pos = 0;
    for group_name in &available_groups {
        println!("  Group header '{}' at display position {}", group_name, display_pos);
        display_pos += 1; // Group header
        
        let repos = app.get_repositories_in_group(group_name);
        for repo in repos {
            println!("    Display[{}]: {}", display_pos, repo.name);  
            display_pos += 1;
        }
    }
    
    // Navigate to Archive group header (position 0) to move our selection there
    // Now that we show empty groups in organize mode, Archive header is at position 0
    app.current_selection = 0;
    let target_group = app.get_group_at_display_position(app.current_selection);
    println!("Target group for move: {:?}", target_group);
    
    println!("\n=== STEP 3: Execute move ===");
    
    let move_result = app.move_selected_repositories()?;
    println!("Move operation result: {}", move_result);
    
    println!("\nFinal state:");
    let updated_groups = app.get_available_groups();
    for group_name in &updated_groups {
        let repos = app.get_repositories_in_group(group_name);
        println!("  {}: {} repos", group_name, repos.len());
        for repo in repos {
            println!("    - {}", repo.name);
        }
    }
    
    println!("\nSelected repositories after move: {}", app.selected_repositories.len());
    
    // Verify the move worked - repositories should now be in Archive
    let archive_repos = app.get_repositories_in_group("Archive");
    let important_repos = app.get_repositories_in_group("Important");
    let ungrouped_repos = app.get_repositories_in_group("Ungrouped");
    
    assert_eq!(archive_repos.len(), 2, "Archive should now have 2 repositories");
    assert_eq!(important_repos.len(), 1, "Important should still have 1 repository");
    assert_eq!(ungrouped_repos.len(), 0, "Ungrouped should now be empty");
    assert_eq!(app.selected_repositories.len(), 0, "Selection should be cleared after move");
    
    println!("\n✅ Move functionality test completed successfully!");
    Ok(())
}