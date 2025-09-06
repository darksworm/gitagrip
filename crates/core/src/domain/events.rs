use super::{
    repo::{RepoId, RepoMeta, RepoStatus},
    commit::Commit,
};

/// Domain events emitted by the core application
#[derive(Debug, Clone)]
pub enum Event {
    /// A repository was discovered during scanning
    RepoDiscovered { id: RepoId, meta: RepoMeta },
    
    /// Repository scanning completed
    ScanCompleted,
    
    /// Repository status was updated
    StatusUpdated { id: RepoId, status: RepoStatus },
    
    /// Progress update during fetch operation
    FetchProgress { id: RepoId, done: u32, total: u32 },
    
    /// Repository fetch completed
    RepoFetched { id: RepoId, ok: bool, msg: Option<String> },
    
    /// Log was loaded for a repository
    LogLoaded { id: RepoId, commits: Vec<Commit> },
    
    /// An error occurred
    Error { id: Option<RepoId>, msg: String },
    
    /// User requested to quit the application
    QuitRequested,
}