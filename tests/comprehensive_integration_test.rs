use anyhow::Result;
use std::fs;
use tempfile::TempDir;

fn create_test_git_repo(path: std::path::PathBuf) -> Result<()> {
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

#[test]
fn test_repository_discovery_stability() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories
    for i in 1..=10 {
        let repo_path = base_path.join(format!("stable-repo-{}", i));
        create_test_git_repo(repo_path)?;
    }
    
    // Run discovery multiple times and ensure consistent results
    let mut all_discoveries = Vec::new();
    
    for i in 0..5 {
        let discovered = gitagrip::scan::find_repos(base_path)?;
        println!("Discovery run {}: found {} repos", i + 1, discovered.len());
        
        // Check that we always find the same number
        if i == 0 {
            all_discoveries.push(discovered);
        } else {
            assert_eq!(discovered.len(), all_discoveries[0].len(), 
                      "Discovery should be stable across runs");
        }
    }
    
    assert_eq!(all_discoveries[0].len(), 10, "Should find all test repositories");
    
    Ok(())
}

#[test]
fn test_background_scanning_no_duplicates() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories
    for i in 1..=3 {
        let repo_path = base_path.join(format!("no-dup-repo-{}", i));
        create_test_git_repo(repo_path)?;
    }
    
    for _run in 0..3 {
        let (tx, rx) = crossbeam_channel::unbounded();
        
        // Start background scan
        let base_path_clone = base_path.to_path_buf();
        let handle = std::thread::spawn(move || {
            gitagrip::scan::scan_repositories_background(base_path_clone, tx)
        });
        
        // Collect all events
        let mut discovered_repos: Vec<gitagrip::scan::Repository> = Vec::new();
        let mut scan_completed = false;
        let mut scan_errors = Vec::new();
        
        // Process events with timeout
        let start_time = std::time::Instant::now();
        while start_time.elapsed() < std::time::Duration::from_secs(2) {
            match rx.recv_timeout(std::time::Duration::from_millis(50)) {
                Ok(event) => {
                    match event {
                        gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                            // Check for duplicates as they come in
                            for existing in &discovered_repos {
                                assert_ne!(existing.path, repo.path, 
                                          "Duplicate repository found: {}", repo.path.display());
                            }
                            discovered_repos.push(repo);
                        }
                        gitagrip::scan::ScanEvent::ScanCompleted => {
                            scan_completed = true;
                        }
                        gitagrip::scan::ScanEvent::ScanError(err) => {
                            scan_errors.push(err);
                        }
                    }
                }
                Err(_) => {
                    if scan_completed {
                        break;
                    }
                }
            }
        }
        
        let _ = handle.join();
        
        assert!(scan_completed, "Background scan should complete");
        assert_eq!(discovered_repos.len(), 3, "Should discover all repositories without duplicates");
        assert!(scan_errors.is_empty(), "Should not have scan errors");
    }
    
    Ok(())
}

#[test]
fn test_ui_update_stability() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories
    for i in 1..=5 {
        let repo_path = base_path.join(format!("ui-test-repo-{}", i));
        create_test_git_repo(repo_path)?;
    }
    
    let (tx, rx) = crossbeam_channel::unbounded();
    
    // Start background scan
    let base_path_clone = base_path.to_path_buf();
    std::thread::spawn(move || {
        gitagrip::scan::scan_repositories_background(base_path_clone, tx)
    });
    
    let mut app_repos = Vec::new();
    let mut update_count = 0;
    let max_updates = 10;
    
    // Simulate UI updates as repos are discovered
    for _ in 0..max_updates {
        match rx.try_recv() {
            Ok(event) => {
                match event {
                    gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                        app_repos.push(repo);
                        update_count += 1;
                        
                        // This simulates what the app does - group repositories for UI
                        let grouped = gitagrip::scan::group_repositories(&app_repos);
                        
                        // Verify grouping is stable and doesn't cause issues
                        assert!(!grouped.is_empty(), "Grouping should never be empty when repos exist");
                        let total_in_groups: usize = grouped.values().map(|v| v.len()).sum();
                        assert_eq!(total_in_groups, app_repos.len(), 
                                  "All repos should be in groups");
                    }
                    gitagrip::scan::ScanEvent::ScanCompleted => {
                        break;
                    }
                    gitagrip::scan::ScanEvent::ScanError(_) => {
                        // Ignore errors for this test
                    }
                }
            }
            Err(_) => {
                std::thread::sleep(std::time::Duration::from_millis(10));
            }
        }
    }
    
    assert!(update_count > 0, "Should receive some repository updates");
    assert_eq!(app_repos.len(), 5, "Should discover all test repositories");
    
    Ok(())
}

