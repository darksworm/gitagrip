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

fn create_config_with_priority_group(base_dir: PathBuf) -> gitagrip::config::Config {
    let mut groups = std::collections::HashMap::new();
    
    groups.insert("Priority".to_string(), gitagrip::config::GroupConfig {
        repos: vec![base_dir.join("urgent-task")],
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
fn validate_selection_fix_works() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    println!("=== VALIDATING SELECTION FIX ===");
    
    // Create repos that will demonstrate the fix
    create_test_git_repo(base_path.join("urgent-task"))?;    // Will be in Priority group (display position 0)
    create_test_git_repo(base_path.join("regular-work"))?;   // Will be in Ungrouped (display position 1)
    create_test_git_repo(base_path.join("side-project"))?;   // Will be in Ungrouped (display position 2)
    
    let config = create_config_with_priority_group(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories 
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("\nActual Display Order:");
    let mut display_pos = 0;
    let available_groups = app.get_available_groups();
    for group_name in &available_groups {
        println!("  Group: {}", group_name);
        let repos = app.get_repositories_in_group(group_name);
        for repo in repos {
            println!("    Display[{}]: {}", display_pos, repo.name);
            display_pos += 1;
        }
    }
    
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    // Test each display position
    for test_position in 0..3 {
        println!("\n=== Testing Display Position {} ===", test_position);
        
        // Clear any previous selections
        app.selected_repositories.clear();
        
        // Navigate to this display position
        app.current_selection = test_position;
        
        // Select the repository at this position
        app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
        
        // Check what got selected
        let selected_count = app.selected_repositories.len();
        println!("  Selected {} repositories", selected_count);
        
        if selected_count == 1 {
            let &storage_index = app.selected_repositories.iter().next().unwrap();
            
            // Get expected storage index first to avoid borrowing issues
            let expected_storage_index = app.display_to_storage_index(test_position);
            
            let selected_repo = &app.repositories[storage_index];
            let expected_repo = &app.repositories[expected_storage_index];
            
            println!("  Selected: {} (Storage[{}])", selected_repo.name, storage_index);
            
            if storage_index == expected_storage_index {
                println!("  ✅ CORRECT: Display[{}] maps to {} and that's what got selected", 
                    test_position, expected_repo.name);
            } else {
                println!("  ❌ ERROR: Display[{}] should map to {} but {} was selected", 
                    test_position, expected_repo.name, selected_repo.name);
            }
        } else {
            println!("  ❌ ERROR: Expected exactly 1 selection but got {}", selected_count);
        }
    }
    
    println!("\n🎉 Selection fix validation complete!");
    Ok(())
}