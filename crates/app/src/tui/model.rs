use gitagrip_core::app::ReadProjection;
use gitagrip_core::domain::{Event, RepoId};
use std::collections::HashMap;

/// The TUI Model - this represents the complete UI state
/// This is separate from the core ReadProjection to allow UI-specific state
#[derive(Debug, Default)]
pub struct TuiModel {
    /// Core data from the application service
    pub projection: ReadProjection,
    
    /// UI-specific state
    pub ui_state: UiState,
    
    /// Current input state
    pub input: InputState,
    
    /// Current view mode
    pub mode: ViewMode,
    
    /// Error messages to display
    pub errors: Vec<String>,
    
    /// Status messages to display
    pub messages: Vec<String>,
    
    /// Whether the application should quit
    pub should_quit: bool,
}

/// UI-specific state (cursor position, selections, etc.)
#[derive(Debug, Default)]
pub struct UiState {
    /// Currently selected group (if any)
    pub selected_group: Option<String>,
    
    /// Currently selected repository (if any)
    pub selected_repo: Option<RepoId>,
    
    /// Cursor position in current list
    pub cursor_position: usize,
    
    /// Whether groups are expanded or collapsed
    pub expanded_groups: HashMap<String, bool>,
    
    /// Current scroll position
    pub scroll_offset: usize,
    
    /// Terminal size
    pub terminal_width: u16,
    pub terminal_height: u16,
    
    /// Whether help is shown
    pub show_help: bool,
    
    /// Whether we're in organize mode
    pub organize_mode: bool,
    
    /// Marked repositories for bulk operations
    pub marked_repos: Vec<RepoId>,
    
    /// Last update timestamp for status refresh indicator
    pub last_refresh: Option<std::time::SystemTime>,
}

/// Input state for text input modes
#[derive(Debug, Default)]
pub struct InputState {
    /// Current input mode
    pub mode: InputMode,
    
    /// Current input text
    pub text: String,
    
    /// Cursor position in input
    pub cursor: usize,
    
    /// Input prompt text
    pub prompt: String,
    
    /// Input placeholder text
    pub placeholder: String,
}

/// Input modes for different text entry scenarios  
#[derive(Debug, Default, Clone, PartialEq)]
pub enum InputMode {
    #[default]
    None,
    
    /// Searching/filtering repositories
    Search,
    
    /// Creating a new group
    NewGroup,
    
    /// Renaming a group
    RenameGroup { old_name: String },
    
    /// Entering a git command
    GitCommand { repo_id: RepoId },
    
    /// Entering a path for scanning
    ScanPath,
}

/// Different view modes for the TUI
#[derive(Debug, Default, Clone, PartialEq)]
pub enum ViewMode {
    #[default]
    /// Normal repository list view
    RepoList,
    
    /// Group management/organization view
    Organize,
    
    /// Repository details view
    RepoDetails { repo_id: RepoId },
    
    /// Commit log view
    CommitLog { repo_id: RepoId },
    
    /// Configuration view
    Config,
    
    /// Help view
    Help,
}

impl TuiModel {
    pub fn new() -> Self {
        Self::default()
    }
    
    /// Update the model with a new projection from the app service
    pub fn update_projection(&mut self, projection: ReadProjection) {
        self.projection = projection;
        
        // Update UI state based on new data
        self.update_ui_state_after_projection_change();
    }
    
