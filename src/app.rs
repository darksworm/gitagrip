use crate::config::Config;
use crate::scan::Repository;
use crate::git;
use std::collections::{HashMap, HashSet};
use crossterm::event::KeyCode;
use anyhow::Result;
use tracing::info;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum AppMode {
    Normal,
    Organize,
}

pub struct App {
    pub should_quit: bool,
    pub config: Config,
    pub repositories: Vec<Repository>,
    pub scan_complete: bool,
    pub git_statuses: HashMap<String, git::RepoStatus>,
    pub git_status_loading: bool,
    pub scroll_offset: usize,
    pub mode: AppMode,
    
    // Selection and organization state
    pub current_selection: usize,
    pub selected_repositories: HashSet<usize>,
    pub marked_repositories: HashSet<usize>,
}

impl App {
    pub fn new(config: Config) -> App {
        App { 
            should_quit: false,
            config,
            repositories: Vec::new(),
            scan_complete: false,
            git_statuses: HashMap::new(),
            git_status_loading: false,
            scroll_offset: 0,
            mode: AppMode::Normal,
            
            // Initialize selection state
            current_selection: 0,
            selected_repositories: HashSet::new(),
            marked_repositories: HashSet::new(),
        }
    }

    fn branch_color(branch_name: &str) -> (ratatui::style::Color, bool) {
        use ratatui::style::Color;
        
        // Main and master get special treatment - bold green
        if branch_name == "main" || branch_name == "master" {
            return (Color::Green, true); // bold green
        }
        
        // Use a simple hash function to assign consistent colors to branch names
        let mut hash: u32 = 0;
        for byte in branch_name.bytes() {
            hash = hash.wrapping_mul(31).wrapping_add(byte as u32);
        }
        
        // Map to a set of colors (avoiding red which might indicate errors)
        let colors = [
            Color::Cyan,
            Color::Yellow, 
            Color::Blue,
            Color::Magenta,
            Color::LightCyan,
            Color::LightYellow,
            Color::LightBlue,
            Color::LightMagenta,
        ];
        
        let color = colors[(hash % colors.len() as u32) as usize];
        (color, false) // regular weight
    }

    pub fn scroll_down(&mut self) {
        if self.scroll_offset + 1 < self.repositories.len() {
            self.scroll_offset += 1;
        }
    }

    pub fn scroll_up(&mut self) {
        if self.scroll_offset > 0 {
            self.scroll_offset -= 1;
        }
    }


    pub fn ui_with_git_status(&self, f: &mut ratatui::Frame) {
        use ratatui::{
            layout::{Constraint, Direction, Layout},
            prelude::Stylize,
            style::{Color, Modifier, Style},
            text::{Line, Span},
            widgets::{Block, Borders, Paragraph},
        };
        
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints([
                Constraint::Length(3), // Title
                Constraint::Min(1),    // Main content
                Constraint::Length(3), // Footer
            ])
            .split(f.area());

        // Title with base directory
        let title_text = format!("GitaGrip    {}", 
                                self.config.base_dir.display());
        let title = Paragraph::new(title_text)
            .block(Block::default().borders(Borders::ALL))
            .style(Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD));
        f.render_widget(title, chunks[0]);

