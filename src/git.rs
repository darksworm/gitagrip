use anyhow::{Context, Result};
use crossbeam_channel::Sender;
use git2::{Repository as GitRepository, StatusOptions};
use std::path::Path;

#[derive(Debug, Clone, PartialEq)]
pub struct RepoStatus {
    pub name: String,
    pub path: std::path::PathBuf,
    pub branch_name: Option<String>,
    pub is_dirty: bool,
    pub ahead_count: usize,
    pub behind_count: usize,
    pub is_detached: bool,
    pub has_staged: bool,
    pub has_unstaged: bool,
    pub last_commit_summary: String,
}

#[derive(Debug)]
pub enum StatusEvent {
    StatusUpdated { repository: String, status: RepoStatus },
    StatusScanCompleted,
    StatusError { repository: String, error: String },
}

pub fn read_status<P: AsRef<Path>>(repo_path: P) -> Result<RepoStatus> {
    let repo_path = repo_path.as_ref();
    let git_repo = GitRepository::open(repo_path)
        .with_context(|| format!("Failed to open git repository at {}", repo_path.display()))?;
    
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
        path: repo_path.to_path_buf(),
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


pub fn compute_statuses_with_events(
    repositories: &[crate::scan::Repository],
    sender: Sender<StatusEvent>,
) -> Result<()> {
    let repositories = repositories.to_vec();
    
    std::thread::spawn(move || {
        for repo in repositories {
            match read_status(&repo.path) {
                Ok(status) => {
                    if sender.send(StatusEvent::StatusUpdated {
                        repository: repo.name.clone(),
                        status,
                    }).is_err() {
                        // Receiver dropped, stop processing
                        return;
                    }
                }
                Err(e) => {
                    if sender
                        .send(StatusEvent::StatusError {
                            repository: repo.name.clone(),
                            error: format!("Failed to read status: {}", e),
                        })
                        .is_err()
                    {
                        // Receiver dropped, stop processing
                        return;
                    }
                }
            }
        }
        
        let _ = sender.send(StatusEvent::StatusScanCompleted);
    });
    
    Ok(())
}