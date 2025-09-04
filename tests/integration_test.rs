use anyhow::Result;
use clap::Parser;
use std::fs;
use std::path::PathBuf;
use tempfile::TempDir;

// This is our "guiding star" integration test for M1
// It tests the complete flow: CLI args -> config loading -> app initialization
#[test]
fn test_m1_config_and_cli_integration() -> Result<()> {
    // Setup: Create a temporary directory for our test config
    let temp_dir = TempDir::new()?;
    let config_dir = temp_dir.path().join(".config").join("yarg");
    fs::create_dir_all(&config_dir)?;
    
    let config_file = config_dir.join("yarg.toml");
    
    // Create a test config file with the expected schema
    let test_config = r#"
version = 1
base_dir = "/tmp/test/repos"

[ui]
show_ahead_behind = true
autosave_on_exit = false

[groups.Work]
repos = [
  "/tmp/test/repos/acme-api",
  "/tmp/test/repos/acme-web",
]

[groups.Personal]
repos = [
  "/tmp/test/repos/dotfiles",
]
"#;
    fs::write(&config_file, test_config)?;
    
    // Test 1: Load config from file
    let config = yarg::config::Config::load(Some(config_file.clone()))?;
    
    assert_eq!(config.version, 1);
    assert_eq!(config.base_dir, PathBuf::from("/tmp/test/repos"));
    assert_eq!(config.ui.show_ahead_behind, true);
    assert_eq!(config.ui.autosave_on_exit, false);
    
    // Check groups
    assert_eq!(config.groups.len(), 2);
    assert!(config.groups.contains_key("Work"));
    assert!(config.groups.contains_key("Personal"));
    
    let work_group = &config.groups["Work"];
    assert_eq!(work_group.repos.len(), 2);
    assert!(work_group.repos.contains(&PathBuf::from("/tmp/test/repos/acme-api")));
    
    // Test 2: CLI override should work
    let cli_args = yarg::cli::CliArgs {
        base_dir: Some(PathBuf::from("/override/path")),
        config: None,
    };
    
    let final_config = yarg::config::Config::from_cli_and_file(cli_args, Some(config_file))?;
    assert_eq!(final_config.base_dir, PathBuf::from("/override/path")); // CLI should override
    assert_eq!(final_config.ui.show_ahead_behind, true); // Other settings preserved
    
    // Test 3: Config should be saveable
    let new_config_file = config_dir.join("yarg_test_save.toml");
    final_config.save(&new_config_file)?;
    
    // Verify saved config can be loaded back
    let reloaded_config = yarg::config::Config::load(Some(new_config_file))?;
    assert_eq!(reloaded_config.base_dir, PathBuf::from("/override/path"));
    
    // Test 4: Default config creation
    let nonexistent_file = temp_dir.path().join("nonexistent.toml");
    let default_config = yarg::config::Config::load(Some(nonexistent_file.clone()))?;
    
    // Should create default config
    assert_eq!(default_config.version, 1);
    assert!(!default_config.base_dir.as_os_str().is_empty());
    assert_eq!(default_config.ui.show_ahead_behind, true);
    assert_eq!(default_config.ui.autosave_on_exit, true);
    assert!(default_config.groups.is_empty());
    
    Ok(())
}

// Test the XDG config path resolution
#[test] 
fn test_xdg_config_path_resolution() -> Result<()> {
    let config_path = yarg::config::get_default_config_path()?;
    
    // Should end with yarg/yarg.toml (may be in different locations on different OS)
    assert!(config_path.ends_with("yarg/yarg.toml"));
    // On macOS it might be in ~/Library/Application Support instead of ~/.config
    let path_str = config_path.to_string_lossy();
    assert!(path_str.contains("yarg") && path_str.ends_with("yarg.toml"));
    
    Ok(())
}

