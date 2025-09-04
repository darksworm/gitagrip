use crate::config::Config;
use crate::scan::Repository;
use crate::git;
use std::collections::HashMap;

#[derive(Debug)]
pub struct RepositoryDisplayInfo {
    pub status_indicator: String,
    pub is_dirty: bool,
    pub branch_info: String,
}

pub struct App {
    pub should_quit: bool,
    pub config: Config,
    pub repositories: Vec<Repository>,
    pub scan_complete: bool,
    pub git_statuses: HashMap<String, git::RepoStatus>,
    pub git_status_loading: bool,
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
        }
    }

    pub fn prepare_repository_display_with_status(
        &self,
        repositories: &[Repository],
        statuses: &HashMap<String, git::RepoStatus>,
    ) -> HashMap<String, RepositoryDisplayInfo> {
        let mut display_data = HashMap::new();
        
        for repo in repositories {
            let display_info = if let Some(status) = statuses.get(&repo.name) {
                let status_indicator = if status.is_dirty { "●" } else { "✓" }.to_string();
                
                let mut branch_parts: Vec<String> = Vec::new();
                if let Some(ref branch_name) = status.branch_name {
                    branch_parts.push(branch_name.clone());
                }
                if status.ahead_count > 0 {
                    branch_parts.push(format!("↑{}", status.ahead_count));
                }
                if status.behind_count > 0 {
                    branch_parts.push(format!("↓{}", status.behind_count));
                }
                
                RepositoryDisplayInfo {
                    status_indicator,
                    is_dirty: status.is_dirty,
                    branch_info: branch_parts.join(" "),
                }
            } else {
                RepositoryDisplayInfo {
                    status_indicator: "?".to_string(),
                    is_dirty: false,
                    branch_info: "Loading...".to_string(),
                }
            };
            
            display_data.insert(repo.name.clone(), display_info);
        }
        
        display_data
    }

    pub fn render_repository_list_with_status(
        &self,
        display_data: &HashMap<String, RepositoryDisplayInfo>,
    ) -> String {
        let mut content = Vec::new();
        
        for (repo_name, display_info) in display_data {
            let line = format!(
                "{} {} {}",
                display_info.status_indicator,
                repo_name,
                display_info.branch_info
            );
            content.push(line);
        }
        
        content.join("\n")
    }

    pub fn create_test_ui_frame(&self) -> String {
        // Simple mock UI frame for testing performance
        format!("Mock UI Frame - {} repositories", self.repositories.len())
    }

    pub fn ui_with_git_status(&self, f: &mut ratatui::Frame) {
        use ratatui::{
            layout::{Constraint, Direction, Layout},
            prelude::Stylize,
            style::{Color, Modifier, Style},
            text::Line,
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
        let title_text = format!("YARG - Yet Another Repo Grouper    {}", 
                                self.config.base_dir.display());
        let title = Paragraph::new(title_text)
            .block(Block::default().borders(Borders::ALL))
            .style(Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD));
        f.render_widget(title, chunks[0]);

        // Main content - show repositories with git status and grouping
        let content_text = if self.repositories.is_empty() {
            if self.scan_complete {
                "No Git repositories found in base directory.".to_string()
            } else {
                "Scanning for repositories...".to_string()
            }
        } else {
            // Restore grouping functionality
            let grouped_repos = crate::scan::group_repositories(&self.repositories);
            let mut text = Vec::new();
            
            for (group_name, repos) in grouped_repos {
                text.push(format!("▼ {}", group_name));
                for repo in repos {
                    // Use cached git status if available, otherwise show loading
                    let (status_indicator, branch_info) = if let Some(status) = self.git_statuses.get(&repo.name) {
                        let indicator = if status.is_dirty { "●" } else { "✓" };
                        
                        let mut parts = Vec::new();
                        if let Some(branch) = &status.branch_name {
                            parts.push(branch.clone());
                        }
                        if status.ahead_count > 0 {
                            parts.push(format!("↑{}", status.ahead_count));
                        }
                        if status.behind_count > 0 {
                            parts.push(format!("↓{}", status.behind_count));
                        }
                        let branch_info = if parts.is_empty() { 
                            "".to_string() 
                        } else { 
                            format!(" ({})", parts.join(" ")) 
                        };
                        
                        (indicator, branch_info)
                    } else if self.git_status_loading {
                        ("⋯", " (loading...)".to_string())
                    } else {
                        ("?", "".to_string())
                    };
                    
                    text.push(format!("  {} {} ({}){}", 
                        status_indicator, 
                        repo.name, 
                        repo.path.display(),
                        branch_info
                    ));
                }
                text.push("".to_string()); // Empty line between groups
            }
            
            if !self.scan_complete {
                text.push("Scanning for more repositories...".to_string());
            } else if self.git_status_loading {
                text.push("Loading git status...".to_string());
            }
            
            text.join("\n")
        };

        let main_content = Paragraph::new(content_text)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .title("Repositories"),
            )
            .style(Style::default().fg(Color::White));
        f.render_widget(main_content, chunks[1]);

        // Footer with keybindings
        let footer = Paragraph::new(Line::from(vec![
            "Press ".into(),
            "q".fg(Color::Yellow).add_modifier(Modifier::BOLD),
            " to quit".into(),
        ]))
        .block(Block::default().borders(Borders::ALL))
        .style(Style::default().fg(Color::Gray));
        f.render_widget(footer, chunks[2]);
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