use clap::Parser;
use std::path::PathBuf;

#[derive(Parser, Debug, PartialEq)]
#[command(name = "yarg")]
#[command(about = "Yet Another Repo Grouper - A fast TUI for managing multiple Git repositories")]
pub struct CliArgs {
    /// Base directory to scan for repositories (overrides config)
    #[arg(long)]
    pub base_dir: Option<PathBuf>,
    
    /// Path to configuration file
    #[arg(long)]
    pub config: Option<PathBuf>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_cli_parse_base_dir_only() {
        let args = CliArgs::parse_from(&["yarg", "--base-dir", "/test/path"]);
        assert_eq!(args.base_dir, Some(PathBuf::from("/test/path")));
        assert_eq!(args.config, None);
    }
    
    #[test]
    fn test_cli_parse_with_config() {
        let args = CliArgs::parse_from(&[
            "yarg",
            "--base-dir", "/test/path", 
            "--config", "/custom/config.toml"
        ]);
        assert_eq!(args.base_dir, Some(PathBuf::from("/test/path")));
        assert_eq!(args.config, Some(PathBuf::from("/custom/config.toml")));
    }
    
    #[test]
    fn test_cli_parse_no_args() {
        let args = CliArgs::parse_from(&["yarg"]);
        assert_eq!(args.base_dir, None);
        assert_eq!(args.config, None);
    }
}