// Test CLI parsing
#[test]
fn test_cli_parsing() -> Result<()> {
    // This will test that clap parsing works correctly
    let args = yarg::cli::CliArgs::parse_from(&["yarg", "--base-dir", "/test/path"]);
    
    assert_eq!(args.base_dir, Some(PathBuf::from("/test/path")));
    assert_eq!(args.config, None);
    
    let args_with_config = yarg::cli::CliArgs::parse_from(&[
        "yarg", 
        "--base-dir", "/test/path",
        "--config", "/custom/config.toml"
    ]);
    
    assert_eq!(args_with_config.base_dir, Some(PathBuf::from("/test/path")));
    assert_eq!(args_with_config.config, Some(PathBuf::from("/custom/config.toml")));
    
    Ok(())
}

// M2 Guiding Star Test: Repository Discovery & Background Scanning
#[test]
fn test_m2_repository_discovery_integration() -> Result<()> {
    // Setup: Create a temporary directory structure with Git repos
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create directory structure:
    // temp/
    // ├── work/
    // │   ├── project-a/  (.git)
    // │   └── project-b/  (.git)
    // ├── personal/
    // │   └── dotfiles/   (.git)
    // └── ungrouped-repo/ (.git)
    
    let work_dir = base_path.join("work");
    let personal_dir = base_path.join("personal");
    fs::create_dir_all(&work_dir)?;
    fs::create_dir_all(&personal_dir)?;
    
    // Create actual Git repositories using git2
    let repos = vec![
        work_dir.join("project-a"),
        work_dir.join("project-b"), 
        personal_dir.join("dotfiles"),
        base_path.join("ungrouped-repo"),
    ];
    
    for repo_path in &repos {
        fs::create_dir_all(repo_path)?;
        let repo = git2::Repository::init(repo_path)?;
        
        // Create a commit so the repo has some content
        let sig = git2::Signature::now("Test User", "test@example.com")?;
        let tree_id = {
            let mut index = repo.index()?;
            // Create a test file
            std::fs::write(repo_path.join("README.md"), "# Test Repo")?;
            index.add_path(std::path::Path::new("README.md"))?;
            index.write()?;
            index.write_tree()?
        };
        let tree = repo.find_tree(tree_id)?;
        repo.commit(
            Some("HEAD"),
            &sig,
            &sig,
            "Initial commit",
            &tree,
            &[],
        )?;
    }
    
    // Test 1: Repository discovery should find all repos
    let discovered_repos = yarg::scan::find_repos(base_path)?;
    
    assert_eq!(discovered_repos.len(), 4);
    
    // Check that all repos were found
    let repo_paths: Vec<_> = discovered_repos.iter().map(|r| &r.path).collect();
    for expected_repo in &repos {
        assert!(repo_paths.contains(&expected_repo), 
                "Repository not found: {}", expected_repo.display());
    }
    
    // Test 2: Auto-grouping should work based on parent directory
    let grouped_repos = yarg::scan::group_repositories(&discovered_repos);
    
    // Should have: Auto: work (2), Auto: personal (1), Ungrouped (1)
    assert_eq!(grouped_repos.len(), 3);
    
    let work_group = grouped_repos.get("Auto: work").expect("Work group should exist");
    assert_eq!(work_group.len(), 2);
    
    let personal_group = grouped_repos.get("Auto: personal").expect("Personal group should exist");
    assert_eq!(personal_group.len(), 1);
    
    let ungrouped = grouped_repos.get("Ungrouped").expect("Ungrouped should exist");
    assert_eq!(ungrouped.len(), 1);
    
    // Test 3: Background scanning with channels should work
    let (tx, rx) = crossbeam_channel::unbounded();
    
    // Spawn background scan
    let base_path_clone = base_path.to_path_buf();
    std::thread::spawn(move || {
        if let Err(e) = yarg::scan::scan_repositories_background(base_path_clone, tx) {
            eprintln!("Background scan failed: {}", e);
        }
    });
    
    // Collect events from channel
    let mut received_repos = Vec::new();
    let mut scan_completed = false;
    
    // Timeout to avoid infinite wait
    let timeout = std::time::Duration::from_secs(5);
    let start = std::time::Instant::now();
    
    while start.elapsed() < timeout && !scan_completed {
        match rx.recv_timeout(std::time::Duration::from_millis(100)) {
            Ok(event) => {
                match event {
                    yarg::scan::ScanEvent::RepoDiscovered(repo) => {
                        received_repos.push(repo);
                    }
                    yarg::scan::ScanEvent::ScanCompleted => {
                        scan_completed = true;
                    }
                    yarg::scan::ScanEvent::ScanError(err) => {
                        panic!("Scan error: {}", err);
                    }
                }
            }
            Err(crossbeam_channel::RecvTimeoutError::Timeout) => {
                // Continue waiting
                continue;
            }
            Err(crossbeam_channel::RecvTimeoutError::Disconnected) => {
                break;
            }
        }
    }
    
    assert!(scan_completed, "Scan should complete");
    assert_eq!(received_repos.len(), 4, "Should receive all 4 repositories");
    
    // Test 4: Repository metadata should be populated
    for repo in &received_repos {
        assert!(!repo.name.is_empty(), "Repository name should not be empty");
        assert!(repo.path.exists(), "Repository path should exist");
        assert!(!repo.auto_group.is_empty(), "Auto group should be populated");
        
        // Check that .git directory exists
        assert!(repo.path.join(".git").exists(), "Should have .git directory");
    }
    
    Ok(())
}

