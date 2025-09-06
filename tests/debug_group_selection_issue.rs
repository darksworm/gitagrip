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

#[test]
fn debug_group_selection_mapping() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    println!("=== DEBUGGING GROUP SELECTION MAPPING ===");
    
    // Create test repositories
    create_test_git_repo(base_path.join("repo-a"))?;
    create_test_git_repo(base_path.join("repo-b"))?;
    create_test_git_repo(base_path.join("repo-c"))?;
    create_test_git_repo(base_path.join("repo-d"))?;
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("Total repositories: {}", app.repositories.len());
    for (i, repo) in app.repositories.iter().enumerate() {
        println!("  [{}] {} -> {}", i, repo.name, repo.path.display());
    }
    
    // Enter organize mode
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    // Navigate to specific repositories and select them
    println!("\n=== SELECTING SPECIFIC REPOSITORIES ===");
    
    // Select repo at index 0 and 2
    println!("Selecting repository at index 0: {}", app.repositories[0].name);
    app.current_selection = 0;
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?; // Space to select
    
    println!("Selecting repository at index 2: {}", app.repositories[2].name);
    app.current_selection = 2;
    app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?; // Space to select
    
    println!("\nSelected repositories (indices): {:?}", app.selected_repositories);
    for &index in &app.selected_repositories {
        if let Some(repo) = app.repositories.get(index) {
            println!("  Selected: [{}] {} -> {}", index, repo.name, repo.path.display());
        }
    }
    
    // Create new group
    println!("\n=== CREATING NEW GROUP ===");
    app.handle_organize_key(crossterm::event::KeyCode::Char('n'))?; // Start group creation
    
    // Type group name
    for c in "TestGroup".chars() {
        app.handle_text_input(&c.to_string())?;
    }
    
    // Confirm group creation
    app.handle_organize_key(crossterm::event::KeyCode::Enter)?;
    
    println!("Group created. Config groups:");
    for (group_name, group_config) in &app.config.groups {
        println!("  Group '{}': {} repos", group_name, group_config.repos.len());
        for repo_path in &group_config.repos {
            println!("    -> {}", repo_path.display());
        }
    }
    
    // Verify which repositories ended up in the group
    if let Some(test_group) = app.config.groups.get("TestGroup") {
        println!("\n=== VERIFYING CORRECT REPOSITORIES IN GROUP ===");
        for repo_path in &test_group.repos {
            // Find which repository this path corresponds to
            let mut found = false;
            for (i, repo) in app.repositories.iter().enumerate() {
                if &repo.path == repo_path {
                    println!("  Group contains: [{}] {} -> {}", i, repo.name, repo.path.display());
                    found = true;
                    break;
                }
            }
            if !found {
                println!("  WARNING: Group contains unknown path: {}", repo_path.display());
            }
        }
        
        // Check if selected repositories (0, 2) are actually in the group
        for &selected_index in &[0usize, 2usize] {
            if let Some(repo) = app.repositories.get(selected_index) {
                let in_group = test_group.repos.contains(&repo.path);
                println!("  Repository [{}] {} in group: {}", selected_index, repo.name, in_group);
                if !in_group {
                    println!("    ERROR: Expected repository [{}] to be in group but it's not!", selected_index);
                }
            }
        }
    }
    
    Ok(())
}