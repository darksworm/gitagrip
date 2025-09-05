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
fn debug_new_group_creation() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create a single test repository
    create_test_git_repo(base_path.join("test-repo"))?;
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Add repository to app
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.add_repository(repo);
    }
    app.scan_complete = true;
    
    println!("Initial state:");
    println!("  Mode: {:?}", app.current_mode());
    println!("  Navigation mode: {:?}", app.get_navigation_mode());
    println!("  Input mode: {:?}", app.get_input_mode());
    println!("  Available groups: {:?}", app.get_available_groups());
    
    // Test 1: Enter ORGANIZE mode
    app.set_mode(gitagrip::app::AppMode::Organize);
    println!("\nAfter entering ORGANIZE mode:");
    println!("  Mode: {:?}", app.current_mode());
    
    // Test 2: Switch to Group navigation mode
    app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
    println!("  Navigation mode: {:?}", app.get_navigation_mode());
    
    // Test 3: Try to create new group with 'n' key
    println!("\nTrying to create new group with 'n' key...");
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('n'))?;
    println!("  Redraw needed: {}", redraw_needed);
    println!("  Input mode after 'n': {:?}", app.get_input_mode());
    println!("  Current input text: '{}'", app.get_current_input_text());
    
    // Test 4: Try typing text input
    if app.get_input_mode() == gitagrip::app::InputMode::GroupName {
        println!("\nTyping 'NewGroup'...");
        app.handle_text_input("NewGroup")?;
        println!("  Current input text after typing: '{}'", app.get_current_input_text());
        
        // Test 5: Confirm with Enter
        println!("\nConfirming with Enter...");
        let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Enter)?;
        println!("  Redraw needed: {}", redraw_needed);
        println!("  Input mode after Enter: {:?}", app.get_input_mode());
        println!("  Available groups after creation: {:?}", app.get_available_groups());
    } else {
        println!("ERROR: 'n' key didn't enter input mode!");
    }
    
    Ok(())
}

#[test]
fn debug_paste_functionality() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories
    create_test_git_repo(base_path.join("repo1"))?;
    create_test_git_repo(base_path.join("repo2"))?;
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Add repositories to app
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.add_repository(repo);
    }
    app.scan_complete = true;
    
    println!("Repositories found: {}", app.repositories.len());
    for (i, repo) in app.repositories.iter().enumerate() {
        println!("  {}: {}", i, repo.name);
    }
    
    // Enter ORGANIZE mode
    app.set_mode(gitagrip::app::AppMode::Organize);
    app.set_navigation_mode(gitagrip::app::NavigationMode::Repository);
    
    println!("\nSelecting repositories...");
    
    // Select first repository with Space
    app.current_selection = 0;
    println!("Current selection: {}", app.current_selection);
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char(' '))?;
    println!("  Space key redraw needed: {}", redraw_needed);
    println!("  Is repo 0 selected: {}", app.is_repository_selected(0));
    println!("  Selected repositories: {:?}", app.get_selected_repositories());
    println!("  Marked repositories: {:?}", app.get_marked_repositories());
    
    // Switch to group navigation and create a group
    app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
    app.handle_mode_specific_key(crossterm::event::KeyCode::Char('n'))?;
    app.handle_text_input("TestGroup")?;
    app.handle_mode_specific_key(crossterm::event::KeyCode::Enter)?;
    
    println!("\nAfter creating TestGroup:");
    println!("  Available groups: {:?}", app.get_available_groups());
    println!("  Current target group: {}", app.get_current_target_group());
    
    // Try to paste
    println!("\nTrying to paste with 'p' key...");
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('p'))?;
    println!("  Paste redraw needed: {}", redraw_needed);
    println!("  Marked repositories after paste: {:?}", app.get_marked_repositories());
    println!("  Selected repositories after paste: {:?}", app.get_selected_repositories());
    
    // Check if repositories were moved to the group
    let test_group_repos = app.get_repositories_in_group("TestGroup");
    println!("  TestGroup repositories: {}", test_group_repos.len());
    for repo in &test_group_repos {
        println!("    {}", repo.name);
    }
    
    Ok(())
}

