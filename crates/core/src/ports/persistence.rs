use crate::domain::repo::Group;
use anyhow::Result;
use std::collections::HashMap;
use std::path::PathBuf;

/// Configuration store interface
pub trait ConfigStore: Send + Sync {
    /// Load configuration from storage
    fn load(&self) -> Result<AppConfig>;
    
    /// Save configuration to storage
    fn save(&self, config: &AppConfig) -> Result<()>;
}

/// State store interface (for caching)
pub trait StateStore: Send + Sync {
    /// Load cached state
    fn load_state(&self) -> Result<Option<CachedState>>;
    
    /// Save state to cache
    fn save_state(&self, state: &CachedState) -> Result<()>;
}

/// Application configuration
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct AppConfig {
    pub version: u32,
    pub base_dir: PathBuf,
    pub ui: UiConfig,
    pub groups: HashMap<String, Group>,
}

/// UI configuration
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UiConfig {
    pub show_ahead_behind: bool,
    pub autosave_on_exit: bool,
}

/// Cached application state
#[derive(Debug, Clone)]
pub struct CachedState {
    pub last_scan_timestamp: i64,
    // Could include cached status, etc.
}

impl Default for AppConfig {
    fn default() -> Self {
        Self {
            version: 1,
            base_dir: std::env::current_dir().unwrap_or_else(|_| PathBuf::from(".")),
            ui: UiConfig::default(),
            groups: HashMap::new(),
        }
    }
}

impl Default for UiConfig {
    fn default() -> Self {
        Self {
            show_ahead_behind: true,
            autosave_on_exit: true,
        }
    }
}