// Test Repository struct serialization for config persistence
#[test]
fn test_repository_struct() -> Result<()> {
    let repo = yarg::scan::Repository {
        name: "test-repo".to_string(),
        path: PathBuf::from("/path/to/repo"),
        auto_group: "Auto: parent".to_string(),
    };
    
    // Test Display trait
    let display_str = format!("{}", repo);
    assert!(display_str.contains("test-repo"));
    
    // Test cloning and equality
    let repo_clone = repo.clone();
    assert_eq!(repo, repo_clone);
    
    Ok(())
}

// Test edge cases for repository discovery
#[test]
fn test_repository_discovery_edge_cases() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Test 1: Empty directory
    let repos = yarg::scan::find_repos(base_path)?;
    assert!(repos.is_empty(), "Should find no repos in empty directory");
    
    // Test 2: Directory with nested .git (should not descend into repo)
    let outer_repo = base_path.join("outer-repo");
    let inner_dir = outer_repo.join("inner");
    fs::create_dir_all(&inner_dir)?;
    
    // Create outer repo
    git2::Repository::init(&outer_repo)?;
    
    // Create what looks like an inner repo
    fs::create_dir_all(inner_dir.join(".git"))?;
    
    let repos = yarg::scan::find_repos(base_path)?;
    assert_eq!(repos.len(), 1, "Should only find outer repo, not descend into it");
    assert_eq!(repos[0].path, outer_repo);
    
    Ok(())
}

