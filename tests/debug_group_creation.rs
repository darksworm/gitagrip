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
fn debug_real_application_flow() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create some test repositories in subdirectories (like real usage)
    let work_dir = base_path.join("work");
    fs::create_dir_all(&work_dir)?;
    create_test_git_repo(work_dir.join("frontend"))?;
    create_test_git_repo(work_dir.join("backend"))?;
    create_test_git_repo(work_dir.join("api"))?;
    create_test_git_repo(work_dir.join("mobile"))?;
    create_test_git_repo(work_dir.join("desktop"))?;
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Add repositories to app (simulate discovery)
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.add_repository(repo);
    }
    app.scan_complete = true;
    
    println!("=== REAL APPLICATION FLOW DEBUG ===");
    println!("Repositories found: {}", app.repositories.len());
    for (i, repo) in app.repositories.iter().enumerate() {
        println!("  {}: {} ({})", i, repo.name, repo.auto_group);
    }
    
    // Enter organize mode (like pressing 'o')
    app.set_mode(gitagrip::app::AppMode::Organize);
    println!("\n✓ Entered organize mode");
    
    // Select 5 repositories (indices 0-4, exactly like user described)
    println!("\n1. Selecting 5 repositories (0-4)...");
    for i in 0..5 {
        app.current_selection = i;
        app.toggle_repository_selection(i);
        println!("   Selected repo {}: {}", i, app.repositories[i].name);
    }
    
    let selected_count = app.selected_repositories.len(); 
    println!("   Total selected: {}", selected_count);
    assert_eq!(selected_count, 5, "Should have 5 repositories selected");
    
    // Press 'n' key (exactly like user)
    println!("\n2. Pressing 'n' key...");
    let n_result = app.handle_organize_key(crossterm::event::KeyCode::Char('n'))?;
    println!("   'n' key result: redraw={}", n_result);
    println!("   Input mode: {:?}", app.get_input_mode());
    
    // Check if we entered input mode
    if app.get_input_mode() == gitagrip::app::InputMode::GroupName {
        println!("   ✓ Successfully entered GroupName input mode");
    } else {
        println!("   ❌ Failed to enter GroupName input mode");
        return Ok(()); // Exit early to see what's wrong
    }
    
    // Type "TestGroup" (exactly like user)
    println!("\n3. Typing 'TestGroup'...");
    app.handle_text_input("TestGroup")?;
    println!("   Input text: '{}'", app.get_current_input_text());
    
    // Press Enter key (exactly like user - this is where it might fail)
    println!("\n4. Pressing Enter key...");
    println!("   Before Enter - selected repos: {}", app.selected_repositories.len());
    println!("   Before Enter - available groups: {:?}", app.get_available_groups());
    
    let enter_result = app.handle_organize_key(crossterm::event::KeyCode::Enter)?;
    println!("   Enter key result: redraw={}", enter_result);
    println!("   After Enter - input mode: {:?}", app.get_input_mode());
    println!("   After Enter - selected repos: {}", app.selected_repositories.len());
    println!("   After Enter - available groups: {:?}", app.get_available_groups());
    
    // Check if group was created
    if app.get_available_groups().contains(&"TestGroup".to_string()) {
        let test_group_repos = app.get_repositories_in_group("TestGroup");
        println!("   ✅ SUCCESS: TestGroup created with {} repositories", test_group_repos.len());
        for repo in test_group_repos {
            println!("     - {}", repo.name);
        }
    } else {
        println!("   ❌ FAILURE: TestGroup was NOT created");
        println!("   This matches the user's experience!");
    }
    
    Ok(())
}

#[test]
fn debug_group_creation_step_by_step() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create some test repositories
    create_test_git_repo(base_path.join("repo1"))?;
    create_test_git_repo(base_path.join("repo2"))?;
    create_test_git_repo(base_path.join("repo3"))?;
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Add repositories to app
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.add_repository(repo);
    }
    app.scan_complete = true;
    
    println!("=== DEBUG GROUP CREATION ===");
    println!("Repositories found: {}", app.repositories.len());
    for (i, repo) in app.repositories.iter().enumerate() {
        println!("  {}: {} ({})", i, repo.name, repo.auto_group);
    }
    
    // Enter organize mode
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    println!("\n1. Initial state:");
    println!("   Selected repositories: {:?}", app.get_selected_repositories());
    println!("   Available groups: {:?}", app.get_available_groups());
    
    // Select repositories 0, 1, 2
    println!("\n2. Selecting repositories 0, 1, 2...");
    app.current_selection = 0;
    app.toggle_repository_selection(0);
    app.toggle_repository_selection(1);
    app.toggle_repository_selection(2);
    
    println!("   Selected repositories after selection: {:?}", app.get_selected_repositories());
    println!("   Selected count: {}", app.selected_repositories.len());
    
    // Check if repositories are actually selected
    for i in 0..3 {
        println!("   Is repo {} selected? {}", i, app.is_repository_selected(i));
    }
    
    // Try to create group with 'n' key
    println!("\n3. Pressing 'n' to create group...");
    let redraw_needed = app.handle_organize_key(crossterm::event::KeyCode::Char('n'))?;
    println!("   Redraw needed: {}", redraw_needed);
    println!("   Input mode: {:?}", app.get_input_mode());
    println!("   Current input text: '{}'", app.get_current_input_text());
    
    // Type group name
    println!("\n4. Typing 'TestGroup'...");
    app.handle_text_input("TestGroup")?;
    println!("   Input text after typing: '{}'", app.get_current_input_text());
    
    // Confirm with Enter
    println!("\n5. Pressing Enter to confirm...");
    let redraw_needed = app.handle_organize_key(crossterm::event::KeyCode::Enter)?;
    println!("   Redraw needed: {}", redraw_needed);
    println!("   Input mode after Enter: {:?}", app.get_input_mode());
    
    // Check results
    println!("\n6. Final state:");
    println!("   Available groups: {:?}", app.get_available_groups());
    
    if app.get_available_groups().contains(&"TestGroup".to_string()) {
        let test_group_repos = app.get_repositories_in_group("TestGroup");
        println!("   TestGroup repositories: {}", test_group_repos.len());
        for repo in test_group_repos {
            println!("     - {}", repo.name);
        }
    } else {
        println!("   ERROR: TestGroup not found!");
    }
    
    println!("   Selected repositories after group creation: {:?}", app.get_selected_repositories());
    
    Ok(())
}