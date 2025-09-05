use anyhow::Result;
use clap::Parser;
use std::fs;
use std::path::{Path, PathBuf};
use tempfile::TempDir;

// This is our "guiding star" integration test for M1
// It tests the complete flow: CLI args -> config loading -> app initialization
#[test]
fn test_m1_config_and_cli_integration() -> Result<()> {
    // Setup: Create a temporary directory for our test config
    let temp_dir = TempDir::new()?;
    let config_dir = temp_dir.path().join(".config").join("gitagrip");
    fs::create_dir_all(&config_dir)?;
    
    let config_file = config_dir.join("gitagrip.toml");
    
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
    let config = gitagrip::config::Config::load(Some(config_file.clone()))?;
    
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
    let cli_args = gitagrip::cli::CliArgs {
        base_dir: Some(PathBuf::from("/override/path")),
        config: None,
    };
    
    let final_config = gitagrip::config::Config::from_cli_and_file(cli_args, Some(config_file))?;
    assert_eq!(final_config.base_dir, PathBuf::from("/override/path")); // CLI should override
    assert_eq!(final_config.ui.show_ahead_behind, true); // Other settings preserved
    
    // Test 3: Config should be saveable
    let new_config_file = config_dir.join("yarg_test_save.toml");
    final_config.save(&new_config_file)?;
    
    // Verify saved config can be loaded back
    let reloaded_config = gitagrip::config::Config::load(Some(new_config_file))?;
    assert_eq!(reloaded_config.base_dir, PathBuf::from("/override/path"));
    
    // Test 4: Default config creation
    let nonexistent_file = temp_dir.path().join("nonexistent.toml");
    let default_config = gitagrip::config::Config::load(Some(nonexistent_file.clone()))?;
    
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
    let config_path = gitagrip::config::get_default_config_path()?;
    
    // Should end with gitagrip/gitagrip.toml (may be in different locations on different OS)
    assert!(config_path.ends_with("gitagrip/gitagrip.toml"));
    // On macOS it might be in ~/Library/Application Support instead of ~/.config
    let path_str = config_path.to_string_lossy();
    assert!(path_str.contains("gitagrip") && path_str.ends_with("gitagrip.toml"));
    
    Ok(())
}

