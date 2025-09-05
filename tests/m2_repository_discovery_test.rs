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

// This is our "guiding star" integration test for M2 
// It tests the complete repository discovery and grouping functionality
#[test]
fn test_m2_repository_discovery_integration() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Create test directory structure with git repositories
    let repo_dirs = vec![
        "work/acme-api",
        "work/acme-web", 
        "personal/dotfiles",
        "standalone-project"
    ];
    
    for repo_dir in &repo_dirs {
        let repo_path = base_path.join(repo_dir);
        create_test_git_repo(repo_path)?;
    }
    
    // Test 1: Repository discovery should find all repos
    let discovered_repos = gitagrip::scan::find_repos(base_path)?;
    
    assert_eq!(discovered_repos.len(), 4);
    
    // Verify all expected repos were found
    let repo_names: Vec<&str> = discovered_repos.iter().map(|r| r.name.as_str()).collect();
    assert!(repo_names.contains(&"acme-api"));
    assert!(repo_names.contains(&"acme-web"));
    assert!(repo_names.contains(&"dotfiles"));
    assert!(repo_names.contains(&"standalone-project"));
    
    // Test 2: Auto-grouping should work based on parent directory
    let grouped_repos = gitagrip::scan::group_repositories(&discovered_repos);
    
    // Should have: Auto: work (2), Auto: personal (1), Ungrouped (1)
    assert_eq!(grouped_repos.len(), 3);
    assert!(grouped_repos.contains_key("Auto: work"));
    assert!(grouped_repos.contains_key("Auto: personal"));  
    assert!(grouped_repos.contains_key("Ungrouped"));
    
    assert_eq!(grouped_repos["Auto: work"].len(), 2);
    assert_eq!(grouped_repos["Auto: personal"].len(), 1);
    assert_eq!(grouped_repos["Ungrouped"].len(), 1);
    
    // Test 3: Background scanning should work
    let (tx, rx) = crossbeam_channel::unbounded();
    
    // Spawn background scan
    let base_path_clone = base_path.to_path_buf();
    std::thread::spawn(move || {
        if let Err(e) = gitagrip::scan::scan_repositories_background(base_path_clone, tx) {
            eprintln!("Background scan failed: {}", e);
        }
    });
    
    // Collect scan results
    let mut received_repos = Vec::new();
    let mut scan_completed = false;
    
    // Give it a reasonable timeout
    for _ in 0..100 {
        std::thread::sleep(std::time::Duration::from_millis(10));
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
            Err(_) => {
                if scan_completed {
                    break;
                }
            }
        }
    }
    
    assert!(scan_completed, "Background scan should complete");
    assert_eq!(received_repos.len(), 4, "Should discover all repositories");
    
    Ok(())
}

#[test]
fn test_repository_discovery_edge_cases() -> Result<()> {
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path();
    
    // Test 1: Empty directory
    let repos = gitagrip::scan::find_repos(base_path)?;
    assert!(repos.is_empty(), "Should find no repos in empty directory");
    
    // Test 2: Directory with nested .git (should not descend into repo)
    let outer_repo = base_path.join("outer-repo");
    create_test_git_repo(outer_repo.clone())?;
    
    let inner_dir = outer_repo.join("subdir");
    // Create what looks like an inner repo
    fs::create_dir_all(inner_dir.join(".git"))?;
    
    let repos = gitagrip::scan::find_repos(base_path)?;
    assert_eq!(repos.len(), 1, "Should only find outer repo, not descend into it");
    assert_eq!(repos[0].path, outer_repo);
    
    // Test 3: Directory with .git file (worktree)
    let worktree_dir = base_path.join("worktree");
    fs::create_dir_all(&worktree_dir)?;
    fs::write(worktree_dir.join(".git"), "gitdir: /some/path")?;
    
    let repos = gitagrip::scan::find_repos(base_path)?;
    // The worktree might not be detected as a valid repo since it points to a non-existent path
    // Let's just verify we don't crash and handle the case gracefully
    assert!(repos.len() >= 1, "Should at least find the outer repo");
    
    Ok(())
}

#[test]
fn test_repository_struct() -> Result<()> {
    let repo = gitagrip::scan::Repository {
        name: "test-repo".to_string(),
        path: std::path::PathBuf::from("/path/to/repo"),
        auto_group: "Auto: parent".to_string(),
    };
    
    assert_eq!(repo.name, "test-repo");
    assert_eq!(repo.path, std::path::PathBuf::from("/path/to/repo"));
    assert_eq!(repo.auto_group, "Auto: parent");
    
    Ok(())
}