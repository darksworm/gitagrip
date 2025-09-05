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
fn test_vim_navigation_guiding_star() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    println!("=== VIM NAVIGATION GUIDING STAR TEST ===");
    
    // Create many repositories to test navigation with
    // Structure: multiple groups with multiple repos each for comprehensive testing
    let work_dir = base_path.join("work");
    let personal_dir = base_path.join("personal");
    let projects_dir = base_path.join("projects");
    
    fs::create_dir_all(&work_dir)?;
    fs::create_dir_all(&personal_dir)?;
    fs::create_dir_all(&projects_dir)?;
    
    // Work group (repos 0-7)
    for i in 1..=8 {
        create_test_git_repo(work_dir.join(format!("work-repo-{:02}", i)))?;
    }
    
    // Personal group (repos 8-12) 
    for i in 1..=5 {
        create_test_git_repo(personal_dir.join(format!("personal-{}", i)))?;
    }
    
    // Projects group (repos 13-19)
    for i in 1..=7 {
        create_test_git_repo(projects_dir.join(format!("project-{}", i)))?;
    }
    
    let config = create_test_config(base_path.to_path_buf());
    let mut app = gitagrip::app::App::new(config, None);
    
    // Discover repositories (exactly like main.rs does)
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    for repo in discovered_repos {
        app.repositories.push(repo);
    }
    app.scan_complete = true;
    
    println!("Total repositories: {}", app.repositories.len());
    assert!(app.repositories.len() >= 15, "Should have at least 15 repos for comprehensive testing");
    
    // Enter organize mode for navigation testing
    app.set_mode(gitagrip::app::AppMode::Organize);
    
    println!("\n=== TESTING VIM NAVIGATION KEYS ===");
    
    // Test 1: gg - Go to top (first repository)
    println!("\n1. Testing 'gg' - Go to top");
    
    // Start somewhere in the middle
    app.current_selection = 7;
    println!("   Starting position: {}", app.current_selection);
    
    // Press 'g' twice to go to top
    let g1_result = app.handle_organize_key(crossterm::event::KeyCode::Char('g'))?;
    println!("   First 'g' result: {}", g1_result);
    
    let g2_result = app.handle_organize_key(crossterm::event::KeyCode::Char('g'))?;
    println!("   Second 'g' result: {}", g2_result);
    println!("   Position after 'gg': {}", app.current_selection);
    
    assert_eq!(app.current_selection, 0, "gg should move to first repository (index 0)");
    println!("   ✅ 'gg' works correctly");
    
    // Test 2: G - Go to bottom (last repository)
    println!("\n2. Testing 'G' - Go to bottom");
    
    let g_result = app.handle_organize_key(crossterm::event::KeyCode::Char('G'))?;
    println!("   'G' result: {}", g_result);
    println!("   Position after 'G': {}", app.current_selection);
    
    let expected_last = app.repositories.len() - 1;
    assert_eq!(app.current_selection, expected_last, "G should move to last repository");
    println!("   ✅ 'G' works correctly (moved to index {})", expected_last);
    
    // Test 3: Page Down navigation
    println!("\n3. Testing Page Down navigation");
    
    // Go back to top first
    app.current_selection = 0;
    
    let page_down_result = app.handle_organize_key(crossterm::event::KeyCode::PageDown)?;
    println!("   Page Down result: {}", page_down_result);
    println!("   Position after Page Down: {}", app.current_selection);
    
    // Page down should move by a reasonable amount (like 10 items)
    assert!(app.current_selection > 5, "Page Down should move significantly down");
    assert!(app.current_selection < app.repositories.len(), "Page Down shouldn't go past end");
    
    let page_down_position = app.current_selection;
    println!("   ✅ Page Down works (moved to index {})", page_down_position);
    
    // Test 4: Page Up navigation
    println!("\n4. Testing Page Up navigation");
    
    let page_up_result = app.handle_organize_key(crossterm::event::KeyCode::PageUp)?;
    println!("   Page Up result: {}", page_up_result);
    println!("   Position after Page Up: {}", app.current_selection);
    
    // Page up should move back up significantly
    assert!(app.current_selection < page_down_position, "Page Up should move up from previous position");
    
    println!("   ✅ Page Up works (moved to index {})", app.current_selection);
    
    // Test 5: Boundary testing - gg at top, G at bottom
    println!("\n5. Testing boundary conditions");
    
    // Test gg when already at top
    app.current_selection = 0;
    app.handle_organize_key(crossterm::event::KeyCode::Char('g'))?;
    app.handle_organize_key(crossterm::event::KeyCode::Char('g'))?;
    assert_eq!(app.current_selection, 0, "gg at top should stay at top");
    println!("   ✅ 'gg' at top stays at top");
    
    // Test G when already at bottom
    app.current_selection = app.repositories.len() - 1;
    app.handle_organize_key(crossterm::event::KeyCode::Char('G'))?;
    assert_eq!(app.current_selection, app.repositories.len() - 1, "G at bottom should stay at bottom");
    println!("   ✅ 'G' at bottom stays at bottom");
    
    // Test Page Down at bottom
    let bottom_pos = app.repositories.len() - 1;
    app.current_selection = bottom_pos;
    app.handle_organize_key(crossterm::event::KeyCode::PageDown)?;
    assert_eq!(app.current_selection, bottom_pos, "Page Down at bottom should stay at bottom");
    println!("   ✅ Page Down at bottom stays at bottom");
    
    // Test Page Up at top
    app.current_selection = 0;
    app.handle_organize_key(crossterm::event::KeyCode::PageUp)?;
    assert_eq!(app.current_selection, 0, "Page Up at top should stay at top");
    println!("   ✅ Page Up at top stays at top");
    
    // Test 6: Normal mode navigation (should also work)
    println!("\n6. Testing vim navigation in NORMAL mode");
    
    app.set_mode(gitagrip::app::AppMode::Normal);
    app.current_selection = 5;
    
    // Test gg in normal mode
    app.handle_mode_specific_key(crossterm::event::KeyCode::Char('g'))?;
    app.handle_mode_specific_key(crossterm::event::KeyCode::Char('g'))?;
    assert_eq!(app.current_selection, 0, "gg should work in normal mode too");
    
    // Test G in normal mode  
    app.handle_mode_specific_key(crossterm::event::KeyCode::Char('G'))?;
    assert_eq!(app.current_selection, app.repositories.len() - 1, "G should work in normal mode too");
    
    println!("   ✅ Vim navigation works in both NORMAL and ORGANIZE modes");
    
    println!("\n🎉 ALL VIM NAVIGATION TESTS PASSED!");
    println!("✅ gg (go to top) implemented");  
    println!("✅ G (go to bottom) implemented");
    println!("✅ Page Up/Down implemented");
    println!("✅ Boundary conditions handled");
    println!("✅ Works in both NORMAL and ORGANIZE modes");
    
    Ok(())
}