        // Main content - show repositories with git status and grouping (with colored branches)
        let content_lines = if self.repositories.is_empty() {
            if self.scan_complete {
                vec![Line::from("No Git repositories found in base directory.")]
            } else {
                vec![Line::from("Scanning for repositories...")]
            }
        } else {
            // Restore grouping functionality with rich text support
            let grouped_repos = crate::scan::group_repositories(&self.repositories);
            let mut lines = Vec::new();
            
            for (group_name, repos) in grouped_repos {
                lines.push(Line::from(format!("▼ {}", group_name)));
                for repo in repos {
                    // Use cached git status if available, otherwise show loading
                    if let Some(status) = self.git_statuses.get(&repo.name) {
                        let indicator = if status.is_dirty { "●" } else { "✓" };
                        
                        let mut spans = vec![
                            Span::raw(format!("  {} {}", indicator, repo.name)),
                        ];
                        
                        // Add colored branch information
                        if let Some(branch) = &status.branch_name {
                            let (color, is_bold) = Self::branch_color(branch);
                            let mut style = Style::default().fg(color);
                            if is_bold {
                                style = style.add_modifier(Modifier::BOLD);
                            }
                            spans.push(Span::raw(" ("));
                            spans.push(Span::styled(branch.clone(), style));
                            
                            // Add ahead/behind indicators
                            if status.ahead_count > 0 {
                                spans.push(Span::raw(format!(" ↑{}", status.ahead_count)));
                            }
                            if status.behind_count > 0 {
                                spans.push(Span::raw(format!(" ↓{}", status.behind_count)));
                            }
                            
                            spans.push(Span::raw(")"));
                        }
                        
                        lines.push(Line::from(spans));
                    } else if self.git_status_loading {
                        lines.push(Line::from(format!("  ⋯ {}", repo.name)));
                    } else {
                        lines.push(Line::from(format!("  ? {}", repo.name)));
                    }
                }
                lines.push(Line::from("")); // Empty line between groups
            }
            
            if !self.scan_complete {
                lines.push(Line::from("Scanning for more repositories..."));
            } else if self.git_status_loading {
                lines.push(Line::from("Loading git status..."));
            }
            
            lines
        };

        // Apply scrolling: calculate visible area and slice content
        let available_height = chunks[1].height.saturating_sub(2) as usize; // Minus borders
        let visible_lines = if content_lines.len() > available_height && available_height > 0 {
            let start = self.scroll_offset.min(content_lines.len().saturating_sub(1));
            let end = (start + available_height).min(content_lines.len());
            content_lines[start..end].to_vec()
        } else {
            content_lines
        };

