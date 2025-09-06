use anyhow::{Context, Result};
use gitagrip_core::domain::{RepoId, RepoMeta};
use gitagrip_core::ports::{DiscoveryPort, DiscoverReq};
use std::collections::BTreeMap;
use std::path::Path;
use walkdir::WalkDir;

/// File system discovery adapter that implements DiscoveryPort
pub struct FsDiscoveryAdapter;

impl FsDiscoveryAdapter {
    pub fn new() -> Self {
        Self
    }

    /// Find all git repositories in the given base path
    fn find_repos<P: AsRef<Path>>(&self, base_path: P) -> Result<Vec<RepoMeta>> {
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
                let auto_group = self.determine_auto_group(&repo_path, base_path);
                
                repositories.push(RepoMeta {
                    name,
                    path: repo_path,
                    auto_group,
                });
            }
        }
        
        Ok(repositories)
    }

    /// Determine the automatic group for a repository based on its path
    fn determine_auto_group(&self, repo_path: &Path, base_path: &Path) -> String {
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

    /// Group repositories by their auto_group field
    pub fn group_repositories(&self, repositories: &[RepoMeta]) -> BTreeMap<String, Vec<RepoMeta>> {
        let mut groups = BTreeMap::new();
        
        for repo in repositories {
            let group_name = repo.auto_group.clone();
            groups.entry(group_name).or_insert_with(Vec::new).push(repo.clone());
        }
        
        // Sort repositories within each group by name for stable ordering
        for repos_in_group in groups.values_mut() {
            repos_in_group.sort_by(|a, b| a.name.cmp(&b.name));
        }
        
        groups
    }
}

impl DiscoveryPort for FsDiscoveryAdapter {
    fn scan(&self, req: DiscoverReq) -> Result<Vec<(RepoId, RepoMeta)>> {
        let repositories = self.find_repos(&req.base)?;
        
        // Convert to the expected format with RepoId
        let mut result = Vec::new();
        for repo in repositories {
            let repo_id = RepoId(format!("{}#{}", repo.name, repo.path.display()));
            result.push((repo_id, repo));
        }
        
        Ok(result)
    }
}

impl Default for FsDiscoveryAdapter {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::PathBuf;
    use tempfile::TempDir;

    fn create_test_git_repo(path: &Path) -> Result<()> {
        fs::create_dir_all(path)?;
        fs::create_dir(path.join(".git"))?;
        Ok(())
    }

    #[test]
    fn test_determine_auto_group() -> Result<()> {
        let adapter = FsDiscoveryAdapter::new();
        let base = Path::new("/base");
        
        // Direct child of base
        let repo1 = Path::new("/base/repo1");
        assert_eq!(adapter.determine_auto_group(repo1, base), "Ungrouped");
        
        // In subdirectory
        let repo2 = Path::new("/base/work/repo2");
        assert_eq!(adapter.determine_auto_group(repo2, base), "Auto: work");
        
        // Nested deeper
        let repo3 = Path::new("/base/projects/frontend/repo3");
        assert_eq!(adapter.determine_auto_group(repo3, base), "Auto: projects");
        
        Ok(())
    }

    #[test]
    fn test_find_repos_empty_directory() -> Result<()> {
        let adapter = FsDiscoveryAdapter::new();
        let temp_dir = TempDir::new()?;
        let repos = adapter.find_repos(temp_dir.path())?;
        assert!(repos.is_empty());
        Ok(())
    }

    #[test]
    fn test_find_repos_with_git_repos() -> Result<()> {
        let adapter = FsDiscoveryAdapter::new();
        let temp_dir = TempDir::new()?;
        let base_path = temp_dir.path();
        
        // Create a fake git repo (just create .git directory)
        let repo_path = base_path.join("test-repo");
        create_test_git_repo(&repo_path)?;
        
        let repos = adapter.find_repos(base_path)?;
        assert_eq!(repos.len(), 1);
        assert_eq!(repos[0].name, "test-repo");
        assert_eq!(repos[0].path, repo_path);
        assert_eq!(repos[0].auto_group, "Ungrouped");
        
        Ok(())
    }

    #[test]
    fn test_group_repositories() -> Result<()> {
        let adapter = FsDiscoveryAdapter::new();
        let repos = vec![
            RepoMeta {
                name: "repo1".to_string(),
                path: PathBuf::from("/base/repo1"),
                auto_group: "Ungrouped".to_string(),
            },
            RepoMeta {
                name: "repo3".to_string(), // Name starts with 3 to test sorting
                path: PathBuf::from("/base/work/repo3"),
                auto_group: "Auto: work".to_string(),
            },
            RepoMeta {
                name: "repo2".to_string(), // Name starts with 2 to test sorting
                path: PathBuf::from("/base/work/repo2"),
                auto_group: "Auto: work".to_string(),
            },
        ];
        
        let grouped = adapter.group_repositories(&repos);
        
        assert_eq!(grouped.len(), 2);
        assert_eq!(grouped["Ungrouped"].len(), 1);
        assert_eq!(grouped["Auto: work"].len(), 2);
        
        // Test that groups are returned in sorted order (BTreeMap)
        let group_names: Vec<_> = grouped.keys().collect();
        assert_eq!(group_names, vec!["Auto: work", "Ungrouped"]); // Alphabetical order
        
        // Test that repos within groups are sorted by name
        let work_repos = &grouped["Auto: work"];
        assert_eq!(work_repos[0].name, "repo2"); // Should come before repo3
        assert_eq!(work_repos[1].name, "repo3");
        
        Ok(())
    }
}