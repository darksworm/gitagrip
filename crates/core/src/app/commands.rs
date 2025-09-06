use std::path::PathBuf;
use crate::domain::repo::RepoId;

/// Commands that can be sent to the application service
#[derive(Debug, Clone)]
pub enum Command {
    /// Rescan repositories in the given base directory
    Rescan { base: PathBuf },
    
    /// Refresh status for specific repositories
    RefreshStatus { ids: Vec<RepoId> },
    
    /// Fetch all repositories
    FetchAll { prune: bool },
    
    /// Open repository in external application
    OpenRepo { id: RepoId },
    
    /// Toggle group visibility/selection
    ToggleGroup { name: String },
    
    /// Show commit log for repository
    ShowLog { 
        id: RepoId, 
        range: Option<String>, 
        limit: usize 
    },
    
    /// Quit the application
    Quit,
}