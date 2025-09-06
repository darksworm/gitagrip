use crate::domain::{Event, RepoId, RepoMeta, RepoStatus, Group};
use std::collections::HashMap;

/// Read-only projection of application state for UI consumption
#[derive(Debug, Default)]
pub struct ReadProjection {
    /// All discovered repositories
    pub repositories: HashMap<RepoId, RepoMeta>,
    
    /// Current status of repositories  
    pub statuses: HashMap<RepoId, RepoStatus>,
    
    /// User-defined groups
    pub groups: HashMap<String, Group>,
    
    /// Whether scanning is in progress
    pub scanning: bool,
    
    /// Whether status refresh is in progress
    pub refreshing_status: bool,
}

impl ReadProjection {
    pub fn new() -> Self {
        Self::default()
    }
    
    /// Apply an event to update the projection
    pub fn apply(&mut self, event: &Event) {
        match event {
            Event::RepoDiscovered { id, meta } => {
                self.repositories.insert(id.clone(), meta.clone());
            }
            
            Event::ScanCompleted => {
                self.scanning = false;
            }
            
            Event::StatusUpdated { id, status } => {
                self.statuses.insert(id.clone(), status.clone());
            }
            
            Event::FetchProgress { .. } => {
                // Could track individual repo fetch progress here
            }
            
            Event::RepoFetched { .. } => {
                // Could update fetch completion status
            }
            
            Event::LogLoaded { .. } => {
                // Could cache log data if needed
            }
            
            Event::Error { .. } => {
                // Could track error state
            }
            
            Event::QuitRequested => {
                // No state change needed
            }
        }
    }
    
    /// Get all repositories grouped by their auto-detected groups
    pub fn repositories_by_auto_group(&self) -> HashMap<String, Vec<&RepoMeta>> {
        let mut groups = HashMap::new();
        
        for repo in self.repositories.values() {
            groups.entry(repo.auto_group.clone())
                .or_insert_with(Vec::new)
                .push(repo);
        }
        
        groups
    }
    
    /// Get repositories in a specific user-defined group
    pub fn repositories_in_group(&self, group_name: &str) -> Vec<&RepoMeta> {
        if let Some(group) = self.groups.get(group_name) {
            group.repos.iter()
                .filter_map(|path| {
                    self.repositories.values()
                        .find(|repo| repo.path == *path)
                })
                .collect()
        } else {
            Vec::new()
        }
    }
    
    /// Get repositories not assigned to any user group
    pub fn ungrouped_repositories(&self) -> Vec<&RepoMeta> {
        let grouped_paths: std::collections::HashSet<_> = self.groups.values()
            .flat_map(|group| &group.repos)
            .collect();
            
        self.repositories.values()
            .filter(|repo| !grouped_paths.contains(&repo.path))
            .collect()
    }
}