    /// Apply an event to update both projection and UI state
    pub fn apply_event(&mut self, event: &Event) {
        // First update the projection
        self.projection.apply(event);
        
        // Then handle UI-specific updates
        match event {
            Event::RepoDiscovered { id, meta: _ } => {
                // If this is the first repo, select it
                if self.ui_state.selected_repo.is_none() {
                    self.ui_state.selected_repo = Some(id.clone());
                }
            }
            
            Event::ScanCompleted => {
                self.messages.push("Repository scan completed".to_string());
            }
            
            Event::StatusUpdated { id: _, status: _ } => {
                self.ui_state.last_refresh = Some(std::time::SystemTime::now());
            }
            
            Event::FetchProgress { id, done, total } => {
                self.messages.push(format!("Fetching {}: {}/{}", id.0, done, total));
            }
            
            Event::RepoFetched { id, ok, msg } => {
                if *ok {
                    self.messages.push(format!("Successfully fetched {}", id.0));
                } else if let Some(error_msg) = msg {
                    self.errors.push(format!("Failed to fetch {}: {}", id.0, error_msg));
                }
            }
            
            Event::LogLoaded { id: _, commits } => {
                self.messages.push(format!("Loaded {} commits", commits.len()));
            }
            
            Event::Error { id, msg } => {
                if let Some(repo_id) = id {
                    self.errors.push(format!("Error for {}: {}", repo_id.0, msg));
                } else {
                    self.errors.push(msg.clone());
                }
            }
            
            Event::QuitRequested => {
                self.should_quit = true;
            }
        }
    }
    
    /// Update UI state after projection changes
    fn update_ui_state_after_projection_change(&mut self) {
        // Remove selections for repos that no longer exist
        if let Some(selected_id) = &self.ui_state.selected_repo {
            if !self.projection.repositories.contains_key(selected_id) {
                self.ui_state.selected_repo = None;
            }
        }
        
        // Clean up expanded groups that no longer exist
        let existing_groups: std::collections::HashSet<_> = self.projection
            .repositories_by_auto_group()
            .keys()
            .cloned()
            .collect();
            
        self.ui_state.expanded_groups.retain(|group_name, _| {
            existing_groups.contains(group_name)
        });
        
        // Clean up marked repos that no longer exist
        self.ui_state.marked_repos.retain(|repo_id| {
            self.projection.repositories.contains_key(repo_id)
        });
    }
    
    /// Get repositories for the current view, respecting filters and grouping
    pub fn get_display_repositories(&self) -> Vec<(Option<String>, Vec<&gitagrip_core::domain::RepoMeta>)> {
        let mut result = Vec::new();
        
        match &self.mode {
            ViewMode::RepoList | ViewMode::Organize => {
                let grouped = self.projection.repositories_by_auto_group();
                
                for (group_name, repos) in grouped {
                    // Skip collapsed groups unless we're in organize mode
                    if !self.mode_is_organize() && !self.is_group_expanded(&group_name) {
                        continue;
                    }
                    
                    result.push((Some(group_name), repos));
                }
                
                // Sort by group name for consistent display
                result.sort_by(|a, b| a.0.cmp(&b.0));
            }
            
            _ => {
                // Other modes might have different repository display logic
            }
        }
        
        result
    }
    
    /// Check if a group is expanded
    pub fn is_group_expanded(&self, group_name: &str) -> bool {
        self.ui_state.expanded_groups.get(group_name).copied().unwrap_or(true)
    }
    
    /// Toggle group expansion state
    pub fn toggle_group(&mut self, group_name: &str) {
        let current = self.is_group_expanded(group_name);
        self.ui_state.expanded_groups.insert(group_name.to_string(), !current);
    }
    
    /// Check if we're in organize mode
    pub fn mode_is_organize(&self) -> bool {
        matches!(self.mode, ViewMode::Organize) || self.ui_state.organize_mode
    }
    
    /// Check if a repo is marked for bulk operations
    pub fn is_repo_marked(&self, repo_id: &RepoId) -> bool {
        self.ui_state.marked_repos.contains(repo_id)
    }
    
    /// Toggle repo mark for bulk operations
    pub fn toggle_repo_mark(&mut self, repo_id: RepoId) {
        if let Some(pos) = self.ui_state.marked_repos.iter().position(|id| id == &repo_id) {
            self.ui_state.marked_repos.remove(pos);
        } else {
            self.ui_state.marked_repos.push(repo_id);
        }
    }
    
    /// Clear all error messages
    pub fn clear_errors(&mut self) {
        self.errors.clear();
    }
    
    /// Clear all status messages
    pub fn clear_messages(&mut self) {
        self.messages.clear();
    }
    
    /// Add a status message
    pub fn add_message(&mut self, message: String) {
        self.messages.push(message);
    }
    
    /// Add an error message
    pub fn add_error(&mut self, error: String) {
        self.errors.push(error);
    }
}