use serde::{Deserialize, Serialize};
use std::path::PathBuf;

/// Unique identifier for a repository
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct RepoId(pub String);

impl RepoId {
    pub fn from_path(path: &std::path::Path) -> Self {
        Self(path.to_string_lossy().to_string())
    }
}

impl std::fmt::Display for RepoId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// Repository metadata (discovered during scanning)
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RepoMeta {
    pub name: String,
    pub path: PathBuf,
    pub auto_group: String,
}

impl std::fmt::Display for RepoMeta {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} ({})", self.name, self.path.display())
    }
}

/// Git repository status information
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RepoStatus {
    pub name: String,
    pub path: PathBuf,
    pub branch_name: Option<String>,
    pub is_dirty: bool,
    pub ahead_count: usize,
    pub behind_count: usize,
    pub is_detached: bool,
    pub has_staged: bool,
    pub has_unstaged: bool,
    pub last_commit_summary: String,
}

/// Ahead/behind counts for a branch
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AheadBehind {
    pub ahead: u32,
    pub behind: u32,
}

/// Repository group configuration
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Group {
    pub name: String,
    pub repos: Vec<PathBuf>,
}

impl Group {
    pub fn new(name: String) -> Self {
        Self {
            name,
            repos: Vec::new(),
        }
    }

    pub fn add_repo(&mut self, path: PathBuf) {
        if !self.repos.contains(&path) {
            self.repos.push(path);
        }
    }

    pub fn remove_repo(&mut self, path: &std::path::Path) {
        self.repos.retain(|p| p != path);
    }

    pub fn contains_repo(&self, path: &std::path::Path) -> bool {
        self.repos.iter().any(|p| p == path)
    }

    pub fn is_empty(&self) -> bool {
        self.repos.is_empty()
    }
}