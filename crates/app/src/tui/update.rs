use anyhow::Result;
use crossterm::event::{KeyCode, KeyModifiers};
use gitagrip_core::app::Command;
use gitagrip_core::domain::{Event, RepoId};
use super::model::{TuiModel, ViewMode, InputMode};

/// Messages that can be sent from the TUI to the application service
#[derive(Debug, Clone)]
pub enum TuiMessage {
    /// Send a command to the app service
    Command(Command),
    
    /// Send an event
    Event(Event),
    
    /// No action needed
    None,
}

/// The Update function - handles user input and updates the model
/// This is the core of the MVU pattern's Update component
pub struct TuiUpdate;

impl TuiUpdate {
    /// Handle a key press and update the model accordingly
    /// Returns a TuiMessage that should be sent to the app service
    pub fn handle_key(model: &mut TuiModel, key: KeyCode, modifiers: KeyModifiers) -> Result<TuiMessage> {
        // Handle global keys first (quit, help, etc.)
        if let Some(msg) = Self::handle_global_keys(model, key, modifiers)? {
            return Ok(msg);
        }
        
        // Handle input mode keys if we're in text input
        if model.input.mode != InputMode::None {
            return Self::handle_input_keys(model, key, modifiers);
        }
        
        // Handle mode-specific keys
        match &model.mode {
            ViewMode::RepoList => Self::handle_repo_list_keys(model, key, modifiers),
            ViewMode::Organize => Self::handle_organize_keys(model, key, modifiers),
            ViewMode::RepoDetails { repo_id } => Self::handle_repo_details_keys(model, key, modifiers, repo_id.clone()),
            ViewMode::CommitLog { repo_id } => Self::handle_commit_log_keys(model, key, modifiers, repo_id.clone()),
            ViewMode::Config => Self::handle_config_keys(model, key, modifiers),
            ViewMode::Help => Self::handle_help_keys(model, key, modifiers),
        }
    }
    
    /// Handle terminal resize
    pub fn handle_resize(model: &mut TuiModel, width: u16, height: u16) -> Result<TuiMessage> {
        model.ui_state.terminal_width = width;
        model.ui_state.terminal_height = height;
        Ok(TuiMessage::None)
    }
    
    /// Handle global keys that work in any mode
    fn handle_global_keys(model: &mut TuiModel, key: KeyCode, modifiers: KeyModifiers) -> Result<Option<TuiMessage>> {
        match key {
            KeyCode::Char('q') if modifiers.is_empty() => {
                // In organize mode, 'q' exits to normal mode
                if matches!(model.mode, ViewMode::Organize) {
                    model.mode = ViewMode::RepoList;
                    model.ui_state.organize_mode = false;
                    Ok(Some(TuiMessage::None))
                } else {
                    // In other modes, 'q' quits
                    Ok(Some(TuiMessage::Command(Command::Quit)))
                }
            }
            
            KeyCode::Char('c') if modifiers.contains(KeyModifiers::CONTROL) => {
                Ok(Some(TuiMessage::Command(Command::Quit)))
            }
            
            KeyCode::Esc => {
                // Escape key behavior depends on context
                if model.input.mode != InputMode::None {
                    // Cancel input mode
                    model.input.mode = InputMode::None;
                    model.input.text.clear();
                    Ok(Some(TuiMessage::None))
                } else if matches!(model.mode, ViewMode::Help) {
                    // Exit help
                    model.mode = ViewMode::RepoList;
                    Ok(Some(TuiMessage::None))
                } else {
                    // Otherwise quit
                    Ok(Some(TuiMessage::Command(Command::Quit)))
                }
            }
            
            KeyCode::Char('?') if modifiers.is_empty() => {
                model.mode = ViewMode::Help;
                Ok(Some(TuiMessage::None))
            }
            
            KeyCode::F(5) => {
                // F5 = refresh all statuses
                let repo_ids: Vec<_> = model.projection.repositories.keys().cloned().collect();
                Ok(Some(TuiMessage::Command(Command::RefreshStatus { ids: repo_ids })))
            }
            
            _ => Ok(None)
        }
    }
    
