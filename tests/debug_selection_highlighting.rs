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

fn create_test_config_with_manual_group(base_dir: PathBuf) -> gitagrip::config::Config {
    let mut groups = std::collections::HashMap::new();
    
    // Create a manual group to force different display vs storage order
    groups.insert("VIP".to_string(), gitagrip::config::GroupConfig {
        repos: vec![base_dir.join("zzz-last-repo")],
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
fn debug_selection_vs_display_mismatch() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    println!("=== DEMONSTRATING SELECTION VS DISPLAY MISMATCH ===");
    
    // Create repositories with names that will be stored alphabetically 
    // but displayed in group order
    create_test_git_repo(base_path.join("aaa-first-repo"))?;
    create_test_git_repo(base_path.join("mmm-middle-repo"))?;
    create_test_git_repo(base_path.join("zzz-last-repo"))?;  // This will be in VIP group
    
    let config = create_test_config_with_manual_group(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("\n=== STORAGE ORDER (self.repositories) ===");
    for (i, repo) in app.repositories.iter().enumerate() {
        println!("  Storage[{}]: {}", i, repo.name);
    }
    
    println!("\n=== DISPLAY ORDER (as shown in UI) ===");
    let available_groups = app.get_available_groups();
    let mut display_index = 0;
    for group_name in &available_groups {
        println!("  Group: {}", group_name);
        let group_repos = app.get_repositories_in_group(group_name);
        for repo in group_repos {
            let storage_index = app.repositories.iter()
                .position(|r| r.path == repo.path)
                .unwrap_or(usize::MAX);
            println!("    Display[{}]: {} (Storage[{}])", display_index, repo.name, storage_index);
            display_index += 1;
        }
    }
    
    // Enter organize mode
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    println!("\n=== TESTING SELECTION MISMATCH ===");
    
    // Test: Navigate to display position 1 and select
    println!("\nTest 1: Navigate to display position 1");
    app.current_selection = 1;  // This is in storage terms, not display terms!
    
    println!("  current_selection = {}", app.current_selection);
    println!("  This corresponds to Storage[1] = {}", 
        app.repositories.get(1).map(|r| &r.name).unwrap_or(&"INVALID".to_string()));
    
    // Select this repository
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
    
    println!("  Selected repositories: {:?}", app.selected_repositories);
    for &storage_idx in &app.selected_repositories {
        if let Some(repo) = app.repositories.get(storage_idx) {
            println!("    Selected: Storage[{}] = {}", storage_idx, repo.name);
        }
    }
    
    // The key question: Which repository is visually highlighted as "current"?
    // And which repository gets selected when we press space?
    
    println!("\n=== VISUAL EXPECTATION vs ACTUAL BEHAVIOR ===");
    println!("Expected by user: Display position 1 should be highlighted and selected");
    println!("Actual behavior: Storage index 1 gets selected");
    
    // Let's see what display position 1 actually is
    let mut display_pos = 0;
    for group_name in &available_groups {
        let group_repos = app.get_repositories_in_group(group_name);
        for repo in group_repos {
            if display_pos == 1 {
                println!("Display position 1 is actually: {} (which user expects to be selected)", repo.name);
                
                // Find its storage index
                let expected_storage_index = app.repositories.iter()
                    .position(|r| r.path == repo.path)
                    .unwrap_or(usize::MAX);
                println!("This corresponds to Storage[{}]", expected_storage_index);
                
                // Check if this is what actually got selected
                if app.selected_repositories.contains(&expected_storage_index) {
                    println!("✅ Correct: The right repository was selected");
                } else {
                    println!("❌ BUG: Wrong repository was selected!");
                    println!("   Expected Storage[{}] to be selected", expected_storage_index);
                    println!("   But selected repositories are: {:?}", app.selected_repositories);
                }
                break;
            }
            display_pos += 1;
        }
    }
    
    Ok(())
}