#[test]
fn test_ui_rendering_consistency() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories in different groups
    let test_structure = vec![
        "work/project-a",
        "work/project-b",
        "personal/blog",
        "personal/dotfiles",
        "experimental/prototype",
    ];
    
    for repo_dir in &test_structure {
        let repo_path = base_path.join(repo_dir);
        create_test_git_repo(repo_path)?;
    }
    
    // Test UI rendering consistency across multiple runs
    let mut all_ui_outputs = Vec::new();
    
    for cycle in 0..5 {
        let discovered_repos = gitagrip::scan::find_repos(base_path)?;
        let grouped_repos = gitagrip::scan::group_repositories(&discovered_repos);
        
        // Simulate what the UI does - convert to display text
        let mut ui_lines = Vec::new();
        for (group_name, repos) in &grouped_repos {
            ui_lines.push(format!("▼ {}", group_name));
            for repo in repos {
                ui_lines.push(format!("  • {}", repo.name));
            }
        }
        
        all_ui_outputs.push(ui_lines);
        println!("UI render cycle {}: {} lines", cycle, all_ui_outputs[cycle].len());
    }
    
    // All renders should produce identical output
    for i in 1..all_ui_outputs.len() {
        assert_eq!(all_ui_outputs[i].len(), all_ui_outputs[0].len(), 
                  "UI output length should be consistent");
                  
        // Check that group headers are consistent
        let headers_0: Vec<&String> = all_ui_outputs[0].iter().filter(|line| line.starts_with("▼")).collect();
        let headers_i: Vec<&String> = all_ui_outputs[i].iter().filter(|line| line.starts_with("▼")).collect();
        assert_eq!(headers_i, headers_0, "Group headers should be consistent across renders");
    }
    
    Ok(())
}

// End-to-end comprehensive test that exercises the full application stack
#[test]
fn test_end_to_end_full_stack() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create a realistic repository structure
    let repos = vec![
        ("work/backend-api", true),   // dirty repo
        ("work/frontend-web", false), // clean repo
        ("personal/dotfiles", false), // clean repo
        ("experiments/ml-project", true), // dirty repo
    ];
    
    for (repo_path, should_be_dirty) in &repos {
        let full_path = base_path.join(repo_path);
        create_test_git_repo(full_path.clone())?;
        
        if *should_be_dirty {
            fs::write(full_path.join("uncommitted.txt"), "dirty content")?;
        }
    }
    
    // Create config
    let config = gitagrip::config::Config {
        version: 1,
        base_dir: base_path.to_path_buf(),
        ui: gitagrip::config::UiConfig {
            show_ahead_behind: true,
            autosave_on_exit: false,
        },
        groups: std::collections::HashMap::new(),
    };
    
    // Initialize app (like main.rs does)
    let mut app = gitagrip::app::App::new(config.clone(), None);
    
    // Set up background scanning (like main.rs does)
    let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();
    let (status_sender, status_receiver) = crossbeam_channel::unbounded();
    
    let base_dir_clone = config.base_dir.clone();
    std::thread::spawn(move || {
        if let Err(e) = gitagrip::scan::scan_repositories_background(base_dir_clone, scan_sender) {
            eprintln!("Background scan failed: {}", e);
        }
    });
    
    // Process events exactly like the real app does
    let timeout = std::time::Duration::from_secs(3);
    let start_time = std::time::Instant::now();
    let mut git_status_started = false;
    
    while start_time.elapsed() < timeout {
        // Process scan events (exactly like app.run does)
        while let Ok(event) = scan_receiver.try_recv() {
            match event {
                gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                    app.repositories.push(repo);
                }
                gitagrip::scan::ScanEvent::ScanCompleted => {
                    app.scan_complete = true;
                    // Start git status loading (like real app)
                    if !app.repositories.is_empty() && !git_status_started {
                        app.git_status_loading = true;
                        git_status_started = true;
                        let repos_for_status = app.repositories.clone();
                        let status_sender_clone = status_sender.clone();
                        std::thread::spawn(move || {
                            if let Err(e) = gitagrip::git::compute_statuses_with_events(&repos_for_status, status_sender_clone) {
                                eprintln!("Background git status failed: {}", e);
                            }
                        });
                    }
                }
                gitagrip::scan::ScanEvent::ScanError(err) => {
                    panic!("Scan error: {}", err);
                }
            }
        }
        
        // Process git status events (exactly like app.run does)
        while let Ok(event) = status_receiver.try_recv() {
            match event {
                gitagrip::git::StatusEvent::StatusUpdated { repository, status } => {
                    app.git_statuses.insert(repository, status);
                }
                gitagrip::git::StatusEvent::StatusScanCompleted => {
                    app.git_status_loading = false;
                }
                gitagrip::git::StatusEvent::StatusError { repository: _, error: _ } => {
                    // Ignore errors for this test
                }
            }
        }
        
        // Check if we're done
        if app.scan_complete && !app.git_status_loading && !app.git_statuses.is_empty() {
            break;
        }
        
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
    
    // Verify the complete system worked
    assert_eq!(app.repositories.len(), 4, "Should discover all repositories");
    assert!(app.scan_complete, "Scan should complete");
    assert!(!app.git_status_loading, "Git status loading should complete");
    assert!(!app.git_statuses.is_empty(), "Should have git status data");
    
    // Verify git status accuracy
    let backend_status = app.git_statuses.get("backend-api").expect("Should have backend status");
    let frontend_status = app.git_statuses.get("frontend-web").expect("Should have frontend status");
    
    assert!(backend_status.is_dirty, "Backend repo should be dirty");
    assert!(!frontend_status.is_dirty, "Frontend repo should be clean");
    
    // Test scrolling works with discovered repositories
    assert_eq!(app.scroll_offset, 0);
    app.scroll_down();
    assert!(app.scroll_offset <= app.repositories.len(), "Scroll should respect bounds");
    
    Ok(())
}