// Test for UI stability and no duplicate discoveries
#[test]
fn test_repository_discovery_stability() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create several repos in nested structure
    let repos = vec![
        base_path.join("frontend").join("app"),
        base_path.join("frontend").join("shared"),
        base_path.join("backend").join("api"),
        base_path.join("backend").join("worker"),
        base_path.join("tools").join("deploy"),
    ];
    
    for repo_path in &repos {
        fs::create_dir_all(repo_path)?;
        let repo = git2::Repository::init(repo_path)?;
        
        // Create a commit
        let sig = git2::Signature::now("Test User", "test@example.com")?;
        let tree_id = {
            let mut index = repo.index()?;
            std::fs::write(repo_path.join("README.md"), "# Test")?;
            index.add_path(std::path::Path::new("README.md"))?;
            index.write()?;
            index.write_tree()?
        };
        let tree = repo.find_tree(tree_id)?;
        repo.commit(Some("HEAD"), &sig, &sig, "Initial commit", &tree, &[])?;
    }
    
    // Run discovery multiple times to check for stability
    let mut all_discoveries = Vec::new();
    
    for i in 0..5 {
        let discovered = yarg::scan::find_repos(base_path)?;
        println!("Discovery run {}: found {} repos", i + 1, discovered.len());
        
        // Check that we always find the same number
        assert_eq!(discovered.len(), 5, "Should always find exactly 5 repositories");
        
        // Check for duplicates within a single run
        let mut paths = Vec::new();
        for repo in &discovered {
            assert!(!paths.contains(&repo.path), 
                   "Duplicate repository found in single scan: {}", repo.path.display());
            paths.push(repo.path.clone());
        }
        
        all_discoveries.push(discovered);
    }
    
    // All runs should find the exact same repositories
    let first_run = &all_discoveries[0];
    for (run_idx, run_repos) in all_discoveries.iter().enumerate().skip(1) {
        assert_eq!(first_run.len(), run_repos.len(), 
                  "Run {} found different number of repos than first run", run_idx + 1);
        
        // Check that the same repos are found (order might differ)
        let first_paths: std::collections::HashSet<_> = first_run.iter().map(|r| &r.path).collect();
        let run_paths: std::collections::HashSet<_> = run_repos.iter().map(|r| &r.path).collect();
        
        assert_eq!(first_paths, run_paths, 
                  "Run {} found different repositories than first run", run_idx + 1);
    }
    
    Ok(())
}

// Test for background scanning race conditions and duplicates
#[test]
fn test_background_scanning_no_duplicates() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create repos
    let repos = vec![
        base_path.join("project1"),
        base_path.join("project2"),
        base_path.join("project3"),
    ];
    
    for repo_path in &repos {
        fs::create_dir_all(repo_path)?;
        git2::Repository::init(repo_path)?;
    }
    
    // Test background scanning multiple times
    for test_run in 0..3 {
        let (tx, rx) = crossbeam_channel::unbounded();
        
        // Start background scan
        let base_path_clone = base_path.to_path_buf();
        let handle = std::thread::spawn(move || {
            yarg::scan::scan_repositories_background(base_path_clone, tx)
        });
        
        // Collect all events
        let mut discovered_repos: Vec<yarg::scan::Repository> = Vec::new();
        let mut scan_completed = false;
        let mut scan_errors = Vec::new();
        
        let timeout = std::time::Duration::from_secs(2);
        let start = std::time::Instant::now();
        
        while start.elapsed() < timeout && !scan_completed {
            match rx.recv_timeout(std::time::Duration::from_millis(50)) {
                Ok(event) => {
                    match event {
                        yarg::scan::ScanEvent::RepoDiscovered(repo) => {
                            // Check for duplicates as they come in
                            for existing in &discovered_repos {
                                assert_ne!(existing.path, repo.path, 
                                          "Duplicate repository discovered: {}", repo.path.display());
                            }
                            discovered_repos.push(repo);
                        }
                        yarg::scan::ScanEvent::ScanCompleted => {
                            scan_completed = true;
                        }
                        yarg::scan::ScanEvent::ScanError(err) => {
                            scan_errors.push(err);
                        }
                    }
                }
                Err(crossbeam_channel::RecvTimeoutError::Timeout) => continue,
                Err(crossbeam_channel::RecvTimeoutError::Disconnected) => break,
            }
        }
        
        handle.join().expect("Background thread should complete successfully")?;
        
        assert!(scan_completed, "Scan should complete in test run {}", test_run + 1);
        assert!(scan_errors.is_empty(), "Should have no scan errors: {:?}", scan_errors);
        assert_eq!(discovered_repos.len(), 3, 
                  "Should find exactly 3 repos in test run {}", test_run + 1);
        
        println!("Background scan test run {} completed successfully", test_run + 1);
    }
    
    Ok(())
}

