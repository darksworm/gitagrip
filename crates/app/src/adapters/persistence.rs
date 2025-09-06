use anyhow::{Context, Result};
use directories::ProjectDirs;
use gitagrip_core::ports::{AppConfig, UiConfig};
use gitagrip_core::ports::ConfigStore;
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};

/// File-based configuration store that implements ConfigStore
pub struct FileConfigStore {
    config_path: PathBuf,
}

impl FileConfigStore {
    pub fn new() -> Result<Self> {
        let config_path = Self::get_default_config_path()?;
        Ok(Self { config_path })
    }

    pub fn with_path<P: AsRef<Path>>(config_path: P) -> Self {
        Self {
            config_path: config_path.as_ref().to_path_buf(),
        }
    }

    fn get_default_config_path() -> Result<PathBuf> {
        let proj_dirs = ProjectDirs::from("", "", "gitagrip")
            .context("Failed to determine project directories")?;
        
        let config_dir = proj_dirs.config_dir();
        Ok(config_dir.join("gitagrip.toml"))
    }

    /// Create default config if it doesn't exist
    fn ensure_config_exists(&self, default_config: &AppConfig) -> Result<()> {
        if !self.config_path.exists() {
            // Create directory if it doesn't exist
            if let Some(parent) = self.config_path.parent() {
                fs::create_dir_all(parent)
                    .context("Failed to create config directory")?;
            }
            self.save(default_config)?;
        }
        Ok(())
    }
}

impl ConfigStore for FileConfigStore {
    fn load(&self) -> Result<AppConfig> {
        let default_config = AppConfig {
            version: 1,
            base_dir: dirs::home_dir().unwrap_or_else(|| PathBuf::from(".")),
            ui: UiConfig {
                show_ahead_behind: true,
                autosave_on_exit: true,
            },
            groups: HashMap::new(),
        };

        self.ensure_config_exists(&default_config)?;

        let contents = fs::read_to_string(&self.config_path)
            .with_context(|| format!("Failed to read config file: {}", self.config_path.display()))?;
        
        let config: AppConfig = toml::from_str(&contents)
            .with_context(|| format!("Failed to parse config file: {}", self.config_path.display()))?;
            
        Ok(config)
    }

    fn save(&self, config: &AppConfig) -> Result<()> {
        let contents = toml::to_string_pretty(config)
            .context("Failed to serialize config to TOML")?;
            
        fs::write(&self.config_path, contents)
            .with_context(|| format!("Failed to write config file: {}", self.config_path.display()))?;
            
        Ok(())
    }

}

impl Default for FileConfigStore {
    fn default() -> Self {
        Self::new().expect("Failed to create default config store")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    #[test]
    fn test_config_load_nonexistent_creates_default() -> Result<()> {
        let temp_dir = TempDir::new()?;
        let config_path = temp_dir.path().join("nonexistent.toml");
        
        let store = FileConfigStore::with_path(&config_path);
        let config = store.load()?;
        
        // Should create default config
        assert_eq!(config.version, 1);
        assert!(config.ui.autosave_on_exit);
        
        // Should have created the file
        assert!(config_path.exists());
        
        Ok(())
    }

    #[test]
    fn test_config_save_and_load() -> Result<()> {
        let temp_dir = TempDir::new()?;
        let config_path = temp_dir.path().join("test.toml");
        
        let store = FileConfigStore::with_path(&config_path);
        
        let config = AppConfig {
            version: 1,
            base_dir: PathBuf::from("/custom/path"),
            ui: UiConfig {
                show_ahead_behind: false,
                autosave_on_exit: true,
            },
            groups: HashMap::new(), // Simplified for now
        };
        
        store.save(&config)?;
        let loaded_config = store.load()?;
        
        assert_eq!(config.base_dir, loaded_config.base_dir);
        assert_eq!(config.ui.show_ahead_behind, loaded_config.ui.show_ahead_behind);
        assert_eq!(config.groups.len(), loaded_config.groups.len());
        
        Ok(())
    }


    #[test]
    fn test_get_default_config_path() -> Result<()> {
        let path = FileConfigStore::get_default_config_path()?;
        assert!(path.ends_with("gitagrip.toml"));
        Ok(())
    }

}