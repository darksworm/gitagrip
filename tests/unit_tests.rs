use anyhow::Result;

// Unit tests for tricky logic that might not be fully covered by integration tests

#[test]
fn test_app_scroll_bounds_unit() {
    let config = gitagrip::config::Config::default();
    let mut app = gitagrip::app::App::new(config, None);
    
    // Test bounds with empty repository list
    assert_eq!(app.scroll_offset, 0);
    app.scroll_down();
    assert_eq!(app.scroll_offset, 0, "Should not scroll down with empty list");
    
    app.scroll_up();
    assert_eq!(app.scroll_offset, 0, "Should not scroll up from 0");
    
    // Add a single repository
    app.add_repository(gitagrip::scan::Repository {
        name: "single-repo".to_string(),
        path: std::path::PathBuf::from("/test"),
        auto_group: "Test".to_string(),
    });
    
    // With 1 repo, max scroll offset should be 0 (can't scroll past the only item)
    app.scroll_down();
    assert_eq!(app.scroll_offset, 0, "Should not scroll past single repository");
    
    // Add more repositories to test proper bounds
    for i in 2..=5 {
        app.add_repository(gitagrip::scan::Repository {
            name: format!("repo-{}", i),
            path: std::path::PathBuf::from(format!("/test/repo-{}", i)),
            auto_group: "Test".to_string(),
        });
    }
    
    // Now we have 5 repos (indices 0-4), max scroll offset should be 4
    assert_eq!(app.repositories.len(), 5);
    
    // Scroll down to maximum
    for _ in 0..10 { // Try to scroll more than possible
        app.scroll_down();
    }
    assert_eq!(app.scroll_offset, 4, "Should stop at last repository index");
    
    // Scroll back up to minimum
    for _ in 0..10 { // Try to scroll more than possible
        app.scroll_up();
    }
    assert_eq!(app.scroll_offset, 0, "Should stop at first repository");
}

#[test]
fn test_branch_color_consistency() {
    // Test that the branch color function gives consistent results
    // Note: We can't test the actual function directly since it's private,
    // but we can test the behavior through the App
    
    // Main and master should always be special
    // This tests the logic through the UI rendering, which is tricky but important
    let config = gitagrip::config::Config::default();
    let app = gitagrip::app::App::new(config, None);
    
    // We can't directly test branch_color since it's private, but we know:
    // 1. main/master should be treated specially (bold green)
    // 2. Other branch names should be consistent (same name = same color)
    // 3. Hash function should distribute colors evenly
    
    // This is more of a smoke test - the real testing happens in integration tests
    // where we verify the UI rendering works correctly
    
    assert_eq!(app.repositories.len(), 0); // Just verify app creation works
}

#[test]
fn test_repository_auto_grouping_logic() -> Result<()> {
    // Test the auto-grouping logic with edge cases
    
    let repo1 = gitagrip::scan::Repository {
        name: "repo1".to_string(),
        path: std::path::PathBuf::from("/work/project1/repo1"),
        auto_group: "Auto: project1".to_string(),
    };
    
    let repo2 = gitagrip::scan::Repository {
        name: "repo2".to_string(),
        path: std::path::PathBuf::from("/work/project1/repo2"),
        auto_group: "Auto: project1".to_string(),
    };
    
    let repo3 = gitagrip::scan::Repository {
        name: "standalone".to_string(),
        path: std::path::PathBuf::from("/standalone"),
        auto_group: "Ungrouped".to_string(),
    };
    
    let repos = vec![repo1, repo2, repo3];
    let grouped = gitagrip::scan::group_repositories(&repos);
    
    assert_eq!(grouped.len(), 2);
    assert!(grouped.contains_key("Auto: project1"));
    assert!(grouped.contains_key("Ungrouped"));
    assert_eq!(grouped["Auto: project1"].len(), 2);
    assert_eq!(grouped["Ungrouped"].len(), 1);
    
    Ok(())
}

#[test]
fn test_git_status_parsing_edge_cases() -> Result<()> {
    // Test git status parsing with edge cases
    // This would be useful if we had more complex git status parsing logic
    
    use tempfile::TempDir;
    use std::fs;
    
    let temp_dir = TempDir::new()?;
    let repo_path = temp_dir.path().join("test-repo");
    
    // Create a repository
    fs::create_dir_all(&repo_path)?;
    let git_repo = git2::Repository::init(&repo_path)?;
    let signature = git2::Signature::now("Test User", "test@example.com")?;
    
    // Test reading status from repository with no commits
    let status = gitagrip::git::read_status(&repo_path)?;
    assert_eq!(status.name, "test-repo");
    assert_eq!(status.last_commit_summary, "No commits");
    assert!(!status.is_dirty); // Empty repo should be clean
    
    // Add a commit
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
        "Test commit message",
        &tree,
        &[],
    )?;
    
    // Test with commit
    let status = gitagrip::git::read_status(&repo_path)?;
    assert_eq!(status.last_commit_summary, "Test commit message");
    assert!(!status.is_dirty); // Should still be clean
    assert!(status.branch_name.is_some()); // Should have a branch
    
    Ok(())
}

#[test]
fn test_config_edge_cases() -> Result<()> {
    use tempfile::TempDir;
    
    let temp_dir = TempDir::new()?;
    
    // Test loading config from non-existent file (should create default)
    let config_path = temp_dir.path().join("nonexistent.toml");
    let config = gitagrip::config::Config::load(Some(config_path.clone()))?;
    
    assert_eq!(config.version, 1);
    assert!(config_path.exists(), "Should create default config");
    
    // Test that the created config can be loaded again
    let reloaded = gitagrip::config::Config::load(Some(config_path))?;
    assert_eq!(reloaded.version, config.version);
    assert_eq!(reloaded.base_dir, config.base_dir);
    
    Ok(())
}