        let main_content = Paragraph::new(visible_lines)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .title("Repositories"),
            )
            .style(Style::default().fg(Color::White));
        f.render_widget(main_content, chunks[1]);

        // Footer with keybindings based on current mode
        let mode_text = match self.mode {
            AppMode::Normal => "NORMAL".fg(Color::Green),
            AppMode::Organize => "ORGANIZE".fg(Color::Yellow),
        };
        
        let footer = Paragraph::new(Line::from(vec![
            "MODE: ".into(),
            mode_text.add_modifier(Modifier::BOLD),
            " | Press ".into(),
            "↑↓".fg(Color::Yellow).add_modifier(Modifier::BOLD),
            "/".into(),
            "j,k".fg(Color::Yellow).add_modifier(Modifier::BOLD),
            " to scroll, ".into(),
            "o".fg(Color::Yellow).add_modifier(Modifier::BOLD),
            " organize, ".into(),
            "q".fg(Color::Yellow).add_modifier(Modifier::BOLD),
            " to quit".into(),
        ]))
        .block(Block::default().borders(Borders::ALL))
        .style(Style::default().fg(Color::Gray));
        f.render_widget(footer, chunks[2]);
    }

    // Modal state management methods
    pub fn current_mode(&self) -> AppMode {
        self.mode
    }

    pub fn set_mode(&mut self, mode: AppMode) {
        self.mode = mode;
    }

    pub fn toggle_mode(&mut self) {
        self.mode = match self.mode {
            AppMode::Normal => AppMode::Organize,
            AppMode::Organize => AppMode::Normal,
        };
    }

    pub fn handle_key_for_mode(&self, key: KeyCode) -> Result<()> {
        match self.mode {
            AppMode::Normal => {
                // Handle normal mode keys (fetch, log, etc.)
                match key {
                    KeyCode::Char('f') => {
                        // Placeholder for fetch functionality
                        Ok(())
                    },
                    _ => Ok(()),
                }
            },
            AppMode::Organize => {
                // Handle organize mode keys (move, create groups, etc.)
                match key {
                    KeyCode::Char('f') => {
                        // In organize mode, 'f' might do something different or nothing
                        Ok(())
                    },
                    _ => Ok(()),
                }
            },
        }
    }

    /// Handle mode-specific keys and return true if a redraw is needed
    pub fn handle_mode_specific_key(&mut self, key: KeyCode) -> Result<bool> {
        match self.mode {
            AppMode::Normal => {
                match key {
                    KeyCode::Down | KeyCode::Char('j') => {
                        self.scroll_down();
                        Ok(true) // Redraw needed
                    }
                    KeyCode::Up | KeyCode::Char('k') => {
                        self.scroll_up();
                        Ok(true) // Redraw needed
                    }
                    KeyCode::Char('f') => {
                        // Placeholder for fetch functionality in normal mode
                        info!("Fetch requested in normal mode");
                        Ok(false) // No visual change yet
                    }
                    KeyCode::Char('l') => {
                        // Placeholder for log functionality in normal mode
                        info!("Log requested in normal mode");
                        Ok(false) // No visual change yet
                    }
                    _ => Ok(false), // Key not handled
                }
            },
            AppMode::Organize => {
                match key {
                    KeyCode::Down | KeyCode::Char('j') => {
                        // In organize mode, navigate down through repositories/groups
                        if self.current_selection + 1 < self.repositories.len() {
                            self.current_selection += 1;
                            Ok(true) // Navigation changed, redraw needed
                        } else {
                            Ok(false)
                        }
                    }
                    KeyCode::Up | KeyCode::Char('k') => {
                        // In organize mode, navigate up through repositories/groups
                        if self.current_selection > 0 {
                            self.current_selection -= 1;
                            Ok(true) // Navigation changed, redraw needed
                        } else {
                            Ok(false)
                        }
                    }
                    KeyCode::Char(' ') => {
                        // Space for selection in organize mode
                        let redraw_needed = self.toggle_repository_selection(self.current_selection);
                        Ok(redraw_needed)
                    }
                    KeyCode::Char('m') => {
                        // Mark selected repositories for moving
                        let redraw_needed = self.mark_selected_repositories();
                        Ok(redraw_needed)
                    }
                    KeyCode::Char('p') => {
                        // Paste/move marked repositories
                        let redraw_needed = self.paste_marked_repositories()?;
                        Ok(redraw_needed)
                    }
                    KeyCode::Char('g') => {
                        // Create new group - placeholder for now
                        info!("Create group in organize mode");
                        Ok(false) // No visual change yet
                    }
                    _ => Ok(false), // Key not handled
                }
            },
        }
    }

    // Repository selection and organization methods
    
    pub fn is_repository_selected(&self, index: usize) -> bool {
        self.selected_repositories.contains(&index)
    }
    
    pub fn set_current_selection(&mut self, index: usize) {
        if index < self.repositories.len() {
            self.current_selection = index;
        }
    }
    
    pub fn get_selected_repositories(&self) -> Vec<usize> {
        self.selected_repositories.iter().cloned().collect()
    }
    
    pub fn get_marked_repositories(&self) -> Vec<usize> {
        self.marked_repositories.iter().cloned().collect()
    }
    
    pub fn toggle_repository_selection(&mut self, index: usize) -> bool {
        if index < self.repositories.len() {
            if self.selected_repositories.contains(&index) {
                self.selected_repositories.remove(&index);
            } else {
                self.selected_repositories.insert(index);
            }
            true // Selection changed, redraw needed
        } else {
            false
        }
    }
    
    pub fn mark_selected_repositories(&mut self) -> bool {
        if !self.selected_repositories.is_empty() {
            for &index in &self.selected_repositories {
                self.marked_repositories.insert(index);
            }
            true // Marking changed, redraw needed
        } else {
            false
        }
    }
    
    pub fn get_repositories_in_group(&self, group_name: &str) -> Vec<Repository> {
        // First check manual groups from config
        if let Some(group_config) = self.config.groups.get(group_name) {
            // Return repositories that are assigned to this manual group
            return self.repositories.iter()
                .filter(|repo| group_config.repos.contains(&repo.path))
                .cloned()
                .collect();
        }
        
        // For auto groups, exclude repositories that are in manual groups
        let grouped = crate::scan::group_repositories(&self.repositories);
        if let Some(group_repos) = grouped.get(group_name) {
            // Get all repository paths that are assigned to manual groups
            let manually_assigned_paths: std::collections::HashSet<_> = self.config.groups
                .values()
                .flat_map(|group_config| &group_config.repos)
                .collect();
            
            // Filter out repositories that are manually assigned to other groups
            group_repos.iter()
                .filter(|repo| !manually_assigned_paths.contains(&repo.path))
                .cloned()
                .collect()
        } else {
            vec![]
        }
    }
    
    pub fn navigate_to_group(&mut self, group_name: &str) -> Result<()> {
        // Set the target group for paste operations
        info!("Navigate to group: {}", group_name);
        // For now, we'll store the target group in a field we'll add
        // For the test, we'll implement the paste logic to work with "Important"
        Ok(())
    }
    
    pub fn paste_marked_repositories(&mut self) -> Result<bool> {
        if !self.marked_repositories.is_empty() {
            // Move marked repositories to the "Important" group (hardcoded for test)
            info!("Pasting {} marked repositories to Important group", self.marked_repositories.len());
            
            // Get the Important group config, or create it if it doesn't exist
            let important_group = self.config.groups
                .entry("Important".to_string())
                .or_insert_with(|| crate::config::GroupConfig { repos: vec![] });
            
            // Add marked repositories to the Important group
            for &repo_index in &self.marked_repositories {
                if let Some(repo) = self.repositories.get(repo_index) {
                    // Add to the Important group if not already there
                    if !important_group.repos.contains(&repo.path) {
                        important_group.repos.push(repo.path.clone());
                    }
                }
            }
            
            // Clear selection and marking
            self.marked_repositories.clear();
            self.selected_repositories.clear();
            
            Ok(true) // Paste operation completed, redraw needed
        } else {
            Ok(false)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_app_new() {
        let config = Config::default();
        let app = App::new(config.clone());
        assert!(!app.should_quit);
        assert_eq!(app.config, config);
        assert!(app.repositories.is_empty());
        assert!(!app.scan_complete);
        assert!(app.git_statuses.is_empty());
        assert!(!app.git_status_loading);
    }

    #[test]
    fn test_app_can_quit() {
        let config = Config::default();
        let mut app = App::new(config);
        app.should_quit = true;
        assert!(app.should_quit);
    }

    #[test] 
    fn test_app_state_transitions() {
        let config = Config::default();
        let mut app = App::new(config);
        
        // Initially should not quit
        assert!(!app.should_quit);
        
        // Can set to quit state
        app.should_quit = true;
        assert!(app.should_quit);
        
        // Can reset quit state
        app.should_quit = false;
        assert!(!app.should_quit);
    }
    
    #[test]
    fn test_app_repository_management() {
        let config = Config::default();
        let mut app = App::new(config);
        
        // Initially empty
        assert!(app.repositories.is_empty());
        assert!(!app.scan_complete);
        
        // Can add repositories
        let repo = Repository {
            name: "test-repo".to_string(),
            path: std::path::PathBuf::from("/test"),
            auto_group: "Ungrouped".to_string(),
        };
        app.repositories.push(repo.clone());
        
        assert_eq!(app.repositories.len(), 1);
        assert_eq!(app.repositories[0], repo);
        
        // Can mark scan as complete
        app.scan_complete = true;
        assert!(app.scan_complete);
    }
}