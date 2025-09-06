//! Integration tests for the new hexagonal architecture

use anyhow::Result;
use gitagrip_core::ports::{AppConfig, UiConfig};
use tempfile::TempDir;

#[tokio::test]
async fn test_new_architecture_basic_startup() -> Result<()> {
    // Create a temporary directory for testing
    let temp_dir = TempDir::new()?;
    let base_path = temp_dir.path().to_path_buf();
    
    // Create test config
    let config = AppConfig {
        version: 1,
        base_dir: base_path.clone(),
        ui: UiConfig {
            show_ahead_behind: true,
            autosave_on_exit: true,
        },
        groups: Default::default(),
    };
    
    // Save config to temp location
    std::env::set_var("HOME", temp_dir.path());
    
    // Create the adapters
    let git_adapter = std::sync::Arc::new(gitagrip::adapters::git::GitAdapter::new());
    let discovery_adapter = std::sync::Arc::new(gitagrip::adapters::discovery::FsDiscoveryAdapter::new());
    let config_store = std::sync::Arc::new(gitagrip::adapters::persistence::FileConfigStore::new()?);
    
    // Create app service
    let (_service, _event_rx, _command_tx) = 
        gitagrip::services::app_service::AppService::new(git_adapter, discovery_adapter, config_store);
    
    // If we get here without panicking, the basic wiring works
    Ok(())
}

#[test]
fn test_new_architecture_imports() {
    // This test just verifies that all the modules are accessible
    let _ = gitagrip::adapters::git::GitAdapter::new();
    let _ = gitagrip::adapters::discovery::FsDiscoveryAdapter::new();
    let _ = gitagrip::tui::TuiModel::new();
    
    // If this compiles and runs, our module structure is correct
}