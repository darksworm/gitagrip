use anyhow::{Context, Result};
use directories::ProjectDirs;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

use crate::cli::CliArgs;

#[derive(Debug, Serialize, Deserialize, PartialEq, Clone)]
pub struct Config {
    pub version: u32,
    pub base_dir: PathBuf,
    pub ui: UiConfig,
    #[serde(default)]
    pub groups: HashMap<String, GroupConfig>,
}

#[derive(Debug, Serialize, Deserialize, PartialEq, Clone)]
pub struct UiConfig {
    pub show_ahead_behind: bool,
    pub autosave_on_exit: bool,
}

#[derive(Debug, Serialize, Deserialize, PartialEq, Clone)]
pub struct GroupConfig {
    pub repos: Vec<PathBuf>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            version: 1,
            base_dir: dirs::home_dir().unwrap_or_else(|| PathBuf::from(".")),
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

pub fn get_default_config_path() -> Result<PathBuf> {
    let proj_dirs = ProjectDirs::from("", "", "gitagrip")
        .context("Failed to determine project directories")?;
    
    let config_dir = proj_dirs.config_dir();
    Ok(config_dir.join("gitagrip.toml"))
}

impl Config {
    pub fn load(config_path: Option<PathBuf>) -> Result<Self> {
        let path = match config_path {
            Some(p) => p,
            None => get_default_config_path()?,
        };

        if !path.exists() {
            let default_config = Config::default();
            // Create directory if it doesn't exist
            if let Some(parent) = path.parent() {
                fs::create_dir_all(parent)
                    .context("Failed to create config directory")?;
            }
            default_config.save(&path)?;
            return Ok(default_config);
        }

        let contents = fs::read_to_string(&path)
            .with_context(|| format!("Failed to read config file: {}", path.display()))?;
        
        let config: Config = toml::from_str(&contents)
            .with_context(|| format!("Failed to parse config file: {}", path.display()))?;
            
        Ok(config)
    }

    pub fn save<P: AsRef<std::path::Path>>(&self, path: P) -> Result<()> {
        let contents = toml::to_string_pretty(self)
            .context("Failed to serialize config to TOML")?;
            
        fs::write(&path, contents)
            .with_context(|| format!("Failed to write config file: {}", path.as_ref().display()))?;
            
        Ok(())
    }

    pub fn from_cli_and_file(cli_args: CliArgs, config_path: Option<PathBuf>) -> Result<Self> {
        let mut config = Self::load(config_path)?;
        
        // CLI args override config file
        if let Some(base_dir) = cli_args.base_dir {
            config.base_dir = base_dir;
        }
        
        Ok(config)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    #[test]
    fn test_config_default() {
        let config = Config::default();
        assert_eq!(config.version, 1);
        assert!(config.ui.show_ahead_behind);
        assert!(config.ui.autosave_on_exit);
        assert!(config.groups.is_empty());
        assert!(!config.base_dir.as_os_str().is_empty());
    }

    #[test]
    fn test_config_serialization_roundtrip() -> Result<()> {
        let mut config = Config::default();
        config.base_dir = PathBuf::from("/test/path");
        config.ui.show_ahead_behind = false;
        
        let mut work_group = GroupConfig { repos: vec![] };
        work_group.repos.push(PathBuf::from("/repo1"));
        work_group.repos.push(PathBuf::from("/repo2"));
        config.groups.insert("Work".to_string(), work_group);

        let toml_str = toml::to_string(&config)?;
        let parsed_config: Config = toml::from_str(&toml_str)?;
        
        assert_eq!(config, parsed_config);
        Ok(())
    }

    #[test]
    fn test_config_load_nonexistent_creates_default() -> Result<()> {
        let temp_dir = TempDir::new()?;
        let config_path = temp_dir.path().join("nonexistent.toml");
        
        let config = Config::load(Some(config_path.clone()))?;
        
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
        
        let mut config = Config::default();
        config.base_dir = PathBuf::from("/custom/path");
        config.ui.show_ahead_behind = false;
        
        config.save(&config_path)?;
        let loaded_config = Config::load(Some(config_path))?;
        
        assert_eq!(config.base_dir, loaded_config.base_dir);
        assert_eq!(config.ui.show_ahead_behind, loaded_config.ui.show_ahead_behind);
        
        Ok(())
    }

    #[test]
    fn test_cli_override() -> Result<()> {
        let cli_args = CliArgs {
            base_dir: Some(PathBuf::from("/override/path")),
            config: None,
        };
        
        let temp_dir = TempDir::new()?;
        let config_path = temp_dir.path().join("test.toml");
        
        // Create a config file with different base_dir
        let original_config = Config {
            base_dir: PathBuf::from("/original/path"),
            ..Config::default()
        };
        original_config.save(&config_path)?;
        
        // CLI should override
        let final_config = Config::from_cli_and_file(cli_args, Some(config_path))?;
        assert_eq!(final_config.base_dir, PathBuf::from("/override/path"));
        
        Ok(())
    }

    #[test]
    fn test_get_default_config_path() -> Result<()> {
        let path = get_default_config_path()?;
        assert!(path.ends_with("gitagrip.toml"));
        Ok(())
    }
}