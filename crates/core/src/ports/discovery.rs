use crate::domain::repo::{RepoId, RepoMeta};
use anyhow::Result;
use std::path::PathBuf;

/// Request for repository discovery
#[derive(Clone, Debug)]
pub struct DiscoverReq {
    pub base: PathBuf,
}

/// Port for repository discovery
pub trait DiscoveryPort: Send + Sync {
    /// Scan for repositories in the given base directory
    /// This is blocking - caller should run in spawn_blocking
    fn scan(&self, req: DiscoverReq) -> Result<Vec<(RepoId, RepoMeta)>>;
}