use crate::config::Config;
use crate::scan::Repository;
use crate::git;
use std::collections::HashMap;

pub struct App {
    pub should_quit: bool,
    pub config: Config,
    pub repositories: Vec<Repository>,
    pub scan_complete: bool,
    pub git_statuses: HashMap<String, git::RepoStatus>,
    pub git_status_loading: bool,
    pub scroll_offset: usize,
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

        // Footer with keybindings
        let footer = Paragraph::new(Line::from(vec![
            "Press ".into(),
            "↑↓".fg(Color::Yellow).add_modifier(Modifier::BOLD),
            "/".into(),
            "j,k".fg(Color::Yellow).add_modifier(Modifier::BOLD),
            " to scroll, ".into(),
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