#[test]
fn debug_scrolling_in_organize_mode() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create many repositories to test scrolling
    for i in 0..10 {
        create_test_git_repo(base_path.join(format!("repo-{:02}", i)))?;
    }
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Add repositories to app
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.add_repository(repo);
    }
    app.scan_complete = true;
    
    println!("Created {} repositories for scrolling test", app.repositories.len());
    
    // Enter ORGANIZE mode
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    println!("Initial state:");
    println!("  Current selection: {}", app.current_selection);
    println!("  Scroll offset: {}", app.scroll_offset);
    
    // Try scrolling down
    println!("\nTrying Down arrow key...");
    let redraw_needed = app.handle_mode_specific_key(crossterm::event::KeyCode::Down)?;
    println!("  Redraw needed: {}", redraw_needed);
    println!("  Current selection after: {}", app.current_selection);
    println!("  Scroll offset after: {}", app.scroll_offset);
    
    // Debug the content calculations
    println!("\nDebugging scroll calculations:");
    println!("  Total repositories: {}", app.repositories.len());
    println!("  Estimated total content lines: {}", app.calculate_total_content_lines());
    println!("  Current selection line index: {}", app.calculate_selection_line_index());
    
    // Try many more down presses to trigger scrolling
    for i in 1..=9 {
        app.handle_mode_specific_key(crossterm::event::KeyCode::Down)?;
        println!("  After {} downs - selection: {}, scroll: {}, line_index: {}", 
                 i+1, app.current_selection, app.scroll_offset, app.calculate_selection_line_index());
    }
    
    Ok(())
}

