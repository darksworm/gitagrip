use anyhow::{Context, Result};
use git2::{Repository as GitRepository, StatusOptions};
use gitagrip_core::domain::{Author, Commit, RepoId, RepoStatus, Timestamp, AheadBehind};
use gitagrip_core::ports::GitPort;
use std::any::Any;
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::{Arc, RwLock};

/// Git adapter that implements GitPort using git2
pub struct GitAdapter {
    /// Cache of repository paths keyed by RepoId
    repo_paths: Arc<RwLock<HashMap<RepoId, PathBuf>>>,
}

impl GitAdapter {
    pub fn new() -> Self {
        Self {
            repo_paths: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Register a repository path with its ID
    pub fn register_repo(&self, id: RepoId, path: PathBuf) {
        let mut paths = self.repo_paths.write().unwrap();
        paths.insert(id, path);
    }

    /// Get repository path for a given ID
    fn get_repo_path(&self, id: &RepoId) -> Option<PathBuf> {
        let paths = self.repo_paths.read().unwrap();
        paths.get(id).cloned()
    }

    /// Open git repository for the given ID
    fn open_repo(&self, id: &RepoId) -> Result<GitRepository> {
        let path = self.get_repo_path(id)
            .ok_or_else(|| anyhow::anyhow!("Repository path not found for ID: {}", id.0))?;
        
        GitRepository::open(&path)
            .with_context(|| format!("Failed to open git repository at {}", path.display()))
    }
}

impl GitPort for GitAdapter {
    fn status(&self, id: &RepoId) -> Result<RepoStatus> {
        let git_repo = self.open_repo(id)?;
        let repo_path = self.get_repo_path(id).unwrap();
        
        let name = repo_path
            .file_name()
            .unwrap_or_default()
            .to_string_lossy()
            .to_string();
        
        // Get branch information
        let head = git_repo.head().ok();
        let (branch_name, is_detached) = match &head {
            Some(reference) => {
                if reference.is_branch() {
                    (reference.shorthand().map(|s| s.to_string()), false)
                } else {
                    // Detached HEAD - show short commit hash
                    if let Some(oid) = reference.target() {
                        (Some(format!("{:.8}", oid)), true)
                    } else {
                        (None, true)
                    }
                }
            }
            None => (None, false),
        };
        
        // Check working directory status
        let mut status_options = StatusOptions::new();
        status_options.include_untracked(true);
        status_options.include_ignored(false);
        
        let statuses = git_repo
            .statuses(Some(&mut status_options))
            .context("Failed to get git status")?;
        
        let is_dirty = !statuses.is_empty();
        let mut has_staged = false;
        let mut has_unstaged = false;
        
        for entry in statuses.iter() {
            let status = entry.status();
            if status.intersects(git2::Status::INDEX_NEW | git2::Status::INDEX_MODIFIED | git2::Status::INDEX_DELETED | git2::Status::INDEX_RENAMED | git2::Status::INDEX_TYPECHANGE) {
                has_staged = true;
            }
            if status.intersects(git2::Status::WT_MODIFIED | git2::Status::WT_DELETED | git2::Status::WT_TYPECHANGE | git2::Status::WT_RENAMED | git2::Status::WT_NEW) {
                has_unstaged = true;
            }
        }
        
        // Get last commit summary
        let last_commit_summary = if let Ok(commit) = git_repo.head().and_then(|r| r.peel_to_commit()) {
            commit.summary().unwrap_or("").to_string()
        } else {
            "No commits".to_string()
        };
        
        // Get ahead/behind counts relative to upstream
        let (ahead_count, behind_count) = if let Some(reference) = &head {
            if let Ok(local_oid) = reference.target().ok_or("No target OID") {
                if let Some(ref_name) = reference.name() {
                    if let Ok(upstream_ref) = git_repo.branch_upstream_name(ref_name) {
                        if let Some(upstream_str) = upstream_ref.as_str() {
                            if let Ok(upstream_oid) = git_repo.refname_to_id(upstream_str) {
                                match git_repo.graph_ahead_behind(local_oid, upstream_oid) {
                                    Ok((ahead, behind)) => (ahead, behind),
                                    Err(_) => (0, 0),
                                }
                            } else {
                                (0, 0)
                            }
                        } else {
                            (0, 0)
                        }
                    } else {
                        (0, 0)
                    }
                } else {
                    (0, 0)
                }
            } else {
                (0, 0)
            }
        } else {
            (0, 0)
        };
        
        Ok(RepoStatus {
            name,
            path: repo_path,
            branch_name,
            is_dirty,
            ahead_count,
            behind_count,
            is_detached,
            has_staged,
            has_unstaged,
            last_commit_summary,
        })
    }

    fn ahead_behind(&self, id: &RepoId, upstream: &str) -> Result<AheadBehind> {
        let git_repo = self.open_repo(id)?;
        
        let head_oid = git_repo
            .head()?
            .target()
            .ok_or_else(|| anyhow::anyhow!("HEAD has no target OID"))?;
        
        let upstream_oid = git_repo
            .refname_to_id(upstream)
            .with_context(|| format!("Failed to resolve upstream ref: {}", upstream))?;
        
        let (ahead, behind) = git_repo
            .graph_ahead_behind(head_oid, upstream_oid)
            .context("Failed to calculate ahead/behind counts")?;
        
        Ok(AheadBehind { 
            ahead: ahead as u32, 
            behind: behind as u32 
        })
    }

    fn fetch(&self, id: &RepoId, remote: &str, prune: bool) -> Result<()> {
        let git_repo = self.open_repo(id)?;
        
        let mut remote_obj = git_repo
            .find_remote(remote)
            .with_context(|| format!("Remote '{}' not found", remote))?;
        
        let mut fetch_options = git2::FetchOptions::new();
        if prune {
            fetch_options.prune(git2::FetchPrune::On);
        }
        
        remote_obj
            .fetch(&[] as &[&str], Some(&mut fetch_options), None)
            .context("Failed to fetch from remote")?;
        
        Ok(())
    }

    fn log(&self, id: &RepoId, range: &str, limit: usize) -> Result<Vec<Commit>> {
        let git_repo = self.open_repo(id)?;
        
        let mut revwalk = git_repo.revwalk()?;
        
        // Parse range (e.g., "HEAD", "origin/main..HEAD", "HEAD~10..HEAD")
        if range.is_empty() || range == "HEAD" {
            revwalk.push_head()?;
        } else if range.contains("..") {
            // Range format like "origin/main..HEAD"
            let parts: Vec<&str> = range.split("..").collect();
            if parts.len() == 2 {
                let from = parts[0];
                let to = parts[1];
                
                if !from.is_empty() {
                    let from_oid = git_repo.revparse_single(from)?.id();
                    revwalk.push(from_oid)?;
                    revwalk.hide(from_oid)?;
                }
                
                if !to.is_empty() {
                    let to_oid = git_repo.revparse_single(to)?.id();
                    revwalk.push(to_oid)?;
                } else {
                    revwalk.push_head()?;
                }
            }
        } else {
            // Single ref
            let oid = git_repo.revparse_single(range)?.id();
            revwalk.push(oid)?;
        }
        
        revwalk.set_sorting(git2::Sort::TIME)?;
        
        let mut commits = Vec::new();
        for (i, oid) in revwalk.enumerate() {
            if i >= limit {
                break;
            }
            
            let oid = oid?;
            let commit = git_repo.find_commit(oid)?;
            
            commits.push(Commit {
                id: format!("{}", oid),
                message: commit.summary().unwrap_or("").to_string(),
                author: Author {
                    name: commit.author().name().unwrap_or("").to_string(),
                    email: commit.author().email().unwrap_or("").to_string(),
                },
                timestamp: Timestamp::new(commit.time().seconds(), commit.time().offset_minutes()),
            });
        }
        
        Ok(commits)
    }

    fn as_any(&self) -> &dyn Any {
        self
    }
}

impl Default for GitAdapter {
    fn default() -> Self {
        Self::new()
    }
}