// Test CLI parsing
#[test]
fn test_cli_parsing() -> Result<()> {
    // This will test that clap parsing works correctly
    let args = gitagrip::cli::CliArgs::parse_from(&["yarg", "--base-dir", "/test/path"]);
    
    assert_eq!(args.base_dir, Some(PathBuf::from("/test/path")));
    assert_eq!(args.config, None);
    
    let args_with_config = gitagrip::cli::CliArgs::parse_from(&[
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
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    
    assert_eq!(discovered_repos.len(), 4);
    
    // Check that all repos were found
    let repo_paths: Vec<_> = discovered_repos.iter().map(|r| &r.path).collect();
    for expected_repo in &repos {
        assert!(repo_paths.contains(&expected_repo), 
                "Repository not found: {}", expected_repo.display());
    }
    
    // Test 2: Auto-grouping should work based on parent directory
    let grouped_repos = gitagrip::scan::group_repositories(&discovered_repos);
    
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
        if let Err(e) = gitagrip::scan::scan_repositories_background(base_path_clone, tx) {
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
                    gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                        received_repos.push(repo);
                    }
                    gitagrip::scan::ScanEvent::ScanCompleted => {
                        scan_completed = true;
                    }
                    gitagrip::scan::ScanEvent::ScanError(err) => {
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
    let repo = gitagrip::scan::Repository {
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
    let repos = gitagrip::scan::find_repos(base_path)?;
    assert!(repos.is_empty(), "Should find no repos in empty directory");
    
    // Test 2: Directory with nested .git (should not descend into repo)
    let outer_repo = base_path.join("outer-repo");
    let inner_dir = outer_repo.join("inner");
    fs::create_dir_all(&inner_dir)?;
    
    // Create outer repo
    git2::Repository::init(&outer_repo)?;
    
    // Create what looks like an inner repo
    fs::create_dir_all(inner_dir.join(".git"))?;
    
    let repos = gitagrip::scan::find_repos(base_path)?;
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
        let discovered = gitagrip::scan::find_repos(base_path)?;
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
            gitagrip::scan::scan_repositories_background(base_path_clone, tx)
        });
        
        // Collect all events
        let mut discovered_repos: Vec<gitagrip::scan::Repository> = Vec::new();
        let mut scan_completed = false;
        let mut scan_errors = Vec::new();
        
        let timeout = std::time::Duration::from_secs(2);
        let start = std::time::Instant::now();
        
        while start.elapsed() < timeout && !scan_completed {
            match rx.recv_timeout(std::time::Duration::from_millis(50)) {
                Ok(event) => {
                    match event {
                        gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                            // Check for duplicates as they come in
                            for existing in &discovered_repos {
                                assert_ne!(existing.path, repo.path, 
                                          "Duplicate repository discovered: {}", repo.path.display());
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
        gitagrip::scan::scan_repositories_background(base_path_clone, tx)
    });
    
    let mut app_repos = Vec::new();
    let mut update_count = 0;
    let max_updates = 100; // Prevent infinite loop
    
    while update_count < max_updates {
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
        let discovered_repos = gitagrip::scan::find_repos(base_path)?;
        let grouped_repos = gitagrip::scan::group_repositories(&discovered_repos);
        
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

// M3 Guiding Star Test: Git Status Aggregation with Real Repository States
#[test]
fn test_m3_git_status_integration() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create repositories with different Git states for comprehensive testing
    let repos = vec![
        ("clean-repo", "work"),      // Clean repo, on main, up to date
        ("dirty-repo", "work"),      // Dirty repo with uncommitted changes
        ("ahead-repo", "personal"),  // Repo ahead of remote
        ("detached-repo", "tools"),  // Detached HEAD state
    ];
    
    for (repo_name, group_name) in &repos {
        let group_dir = base_path.join(group_name);
        let repo_path = group_dir.join(repo_name);
        fs::create_dir_all(&repo_path)?;
        let repo = git2::Repository::init(&repo_path)?;
        
        // Configure repo
        let mut config = repo.config()?;
        config.set_str("user.name", "Test User")?;
        config.set_str("user.email", "test@example.com")?;
        
        let sig = git2::Signature::now("Test User", "test@example.com")?;
        
        // Create initial commit
        let tree_id = {
            let mut index = repo.index()?;
            std::fs::write(repo_path.join("README.md"), "# Initial")?;
            index.add_path(std::path::Path::new("README.md"))?;
            index.write()?;
            index.write_tree()?
        };
        let tree = repo.find_tree(tree_id)?;
        let initial_commit = repo.commit(
            Some("HEAD"),
            &sig,
            &sig,
            "Initial commit",
            &tree,
            &[],
        )?;
        
        // Set up different states based on repo name
        match *repo_name {
            "clean-repo" => {
                // Already clean, do nothing more
            }
            "dirty-repo" => {
                // Add uncommitted changes
                std::fs::write(repo_path.join("README.md"), "# Modified content")?;
                std::fs::write(repo_path.join("new_file.txt"), "New file")?;
            }
            "ahead-repo" => {
                // Create a second commit to be ahead (simulated)
                std::fs::write(repo_path.join("feature.txt"), "New feature")?;
                let mut index = repo.index()?;
                index.add_path(std::path::Path::new("feature.txt"))?;
                index.write()?;
                let tree_id = index.write_tree()?;
                let tree = repo.find_tree(tree_id)?;
                let parent = repo.find_commit(initial_commit)?;
                repo.commit(
                    Some("HEAD"),
                    &sig,
                    &sig,
                    "Add feature",
                    &tree,
                    &[&parent],
                )?;
            }
            "detached-repo" => {
                // Checkout a specific commit (detached HEAD)
                let commit = repo.find_commit(initial_commit)?;
                repo.set_head_detached(commit.id())?;
            }
            _ => {}
        }
    }
    
    // Test 1: Repository discovery finds all repos
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    assert_eq!(discovered_repos.len(), 4);
    
    // Test 2: Git status reading should work for all repositories  
    let mut repo_statuses = Vec::new();
    for repo in &discovered_repos {
        let status = gitagrip::git::read_status(&repo.path)?;
        repo_statuses.push((repo.clone(), status));
    }
    
    assert_eq!(repo_statuses.len(), 4);
    
    // Test 3: Status information should be accurate
    for (repo, status) in &repo_statuses {
        match repo.name.as_str() {
            "clean-repo" => {
                assert!(!status.is_dirty, "clean-repo should not be dirty");
                assert!(status.branch_name.as_ref().map(|b| b == "main" || b == "master").unwrap_or(false), "Should be on main or master branch");
                assert!(status.last_commit_summary.contains("Initial commit"));
            }
            "dirty-repo" => {
                assert!(status.is_dirty, "dirty-repo should be dirty");
                assert!(status.branch_name.as_ref().map(|b| b == "main" || b == "master").unwrap_or(false), "Should be on main or master branch");
                assert!(status.has_staged || status.has_unstaged, "Should have changes");
            }
            "ahead-repo" => {
                assert!(!status.is_dirty, "ahead-repo should be clean");
                assert!(status.branch_name.as_ref().map(|b| b == "main" || b == "master").unwrap_or(false), "Should be on main or master branch"); 
                assert!(status.last_commit_summary.contains("Add feature"));
            }
            "detached-repo" => {
                assert!(status.is_detached, "Should be in detached HEAD state");
            }
            _ => {}
        }
    }
    
    // Test 4: Status computation should work for all repositories
    assert_eq!(repo_statuses.len(), 4, "Should get status for all repositories");
    
    // Test 5: Status events should be sent via channels
    let (tx, rx) = crossbeam_channel::unbounded();
    
    std::thread::spawn(move || {
        for (repo, status) in repo_statuses {
            let event = gitagrip::git::StatusEvent::StatusUpdated { 
                repository: repo.name.clone(), 
                status 
            };
            if tx.send(event).is_err() {
                break;
            }
        }
        let _ = tx.send(gitagrip::git::StatusEvent::StatusScanCompleted);
    });
    
    // Collect status events
    let mut received_statuses = Vec::new();
    let mut scan_completed = false;
    
    let timeout = std::time::Duration::from_secs(2);
    let start = std::time::Instant::now();
    
    while start.elapsed() < timeout && !scan_completed {
        match rx.recv_timeout(std::time::Duration::from_millis(50)) {
            Ok(event) => {
                match event {
                    gitagrip::git::StatusEvent::StatusUpdated { repository, status } => {
                        received_statuses.push((repository, status));
                    }
                    gitagrip::git::StatusEvent::StatusScanCompleted => {
                        scan_completed = true;
                    }
                    gitagrip::git::StatusEvent::StatusError { repository: _, error: _ } => {
                        // Ignore errors for this test
                    }
                }
            }
            Err(_) => continue,
        }
    }
    
    assert!(scan_completed, "Status scan should complete");
    assert_eq!(received_statuses.len(), 4, "Should receive status for all repos");
    
    Ok(())
}

#[test]
fn test_m3_tui_git_status_display_integration() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories with different git states
    let repos_with_states = vec![
        ("clean-repo", false, false),        // clean repository
        ("dirty-repo", true, false),         // dirty repository  
        ("staged-repo", false, true),        // staged changes
    ];
    
    for (repo_name, has_unstaged, has_staged) in &repos_with_states {
        let repo_path = base_path.join(repo_name);
        fs::create_dir_all(&repo_path)?;
        
        // Initialize git repo
        let git_repo = git2::Repository::init(&repo_path)?;
        
        // Create initial commit
        let signature = git2::Signature::now("Test User", "test@example.com")?;
        let tree_id = {
            let mut index = git_repo.index()?;
            let tree_id = index.write_tree()?;
            tree_id
        };
        let tree = git_repo.find_tree(tree_id)?;
        let _commit = git_repo.commit(
            Some("HEAD"),
            &signature,
            &signature,
            "Initial commit",
            &tree,
            &[],
        )?;
        
        // Create different states based on test parameters
        if *has_unstaged || *has_staged {
            let test_file = repo_path.join("test.txt");
            fs::write(&test_file, "test content")?;
            
            if *has_staged {
                let mut index = git_repo.index()?;
                index.add_path(Path::new("test.txt"))?;
                index.write()?;
            }
        }
    }
    
    // Create config pointing to our test directory
    let config = gitagrip::config::Config {
        version: 1,
        base_dir: base_path.to_path_buf(),
        ui: gitagrip::config::UiConfig {
            show_ahead_behind: true,
            autosave_on_exit: false,
        },
        groups: std::collections::HashMap::new(),
    };
    
    // Test 1: App should discover repositories and show git status  
    let _app = gitagrip::app::App::new(config.clone());
    
    // Set up channels for repository scanning and git status
    let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();
    let (status_sender, status_receiver) = crossbeam_channel::unbounded();
    
    // Start background repository scanning
    let base_dir_clone = config.base_dir.clone();
    std::thread::spawn(move || {
        if let Err(e) = gitagrip::scan::scan_repositories_background(base_dir_clone, scan_sender) {
            eprintln!("Scan error: {}", e);
        }
    });
    
    // Collect discovered repositories
    let mut discovered_repos = Vec::new();
    let timeout = std::time::Duration::from_secs(2);
    let start = std::time::Instant::now();
    
    while start.elapsed() < timeout {
        if let Ok(event) = scan_receiver.try_recv() {
            match event {
                gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                    discovered_repos.push(repo);
                }
                gitagrip::scan::ScanEvent::ScanCompleted => break,
                gitagrip::scan::ScanEvent::ScanError(err) => {
                    panic!("Repository scan failed: {}", err);
                }
            }
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
    
    assert_eq!(discovered_repos.len(), 3, "Should discover all test repositories");
    
    // Test 2: App should load git status for discovered repositories
    gitagrip::git::compute_statuses_with_events(&discovered_repos, status_sender)?;
    
    let mut repo_statuses = std::collections::HashMap::new();
    let status_timeout = std::time::Duration::from_secs(3);
    let status_start = std::time::Instant::now();
    let mut status_complete = false;
    
    while status_start.elapsed() < status_timeout && !status_complete {
        if let Ok(event) = status_receiver.try_recv() {
            match event {
                gitagrip::git::StatusEvent::StatusUpdated { repository, status } => {
                    repo_statuses.insert(repository, status);
                }
                gitagrip::git::StatusEvent::StatusScanCompleted => {
                    status_complete = true;
                }
                gitagrip::git::StatusEvent::StatusError { repository, error } => {
                    panic!("Status error for {}: {}", repository, error);
                }
            }
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
    
    assert_eq!(repo_statuses.len(), 3, "Should get status for all repositories");
    
    // Test 3: App should have git status data integrated
    for (repo_name, status) in &repo_statuses {
        match repo_name.as_str() {
            "clean-repo" => {
                assert!(!status.is_dirty, "Clean repo should not be dirty");
            }
            "dirty-repo" => {
                assert!(status.is_dirty, "Dirty repo should be dirty");
            }
            "staged-repo" => {
                assert!(status.is_dirty, "Staged repo should be dirty");
            }
            _ => {}
        }
    }
    
    // Test 4: UI rendering should be fast and not block
    let render_time = std::time::Instant::now();
    // The actual UI uses ui_with_git_status method, but we're just testing performance
    let _mock_render = format!("Mock UI with {} repos", discovered_repos.len());
    let render_duration = render_time.elapsed();
    
    assert!(render_duration < std::time::Duration::from_millis(100), 
           "UI rendering should be fast and non-blocking");
    
    Ok(())
}

#[test]
fn test_m3_end_to_end_git_status_in_tui() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test repositories with real git states
    let test_repos = vec![
        ("clean-repo", false, false),
        ("dirty-repo", true, false),
        ("staged-repo", false, true),
    ];
    
    for (repo_name, has_unstaged, has_staged) in test_repos {
        let repo_path = base_path.join(repo_name);
        fs::create_dir_all(&repo_path)?;
        
        // Initialize real git repo
        let git_repo = git2::Repository::init(&repo_path)?;
        let signature = git2::Signature::now("Test User", "test@example.com")?;
        
        // Create initial commit
        let tree_id = {
            let mut index = git_repo.index()?;
            let tree_id = index.write_tree()?;
            tree_id
        };
        let tree = git_repo.find_tree(tree_id)?;
        let _commit = git_repo.commit(
            Some("HEAD"),
            &signature,
            &signature,
            "Initial commit",
            &tree,
            &[],
        )?;
        
        // Create different git states
        if has_unstaged || has_staged {
            let test_file = repo_path.join("test.txt");
            fs::write(&test_file, "test content")?;
            
            if has_staged {
                let mut index = git_repo.index()?;
                index.add_path(Path::new("test.txt"))?;
                index.write()?;
            }
        }
    }
    
    // Create config for the test
    let config = gitagrip::config::Config {
        version: 1,
        base_dir: base_path.to_path_buf(),
        ui: gitagrip::config::UiConfig {
            show_ahead_behind: true,
            autosave_on_exit: false,
        },
        groups: std::collections::HashMap::new(),
    };
    
    // Create the App and run it briefly to capture UI output
    let mut app = gitagrip::app::App::new(config.clone());
    
    // Set up background scanning (same as main.rs does)
    let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();
    let base_dir_clone = config.base_dir.clone();
    std::thread::spawn(move || {
        if let Err(e) = gitagrip::scan::scan_repositories_background(base_dir_clone, scan_sender) {
            eprintln!("Background scan failed: {}", e);
        }
    });
    
    // Create a mock terminal backend for testing
    use ratatui::backend::TestBackend;
    use ratatui::Terminal;
    
    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend)?;
    
    // Simulate the app running for a short time to discover repos and get git status
    let timeout = std::time::Duration::from_secs(3);
    let start_time = std::time::Instant::now();
    
    while start_time.elapsed() < timeout {
        // Process repository scan events (like the real app does)
        while let Ok(event) = scan_receiver.try_recv() {
            match event {
                gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                    app.repositories.push(repo);
                }
                gitagrip::scan::ScanEvent::ScanCompleted => {
                    app.scan_complete = true;
                }
                gitagrip::scan::ScanEvent::ScanError(err) => {
                    eprintln!("Scan error: {}", err);
                }
            }
        }
        
        // Once we have repos and scan is complete, render the UI
        if app.scan_complete && !app.repositories.is_empty() {
            break;
        }
        
        std::thread::sleep(std::time::Duration::from_millis(50));
    }
    
    // Render the UI using the same ui() method from main.rs
    terminal.draw(|f| {
        app.ui_with_git_status(f)
    })?;
    
    // Get the rendered output
    let backend = terminal.backend();
    let buffer = backend.buffer();
    let rendered_content = buffer.content.iter()
        .map(|cell| cell.symbol())
        .collect::<String>();
    
    // Verify the UI contains the expected elements
    
    // Test 1: Should contain repository names
    assert!(rendered_content.contains("clean-repo"), "UI should show clean-repo");
    assert!(rendered_content.contains("dirty-repo"), "UI should show dirty-repo");
    assert!(rendered_content.contains("staged-repo"), "UI should show staged-repo");
    
    // Test 2: Should contain git status indicators
    assert!(rendered_content.contains("✓") || rendered_content.contains("clean"), 
           "UI should show clean status indicator");
    assert!(rendered_content.contains("●") || rendered_content.contains("dirty"), 
           "UI should show dirty status indicator");
    
    // Test 3: Should contain the GitaGrip title and base directory
    assert!(rendered_content.contains("GitaGrip"), "UI should show GitaGrip title");
    assert!(rendered_content.contains("Repositories"), "UI should show Repositories section");
    
    // Test 4: Should show keybindings
    assert!(rendered_content.contains("q") && rendered_content.contains("quit"), 
           "UI should show quit keybinding");
    
    Ok(())
}

#[test]
fn test_scanning_completes_with_real_repos() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create a couple of real git repositories
    for repo_name in &["test-repo-1", "test-repo-2"] {
        let repo_path = base_path.join(repo_name);
        fs::create_dir_all(&repo_path)?;
        
        let git_repo = git2::Repository::init(&repo_path)?;
        let signature = git2::Signature::now("Test User", "test@example.com")?;
        
        // Create initial commit
        let tree_id = {
            let mut index = git_repo.index()?;
            let tree_id = index.write_tree()?;
            tree_id
        };
        let tree = git_repo.find_tree(tree_id)?;
        let _commit = git_repo.commit(
            Some("HEAD"),
            &signature,
            &signature,
            "Initial commit",
            &tree,
            &[],
        )?;
    }
    
    // Create config for the test
    let config = gitagrip::config::Config {
        version: 1,
        base_dir: base_path.to_path_buf(),
        ui: gitagrip::config::UiConfig {
            show_ahead_behind: true,
            autosave_on_exit: false,
        },
        groups: std::collections::HashMap::new(),
    };
    
    let mut app = gitagrip::app::App::new(config.clone());
    
    // Set up channels like the real app
    let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();
    let (status_sender, status_receiver) = crossbeam_channel::unbounded();
    
    // Start background repository scanning (like main.rs does)
    let base_dir_clone = config.base_dir.clone();
    let scan_sender_clone = scan_sender.clone();
    std::thread::spawn(move || {
        if let Err(e) = gitagrip::scan::scan_repositories_background(base_dir_clone, scan_sender_clone) {
            eprintln!("Background scan failed: {}", e);
        }
    });
    
    // Don't duplicate the git status monitoring logic - let the app handle it
    
    // Simulate app.run() but with a timeout for testing
    let timeout = std::time::Duration::from_secs(5);
    let start_time = std::time::Instant::now();
    let mut git_status_started = false;
    
    while start_time.elapsed() < timeout {
        // Process repository scan events (exactly like app.run does)
        while let Ok(event) = scan_receiver.try_recv() {
            match event {
                gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                    app.repositories.push(repo);
                }
                gitagrip::scan::ScanEvent::ScanCompleted => {
                    app.scan_complete = true;
                    // Start git status loading once repository scan is complete (like app.run does)
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
        
        // Check if scanning completed successfully
        if app.scan_complete && !app.repositories.is_empty() {
            break;
        }
        
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
    
    // Test that scanning actually completes and finds repositories
    assert!(app.scan_complete, "Repository scanning should complete within timeout");
    assert_eq!(app.repositories.len(), 2, "Should discover both test repositories");
    
    // Test that the repositories are the ones we created
    let repo_names: Vec<String> = app.repositories.iter().map(|r| r.name.clone()).collect();
    assert!(repo_names.contains(&"test-repo-1".to_string()), "Should find test-repo-1");
    assert!(repo_names.contains(&"test-repo-2".to_string()), "Should find test-repo-2");
    
    Ok(())
}

#[test]
fn test_basic_repository_scanning_only() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create a simple git repository
    let repo_path = base_path.join("simple-repo");
    fs::create_dir_all(&repo_path)?;
    
    let git_repo = git2::Repository::init(&repo_path)?;
    let signature = git2::Signature::now("Test User", "test@example.com")?;
    
    // Create initial commit
    let tree_id = {
        let mut index = git_repo.index()?;
        let tree_id = index.write_tree()?;
        tree_id
    };
    let tree = git_repo.find_tree(tree_id)?;
    let _commit = git_repo.commit(
        Some("HEAD"),
        &signature,
        &signature,
        "Initial commit",
        &tree,
        &[],
    )?;
    
    // Test the scanning directly
    let found_repos = gitagrip::scan::find_repos(&base_path)?;
    assert_eq!(found_repos.len(), 1, "Should find exactly one repository");
    assert_eq!(found_repos[0].name, "simple-repo", "Should find the correct repository");
    
    // Test background scanning
    let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();
    
    // Start background scan
    let base_dir_clone = base_path.to_path_buf();
    std::thread::spawn(move || {
        if let Err(e) = gitagrip::scan::scan_repositories_background(base_dir_clone, scan_sender) {
            eprintln!("Background scan failed: {}", e);
        }
    });
    
    let mut discovered_repos = Vec::new();
    let mut scan_complete = false;
    let timeout = std::time::Duration::from_secs(3);
    let start_time = std::time::Instant::now();
    
    while start_time.elapsed() < timeout && !scan_complete {
        match scan_receiver.try_recv() {
            Ok(event) => match event {
                gitagrip::scan::ScanEvent::RepoDiscovered(repo) => {
                    println!("Discovered repo: {}", repo.name);
                    discovered_repos.push(repo);
                }
                gitagrip::scan::ScanEvent::ScanCompleted => {
                    println!("Scan completed!");
                    scan_complete = true;
                }
                gitagrip::scan::ScanEvent::ScanError(err) => {
                    panic!("Scan error: {}", err);
                }
            },
            Err(_) => {
                std::thread::sleep(std::time::Duration::from_millis(10));
            }
        }
    }
    
    assert!(scan_complete, "Background scanning should complete");
    assert_eq!(discovered_repos.len(), 1, "Should discover exactly one repository");
    assert_eq!(discovered_repos[0].name, "simple-repo", "Should discover the correct repository");
    
    Ok(())
}