#[test]
fn test_move_repos_between_groups_with_persistence() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    let config_path = temp_dir.path().join("config.toml");
    
    // Create test repositories
    create_test_git_repo(base_path.join("frontend-app"))?;
    create_test_git_repo(base_path.join("backend-api"))?;
    create_test_git_repo(base_path.join("mobile-app"))?;
    
    // Phase 1: Initial setup - create app and organize repositories
    {
        let mut config = create_test_config(base_path.to_path_buf());
        
        // Add an existing "Legacy" group
        config.groups.insert("Legacy".to_string(), gitagrip::config::GroupConfig {
            repos: vec![],
        });
        
        // Save initial config
        config.save(&config_path)?;
        
        let mut app = gitagrip::app::App::new(config, None);
        
        // Discover repositories
        let discovered_repos = gitagrip::scan::find_repos(base_path)?;
        for repo in discovered_repos {
            app.add_repository(repo);
        }
        app.scan_complete = true;
        
        println!("=== PHASE 1: Initial Organization ===");
        println!("Repositories found: {}", app.repositories.len());
        for (i, repo) in app.repositories.iter().enumerate() {
            println!("  {}: {} ({})", i, repo.name, repo.auto_group);
        }
        
        // Move frontend-app to Legacy group first
        app.set_mode(gitagrip::app::AppMode::Organize);
        
        // Find and select frontend-app
        for (i, repo) in app.repositories.iter().enumerate() {
            if repo.name == "frontend-app" {
                app.current_selection = i;
                break;
            }
        }
        
        // Select it with Space (repository navigation mode)
        app.set_navigation_mode(gitagrip::app::NavigationMode::Repository);
        app.handle_mode_specific_key(crossterm::event::KeyCode::Char(' '))?;
        
        // Switch to group navigation and navigate to Legacy
        app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
        app.navigate_to_group("Legacy")?;
        
        // Paste to Legacy group
        app.handle_mode_specific_key(crossterm::event::KeyCode::Char('p'))?;
        
        println!("\nAfter moving frontend-app to Legacy:");
        let legacy_repos = app.get_repositories_in_group("Legacy");
        println!("  Legacy group: {} repositories", legacy_repos.len());
        for repo in &legacy_repos {
            println!("    - {}", repo.name);
        }
        
        // Save the config after this change
        app.config.save(&config_path)?;
        
        println!("  Config saved after first move");
    }
    
    // Phase 2: Restart app (simulate app restart) and create new group
    {
        println!("\n=== PHASE 2: App Restart and New Group Creation ===");
        
        // Load config from file (simulates app restart)
        let loaded_config = gitagrip::config::Config::load(Some(config_path.clone()))?;
        let mut app = gitagrip::app::App::new(loaded_config, None);
        
        // Rediscover repositories (like app startup)
        let discovered_repos = gitagrip::scan::find_repos(base_path)?;
        for repo in discovered_repos {
            app.add_repository(repo);
        }
        app.scan_complete = true;
        
        // Verify Legacy group persisted
        let legacy_repos = app.get_repositories_in_group("Legacy");
        println!("Legacy group after restart: {} repositories", legacy_repos.len());
        assert_eq!(legacy_repos.len(), 1, "Legacy group should have 1 repository after restart");
        assert_eq!(legacy_repos[0].name, "frontend-app", "Should be frontend-app in Legacy");
        
        // Enter organize mode and create "Production" group
        app.set_mode(gitagrip::app::AppMode::Organize);
        app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
        
        // Create new "Production" group
        app.handle_mode_specific_key(crossterm::event::KeyCode::Char('n'))?;
        app.handle_text_input("Production")?;
        app.handle_mode_specific_key(crossterm::event::KeyCode::Enter)?;
        
        println!("Available groups after creating Production: {:?}", app.get_available_groups());
        
        // Move frontend-app FROM Legacy TO Production
        // First, switch to repository mode and find frontend-app
        app.set_navigation_mode(gitagrip::app::NavigationMode::Repository);
        for (i, repo) in app.repositories.iter().enumerate() {
            if repo.name == "frontend-app" {
                app.current_selection = i;
                break;
            }
        }
        
        // Select frontend-app with Space 
        app.handle_mode_specific_key(crossterm::event::KeyCode::Char(' '))?;
        println!("Selected repositories: {:?}", app.get_selected_repository_names());
        
        // Switch to group navigation and go to Production group
        app.set_navigation_mode(gitagrip::app::NavigationMode::Group);
        app.navigate_to_group("Production")?;
        println!("Current target group: {}", app.get_current_target_group());
        
        // Paste to Production group
        let paste_result = app.handle_mode_specific_key(crossterm::event::KeyCode::Char('p'))?;
        println!("Paste result: {}", paste_result);
        
        // Verify the move
        let production_repos = app.get_repositories_in_group("Production");
        let legacy_repos_after = app.get_repositories_in_group("Legacy");
        
        println!("\nAfter moving frontend-app to Production:");
        println!("  Production group: {} repositories", production_repos.len());
        for repo in &production_repos {
            println!("    - {}", repo.name);
        }
        println!("  Legacy group: {} repositories", legacy_repos_after.len());
        
        // Save config again
        app.config.save(&config_path)?;
    }
    
    // Phase 3: Final restart to verify persistence
    {
        println!("\n=== PHASE 3: Final Restart Verification ===");
        
        let final_config = gitagrip::config::Config::load(Some(config_path))?;
        let mut app = gitagrip::app::App::new(final_config, None);
        
        let discovered_repos = gitagrip::scan::find_repos(base_path)?;
        for repo in discovered_repos {
            app.add_repository(repo);
        }
        app.scan_complete = true;
        
        // Verify final state
        let production_repos = app.get_repositories_in_group("Production");
        let legacy_repos = app.get_repositories_in_group("Legacy");
        let ungrouped_repos = app.get_repositories_in_group("Ungrouped");
        
        println!("Final state after restart:");
        println!("  Production: {} repos", production_repos.len());
        for repo in &production_repos {
            println!("    - {}", repo.name);
        }
        println!("  Legacy: {} repos", legacy_repos.len());
        println!("  Ungrouped: {} repos", ungrouped_repos.len());
        for repo in &ungrouped_repos {
            println!("    - {}", repo.name);
        }
        
        // Assertions
        assert_eq!(production_repos.len(), 1, "Production should have 1 repository");
        assert_eq!(production_repos[0].name, "frontend-app", "Should be frontend-app in Production");
        assert_eq!(legacy_repos.len(), 0, "Legacy should be empty after move");
        assert_eq!(ungrouped_repos.len(), 2, "Should have 2 ungrouped repos (backend-api, mobile-app)");
    }
    
    println!("\n✅ Group persistence and repository movement test passed!");
    Ok(())
}