    /// Handle keys when in text input mode
    fn handle_input_keys(model: &mut TuiModel, key: KeyCode, _modifiers: KeyModifiers) -> Result<TuiMessage> {
        match key {
            KeyCode::Char(c) => {
                model.input.text.push(c);
                Ok(TuiMessage::None)
            }
            
            KeyCode::Backspace => {
                model.input.text.pop();
                Ok(TuiMessage::None)
            }
            
            KeyCode::Enter => {
                let text = model.input.text.clone();
                let input_mode = model.input.mode.clone();
                
                // Clear input state
                model.input.mode = InputMode::None;
                model.input.text.clear();
                
                // Process the input based on mode
                Self::process_input_submission(model, input_mode, text)
            }
            
            KeyCode::Esc => {
                // Cancel input
                model.input.mode = InputMode::None;
                model.input.text.clear();
                Ok(TuiMessage::None)
            }
            
            _ => Ok(TuiMessage::None)
        }
    }
    
    /// Process submitted input text
    fn process_input_submission(model: &mut TuiModel, input_mode: InputMode, text: String) -> Result<TuiMessage> {
        match input_mode {
            InputMode::None => Ok(TuiMessage::None),
            
            InputMode::Search => {
                // TODO: Implement search filtering
                model.add_message(format!("Search not implemented yet: {}", text));
                Ok(TuiMessage::None)
            }
            
            InputMode::NewGroup => {
                // TODO: Create new group
                model.add_message(format!("New group creation not implemented yet: {}", text));
                Ok(TuiMessage::None)
            }
            
            InputMode::RenameGroup { old_name: _ } => {
                // TODO: Rename group
                model.add_message(format!("Group renaming not implemented yet: {}", text));
                Ok(TuiMessage::None)
            }
            
            InputMode::GitCommand { repo_id: _ } => {
                // TODO: Execute git command
                model.add_message(format!("Git command execution not implemented yet: {}", text));
                Ok(TuiMessage::None)
            }
            
            InputMode::ScanPath => {
                let path = std::path::PathBuf::from(text);
                Ok(TuiMessage::Command(Command::Rescan { base: path }))
            }
        }
    }
    
    /// Handle keys in repository list view
    fn handle_repo_list_keys(model: &mut TuiModel, key: KeyCode, modifiers: KeyModifiers) -> Result<TuiMessage> {
        match key {
            // Navigation
            KeyCode::Up | KeyCode::Char('k') => {
                if model.ui_state.cursor_position > 0 {
                    model.ui_state.cursor_position -= 1;
                }
                Ok(TuiMessage::None)
            }
            
            KeyCode::Down | KeyCode::Char('j') => {
                model.ui_state.cursor_position += 1;
                Ok(TuiMessage::None)
            }
            
            KeyCode::Left | KeyCode::Char('h') => {
                // Collapse group or go to parent
                Self::handle_left_navigation(model)
            }
            
            KeyCode::Right | KeyCode::Char('l') => {
                // Expand group or enter selection
                Self::handle_right_navigation(model)
            }
            
            // Mode switches
            KeyCode::Char('o') => {
                model.mode = ViewMode::Organize;
                model.ui_state.organize_mode = true;
                Ok(TuiMessage::None)
            }
            
            // Actions
            KeyCode::Char('r') if modifiers.is_empty() => {
                // Refresh selected repo or all repos
                if let Some(selected_id) = &model.ui_state.selected_repo {
                    Ok(TuiMessage::Command(Command::RefreshStatus { 
                        ids: vec![selected_id.clone()] 
                    }))
                } else {
                    let repo_ids: Vec<_> = model.projection.repositories.keys().cloned().collect();
                    Ok(TuiMessage::Command(Command::RefreshStatus { ids: repo_ids }))
                }
            }
            
            KeyCode::Char('f') => {
                // Fetch all repositories
                Ok(TuiMessage::Command(Command::FetchAll { prune: false }))
            }
            
            KeyCode::Char('F') => {
                // Fetch all repositories with prune
                Ok(TuiMessage::Command(Command::FetchAll { prune: true }))
            }
            
            KeyCode::Char('s') => {
                // Start scan path input
                model.input.mode = InputMode::ScanPath;
                model.input.prompt = "Enter path to scan:".to_string();
                Ok(TuiMessage::None)
            }
            
            KeyCode::Char('/') => {
                // Start search input
                model.input.mode = InputMode::Search;
                model.input.prompt = "Search repositories:".to_string();
                Ok(TuiMessage::None)
            }
            
            KeyCode::Enter => {
                // Open repo details or execute default action
                if let Some(selected_id) = &model.ui_state.selected_repo {
                    model.mode = ViewMode::RepoDetails { repo_id: selected_id.clone() };
                }
                Ok(TuiMessage::None)
            }
            
            _ => Ok(TuiMessage::None)
        }
    }
    
