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
    
    // Test 3: Save and reload should work
    let new_config_file = temp_dir.path().join("new_config.toml");
    final_config.save(&new_config_file)?;
    
    // Verify saved config can be loaded back
    let reloaded_config = gitagrip::config::Config::load(Some(new_config_file))?;
    assert_eq!(reloaded_config.base_dir, PathBuf::from("/override/path"));
    
    // Test 4: Default config creation
    let nonexistent_file = temp_dir.path().join("nonexistent.toml");
    let default_config = gitagrip::config::Config::load(Some(nonexistent_file.clone()))?;
    
    // Should create default config
    assert_eq!(default_config.version, 1);
    assert!(nonexistent_file.exists(), "Should create default config file");
    
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

// Test CLI parsing functionality
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