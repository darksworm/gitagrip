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

fn create_test_config_with_groups(base_dir: PathBuf) -> gitagrip::config::Config {
    let mut groups = std::collections::HashMap::new();
    
    // Create a manual group to force a different display order
    groups.insert("Priority".to_string(), gitagrip::config::GroupConfig {
        repos: vec![base_dir.join("important-repo")],
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
fn debug_ui_index_vs_repo_index_mapping() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    println!("=== DEBUGGING UI INDEX VS REPOSITORY INDEX MAPPING ===");
    
    // Create repositories that will have different orders when grouped
    create_test_git_repo(base_path.join("important-repo"))?; // Will be in Priority group first
    create_test_git_repo(base_path.join("alpha-repo"))?;     // Will be in auto group
    create_test_git_repo(base_path.join("beta-repo"))?;      // Will be in auto group
    create_test_git_repo(base_path.join("gamma-repo"))?;     // Will be in auto group
    
    let config = create_test_config_with_groups(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("\n=== REPOSITORY STORAGE ORDER (self.repositories) ===");
    for (i, repo) in app.repositories.iter().enumerate() {
        println!("  Storage[{}]: {} -> {}", i, repo.name, repo.path.display());
    }
    
    println!("\n=== GROUP ORGANIZATION ===");
    let available_groups = app.get_available_groups();
    for group_name in &available_groups {
        let group_repos = app.get_repositories_in_group(group_name);
        println!("  Group '{}': {} repos", group_name, group_repos.len());
        for repo in group_repos {
            // Find this repo's storage index
            let storage_index = app.repositories.iter()
                .position(|r| r.path == repo.path)
                .unwrap_or(usize::MAX);
            println!("    Display: {} -> Storage[{}] -> {}", repo.name, storage_index, repo.path.display());
        }
    }
    
    // Enter organize mode and test navigation/selection
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    println!("\n=== TESTING NAVIGATION TO UI POSITION 2 ===");
    // Navigate to what appears to be the 3rd item in the UI (index 2)
    app.current_selection = 2;
    println!("Set current_selection to UI position 2");
    println!("This should correspond to: {}", 
        if let Some(repo) = app.repositories.get(2) {
            format!("Storage[2] = {}", repo.name)
        } else {
            "INVALID INDEX".to_string()
        }
    );
    
    // Select this repository
    println!("\n=== SELECTING REPOSITORY AT UI POSITION 2 ===");
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?; // Space to select
    
    println!("Selected repositories (storage indices): {:?}", app.selected_repositories);
    for &storage_index in &app.selected_repositories {
        if let Some(repo) = app.repositories.get(storage_index) {
            println!("  Selected Storage[{}]: {} -> {}", storage_index, repo.name, repo.path.display());
        }
    }
    
    // Now create a group and see what gets moved
    println!("\n=== CREATING GROUP FROM SELECTION ===");
    app.handle_organize_key(crossterm::event::KeyCode::Char('n'))?; // Start group creation
    for c in "TestSelection".chars() {
        app.handle_text_input(&c.to_string())?;
    }
    app.handle_organize_key(crossterm::event::KeyCode::Enter)?; // Confirm
    
    println!("Group created. Let's see what ended up in the TestSelection group:");
    if let Some(test_group) = app.config.groups.get("TestSelection") {
        for repo_path in &test_group.repos {
            let storage_index = app.repositories.iter()
                .position(|r| &r.path == repo_path)
                .unwrap_or(usize::MAX);
            if let Some(repo) = app.repositories.get(storage_index) {
                println!("  Group contains Storage[{}]: {} -> {}", storage_index, repo.name, repo.path.display());
            }
        }
    }
    
    Ok(())
}