    /// Handle keys in organize mode
    fn handle_organize_keys(model: &mut TuiModel, key: KeyCode, _modifiers: KeyModifiers) -> Result<TuiMessage> {
        match key {
            // Navigation (same as repo list)
            KeyCode::Up | KeyCode::Char('k') => {
                if model.ui_state.cursor_position > 0 {
                    model.ui_state.cursor_position -= 1;
                }
                Ok(TuiMessage::None)
            }
            
            KeyCode::Down | KeyCode::Char('j') => {
                model.ui_state.cursor_position += 1;
                Ok(TuiMessage::None)
            }
            
            // Mark/unmark repositories
            KeyCode::Char(' ') => {
                if let Some(selected_id) = &model.ui_state.selected_repo {
                    model.toggle_repo_mark(selected_id.clone());
                }
                Ok(TuiMessage::None)
            }
            
            KeyCode::Char('n') => {
                // Create new group
                model.input.mode = InputMode::NewGroup;
                model.input.prompt = "Enter new group name:".to_string();
                Ok(TuiMessage::None)
            }
            
            _ => Ok(TuiMessage::None)
        }
    }
    
    /// Handle keys in repository details view
    fn handle_repo_details_keys(model: &mut TuiModel, key: KeyCode, _modifiers: KeyModifiers, repo_id: RepoId) -> Result<TuiMessage> {
        match key {
            KeyCode::Char('l') => {
                // Show commit log
                model.mode = ViewMode::CommitLog { repo_id: repo_id.clone() };
                Ok(TuiMessage::Command(Command::ShowLog { 
                    id: repo_id, 
                    range: None, 
                    limit: 50 
                }))
            }
            
            KeyCode::Char('f') => {
                // Fetch this repository
                Ok(TuiMessage::Command(Command::FetchAll { prune: false }))
            }
            
            KeyCode::Char('o') => {
                // Open in external app
                Ok(TuiMessage::Command(Command::OpenRepo { id: repo_id }))
            }
            
            KeyCode::Char('b') | KeyCode::Backspace => {
                // Go back to repo list
                model.mode = ViewMode::RepoList;
                Ok(TuiMessage::None)
            }
            
            _ => Ok(TuiMessage::None)
        }
    }
    
    /// Handle keys in commit log view  
    fn handle_commit_log_keys(model: &mut TuiModel, key: KeyCode, _modifiers: KeyModifiers, _repo_id: RepoId) -> Result<TuiMessage> {
        match key {
            // Navigation
            KeyCode::Up | KeyCode::Char('k') => {
                if model.ui_state.cursor_position > 0 {
                    model.ui_state.cursor_position -= 1;
                }
                Ok(TuiMessage::None)
            }
            
            KeyCode::Down | KeyCode::Char('j') => {
                model.ui_state.cursor_position += 1;
                Ok(TuiMessage::None)
            }
            
            KeyCode::Char('b') | KeyCode::Backspace => {
                // Go back to repo list
                model.mode = ViewMode::RepoList;
                Ok(TuiMessage::None)
            }
            
            _ => Ok(TuiMessage::None)
        }
    }
    
    /// Handle keys in config view
    fn handle_config_keys(model: &mut TuiModel, key: KeyCode, _modifiers: KeyModifiers) -> Result<TuiMessage> {
        match key {
            KeyCode::Char('b') | KeyCode::Backspace => {
                model.mode = ViewMode::RepoList;
                Ok(TuiMessage::None)
            }
            
            _ => Ok(TuiMessage::None)
        }
    }
    
    /// Handle keys in help view
    fn handle_help_keys(model: &mut TuiModel, _key: KeyCode, _modifiers: KeyModifiers) -> Result<TuiMessage> {
        // Any key exits help
        model.mode = ViewMode::RepoList;
        Ok(TuiMessage::None)
    }
    
    /// Handle left navigation (collapse/parent)
    fn handle_left_navigation(_model: &mut TuiModel) -> Result<TuiMessage> {
        // TODO: Implement group collapse logic
        Ok(TuiMessage::None)
    }
    
    /// Handle right navigation (expand/enter)
    fn handle_right_navigation(_model: &mut TuiModel) -> Result<TuiMessage> {
        // TODO: Implement group expand logic
        Ok(TuiMessage::None)
    }
}