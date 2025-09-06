use crate::domain::{repo::{RepoId, RepoStatus, AheadBehind}, commit::Commit};
use anyhow::Result;
use std::any::Any;

/// Port for Git operations
pub trait GitPort: Send + Sync {
    /// Get the current status of a repository
    fn status(&self, id: &RepoId) -> Result<RepoStatus>;
    
    /// Get ahead/behind counts for a branch against its upstream
    fn ahead_behind(&self, id: &RepoId, upstream: &str) -> Result<AheadBehind>;
    
    /// Fetch from remote repository
    fn fetch(&self, id: &RepoId, remote: &str, prune: bool) -> Result<()>;
    
    /// Get commit log for a repository
    fn log(&self, id: &RepoId, range: &str, limit: usize) -> Result<Vec<Commit>>;
    
    /// Downcast helper for accessing concrete implementations
    fn as_any(&self) -> &dyn Any;
}