// Test rapid UI updates don't cause issues
#[test] 
fn test_ui_update_stability() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create repos
    for i in 0..10 {
        let repo_path = base_path.join(format!("repo{}", i));
        fs::create_dir_all(&repo_path)?;
        git2::Repository::init(&repo_path)?;
    }
    
    // Simulate rapid UI updates like the app would do
    let (tx, rx) = crossbeam_channel::unbounded();
    
    // Start background scan
    let base_path_clone = base_path.to_path_buf();
    std::thread::spawn(move || {
        yarg::scan::scan_repositories_background(base_path_clone, tx)
    });
    
    let mut app_repos = Vec::new();
    let mut update_count = 0;
    let max_updates = 100; // Prevent infinite loop
    
    while update_count < max_updates {
        match rx.try_recv() {
            Ok(event) => {
                match event {
                    yarg::scan::ScanEvent::RepoDiscovered(repo) => {
                        app_repos.push(repo);
                        update_count += 1;
                        
                        // This simulates what the app does - group repositories for UI
                        let grouped = yarg::scan::group_repositories(&app_repos);
                        
                        // Verify grouping is stable and doesn't cause issues
                        assert!(!grouped.is_empty(), "Grouping should never be empty when repos exist");
                        
                        let total_in_groups: usize = grouped.values().map(|v| v.len()).sum();
                        assert_eq!(total_in_groups, app_repos.len(), 
                                  "All repos should be in groups");
                    }
                    yarg::scan::ScanEvent::ScanCompleted => {
                        break;
                    }
                    yarg::scan::ScanEvent::ScanError(_) => {
                        // Ignore errors for this test
                    }
                }
            }
            Err(crossbeam_channel::TryRecvError::Empty) => {
                // No more events right now
                std::thread::sleep(std::time::Duration::from_millis(1));
            }
            Err(crossbeam_channel::TryRecvError::Disconnected) => {
                break;
            }
        }
    }
    
    assert_eq!(app_repos.len(), 10, "Should discover all 10 repositories");
    
    // Check final state for duplicates
    let mut seen_paths = std::collections::HashSet::new();
    for repo in &app_repos {
        assert!(seen_paths.insert(repo.path.clone()), 
               "Duplicate repository in final state: {}", repo.path.display());
    }
    
    Ok(())
}

// Test for consistent UI ordering and rendering stability  
#[test]
fn test_ui_rendering_consistency() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create repos in specific order to test sorting stability
    let repos = vec![
        ("zebra-repo", "zebra"),
        ("alpha-repo", "alpha"), 
        ("beta-repo", "beta"),
        ("charlie-repo", "charlie"),
    ];
    
    for (repo_name, group_name) in &repos {
        let group_dir = base_path.join(group_name);
        let repo_path = group_dir.join(repo_name);
        fs::create_dir_all(&repo_path)?;
        git2::Repository::init(&repo_path)?;
    }
    
    // Run multiple discovery + grouping cycles
    let mut all_ui_outputs = Vec::new();
    
    for cycle in 0..5 {
        let discovered_repos = yarg::scan::find_repos(base_path)?;
        let grouped_repos = yarg::scan::group_repositories(&discovered_repos);
        
        // Simulate what the UI does - convert to display text
        let mut ui_lines = Vec::new();
        for (group_name, repos_in_group) in &grouped_repos {
            ui_lines.push(format!("▼ {}", group_name));
            for repo in repos_in_group {
                ui_lines.push(format!("  {} ({})", repo.name, repo.path.display()));
            }
            ui_lines.push("".to_string());
        }
        
        println!("Cycle {}: UI output has {} lines", cycle + 1, ui_lines.len());
        if cycle == 0 {
            println!("First cycle UI lines: {:?}", ui_lines);
        }
        all_ui_outputs.push(ui_lines);
    }
    
    // All UI outputs should be identical for stability
    let first_output = &all_ui_outputs[0];
    for (cycle_idx, output) in all_ui_outputs.iter().enumerate().skip(1) {
        if first_output != output {
            println!("Cycle {} differs from first cycle!", cycle_idx + 1);
            println!("First: {:?}", first_output);
            println!("Current: {:?}", output);
        }
        assert_eq!(first_output, output, 
                  "UI output cycle {} differs from first cycle - HashMap ordering is unstable!", cycle_idx + 1);
    }
    
    Ok(())
}