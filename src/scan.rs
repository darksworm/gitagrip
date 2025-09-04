use anyhow::{Context, Result};
use crossbeam_channel::Sender;
use std::collections::HashMap;
use std::fmt;
use std::path::{Path, PathBuf};
use walkdir::WalkDir;

#[derive(Debug, Clone, PartialEq)]
pub struct Repository {
    pub name: String,
    pub path: PathBuf,
    pub auto_group: String,
}

impl fmt::Display for Repository {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} ({})", self.name, self.path.display())
    }
}

#[derive(Debug)]
pub enum ScanEvent {
    RepoDiscovered(Repository),
    ScanCompleted,
    ScanError(String),
}

pub fn find_repos<P: AsRef<Path>>(base_path: P) -> Result<Vec<Repository>> {
    let mut repositories = Vec::new();
    let base_path = base_path.as_ref();
    
    for entry in WalkDir::new(base_path)
        .into_iter()
        .filter_entry(|e| {
            // Skip .git directories and don't descend into them
            if e.file_name() == ".git" {
                return false;
            }
            
            // If we're in a Git repository (parent has .git), don't descend further
            if let Some(parent) = e.path().parent() {
                if parent.join(".git").exists() && parent != base_path {
                    return false;
                }
            }
            
            true
        })
    {
        let entry = entry.context("Failed to read directory entry")?;
        
        // Check if this directory contains a .git subdirectory
        if entry.path().join(".git").is_dir() {
            let repo_path = entry.path().to_path_buf();
            let name = repo_path
                .file_name()
                .unwrap_or_default()
                .to_string_lossy()
                .to_string();
            
            // Determine auto group based on parent directory
            let auto_group = determine_auto_group(&repo_path, base_path);
            
            repositories.push(Repository {
                name,
                path: repo_path,
                auto_group,
            });
        }
    }
    
    Ok(repositories)
}

pub fn group_repositories(repositories: &[Repository]) -> HashMap<String, Vec<Repository>> {
    let mut groups = HashMap::new();
    
    for repo in repositories {
        let group_name = repo.auto_group.clone();
        groups.entry(group_name).or_insert_with(Vec::new).push(repo.clone());
    }
    
    groups
}

pub fn scan_repositories_background<P: AsRef<Path>>(
    base_path: P,
    sender: Sender<ScanEvent>,
) -> Result<()> {
    let repos = match find_repos(base_path) {
        Ok(repos) => repos,
        Err(e) => {
            let _ = sender.send(ScanEvent::ScanError(e.to_string()));
            return Err(e);
        }
    };
    
    for repo in repos {
        if sender.send(ScanEvent::RepoDiscovered(repo)).is_err() {
            // Receiver dropped, stop scanning
            return Ok(());
        }
    }
    
    let _ = sender.send(ScanEvent::ScanCompleted);
    Ok(())
}

fn determine_auto_group(repo_path: &Path, base_path: &Path) -> String {
    if let Ok(relative_path) = repo_path.strip_prefix(base_path) {
        if let Some(parent) = relative_path.parent() {
            if parent == Path::new("") {
                // Repository is directly in base_path
                return "Ungrouped".to_string();
            }
            
            // Get the first component after base_path for grouping
            if let Some(first_component) = parent.components().next() {
                if let Some(name) = first_component.as_os_str().to_str() {
                    return format!("Auto: {}", name);
                }
            }
        }
    }
    
    "Ungrouped".to_string()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::TempDir;

    #[test]
    fn test_repository_display() {
        let repo = Repository {
            name: "test-repo".to_string(),
            path: PathBuf::from("/path/to/repo"),
            auto_group: "Auto: parent".to_string(),
        };
        
        let display_str = format!("{}", repo);
        assert!(display_str.contains("test-repo"));
        assert!(display_str.contains("/path/to/repo"));
    }

    #[test]
    fn test_determine_auto_group() {
        let base = Path::new("/base");
        
        // Direct child of base
        let repo1 = Path::new("/base/repo1");
        assert_eq!(determine_auto_group(repo1, base), "Ungrouped");
        
        // In subdirectory
        let repo2 = Path::new("/base/work/repo2");
        assert_eq!(determine_auto_group(repo2, base), "Auto: work");
        
        // Nested deeper
        let repo3 = Path::new("/base/projects/frontend/repo3");
        assert_eq!(determine_auto_group(repo3, base), "Auto: projects");
    }

    #[test]
    fn test_find_repos_empty_directory() -> Result<()> {
        let temp_dir = TempDir::new()?;
        let repos = find_repos(temp_dir.path())?;
        assert!(repos.is_empty());
        Ok(())
    }

    #[test]
    fn test_find_repos_with_git_repos() -> Result<()> {
        let temp_dir = TempDir::new()?;
        let base_path = temp_dir.path();
        
        // Create a fake git repo (just create .git directory)
        let repo_path = base_path.join("test-repo");
        fs::create_dir_all(&repo_path)?;
        fs::create_dir(repo_path.join(".git"))?;
        
        let repos = find_repos(base_path)?;
        assert_eq!(repos.len(), 1);
        assert_eq!(repos[0].name, "test-repo");
        assert_eq!(repos[0].path, repo_path);
        assert_eq!(repos[0].auto_group, "Ungrouped");
        
        Ok(())
    }

    #[test]
    fn test_group_repositories() {
        let repos = vec![
            Repository {
                name: "repo1".to_string(),
                path: PathBuf::from("/base/repo1"),
                auto_group: "Ungrouped".to_string(),
            },
            Repository {
                name: "repo2".to_string(),
                path: PathBuf::from("/base/work/repo2"),
                auto_group: "Auto: work".to_string(),
            },
            Repository {
                name: "repo3".to_string(),
                path: PathBuf::from("/base/work/repo3"),
                auto_group: "Auto: work".to_string(),
            },
        ];
        
        let grouped = group_repositories(&repos);
        
        assert_eq!(grouped.len(), 2);
        assert_eq!(grouped["Ungrouped"].len(), 1);
        assert_eq!(grouped["Auto: work"].len(), 2);
    }

    #[test]
    fn test_scan_event_types() {
        let repo = Repository {
            name: "test".to_string(),
            path: PathBuf::from("/test"),
            auto_group: "Ungrouped".to_string(),
        };
        
        // Test that we can create different event types
        let _discovered = ScanEvent::RepoDiscovered(repo);
        let _completed = ScanEvent::ScanCompleted;
        let _error = ScanEvent::ScanError("test error".to_string());
    }
}