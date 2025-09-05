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
fn test_exactly_like_main_rs() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create 5 repositories just like user described
    let work_dir = base_path.join("work");
    fs::create_dir_all(&work_dir)?;
    create_test_git_repo(work_dir.join("frontend"))?;
    create_test_git_repo(work_dir.join("backend"))?;
    create_test_git_repo(work_dir.join("api"))?;
    create_test_git_repo(work_dir.join("mobile"))?;
    create_test_git_repo(work_dir.join("desktop"))?;
    
    let config = create_test_config(base_path.to_path_buf());
    
    // This simulates exactly what main.rs does
    let mut app = gitagrip::app::App::new(config.clone(), None);
    
    println!("=== TESTING EXACTLY LIKE MAIN.RS ===");
    
    // Simulate repository discovery (exactly like main.rs does)
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("Repositories discovered: {}", app.repositories.len());
    for (i, repo) in app.repositories.iter().enumerate() {
        println!("  {}: {} ({})", i, repo.name, repo.auto_group);
    }
    
    // Enter organize mode (pressing 'o')
    app.set_mode(gitagrip::app::AppMode::Organize);
    println!("\n✓ Entered organize mode");
    
    // Select 5 repositories (user's exact workflow)
    println!("\n=== USER WORKFLOW START ===");
    println!("1. Selecting 5 repositories using Space key...");
    
    for i in 0..5 {
        app.current_selection = i;
        // This simulates what happens when user presses Space
        let space_result = app.handle_organize_key(crossterm::event::KeyCode::Char(' '))?;
        println!("   Space on repo {}: result={}", i, space_result);
    }
    
    println!("   Selected repositories: {}", app.selected_repositories.len());
    
    // Press 'n' to create group (user's exact action)
    println!("\n2. Pressing 'n' key to create group...");
    let n_result = app.handle_organize_key(crossterm::event::KeyCode::Char('n'))?;
    println!("   'n' key result: {}", n_result);
    println!("   Input mode: {:?}", app.get_input_mode());
    
    if app.get_input_mode() != gitagrip::app::InputMode::GroupName {
        println!("❌ FAILED: 'n' key did not enter GroupName input mode");
        return Ok(());
    }
    
    // Type group name (user's exact action)
    println!("\n3. User types 'MyGroup'...");
    
    // Simulate typing each character (like main.rs does)
    for c in "MyGroup".chars() {
        app.handle_text_input(&c.to_string())?;
    }
    println!("   Input text: '{}'", app.get_current_input_text());
    
    // Press Enter (user's exact action) - this is where main.rs calls handle_organize_key
    println!("\n4. Pressing Enter key...");
    println!("   Before Enter:");
    println!("     - Selected repos: {}", app.selected_repositories.len());
    println!("     - Available groups: {:?}", app.get_available_groups());
    
    // THIS IS THE CRITICAL PART - exactly what main.rs does when Enter is pressed
    let enter_result = app.handle_organize_key(crossterm::event::KeyCode::Enter)?;
    
    println!("   After Enter:");
    println!("     - Result: {}", enter_result);
    println!("     - Input mode: {:?}", app.get_input_mode());
    println!("     - Selected repos: {}", app.selected_repositories.len());
    println!("     - Available groups: {:?}", app.get_available_groups());
    
    // Check if group was actually created and persisted
    if app.get_available_groups().contains(&"MyGroup".to_string()) {
        let my_group_repos = app.get_repositories_in_group("MyGroup");
        println!("   ✅ SUCCESS: MyGroup created with {} repositories", my_group_repos.len());
        for repo in my_group_repos {
            println!("     - {}", repo.name);
        }
        
        // NOW TEST PERSISTENCE - this is what the user is experiencing
        println!("\n=== TESTING PERSISTENCE ===");
        
        // Check if config was actually saved to disk
        let config_path = gitagrip::config::get_default_config_path()?;
        if config_path.exists() {
            println!("Config file exists at: {}", config_path.display());
            
            // Read the config file content
            let config_content = std::fs::read_to_string(&config_path)?;
            println!("Config file content:\n{}", config_content);
            
            // Load the config properly (like main.rs does)
            let loaded_config = gitagrip::config::Config::load(None)?;
            println!("Loaded config groups: {:?}", loaded_config.groups.keys().collect::<Vec<_>>());
            
            let new_app = gitagrip::app::App::new(loaded_config, None);
            
            // Check if the group persists in a fresh app instance
            if new_app.get_available_groups().contains(&"MyGroup".to_string()) {
                println!("✅ PERSISTENCE SUCCESS: Group survived app restart");
            } else {
                println!("❌ PERSISTENCE FAILED: Group lost after app restart");
                println!("Available groups in new app: {:?}", new_app.get_available_groups());
            }
            
        } else {
            println!("❌ CONFIG FILE NOT FOUND at: {}", config_path.display());
            println!("This means the save_config() method is not working properly");
        }
        
    } else {
        println!("   ❌ FAILED: MyGroup was NOT created");
        println!("   Available groups: {:?}", app.get_available_groups());
    }
    
    Ok(())
}