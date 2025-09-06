use ratatui::{
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, Clear, List, ListItem, Paragraph, Wrap},
    Frame,
};
use super::model::{TuiModel, ViewMode, InputMode};

/// The View component of MVU - responsible for rendering the model
pub struct TuiView;

impl TuiView {
    /// Render the entire TUI based on the current model state
    pub fn render(model: &TuiModel, frame: &mut Frame) {
        let size = frame.area();
        
        // Update model with current terminal size (this is a bit of a hack but works for now)
        // In a proper implementation, we'd handle this in the update loop
        
        // Main layout
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints([
                Constraint::Length(1), // Title bar
                Constraint::Min(0),    // Main content
                Constraint::Length(3), // Status/input bar
            ])
            .split(size);
        
        // Render title bar
        Self::render_title_bar(model, frame, chunks[0]);
        
        // Render main content based on current mode
        match &model.mode {
            ViewMode::RepoList => Self::render_repo_list(model, frame, chunks[1]),
            ViewMode::Organize => Self::render_organize_view(model, frame, chunks[1]),
            ViewMode::RepoDetails { repo_id } => Self::render_repo_details(model, frame, chunks[1], repo_id),
            ViewMode::CommitLog { repo_id } => Self::render_commit_log(model, frame, chunks[1], repo_id),
            ViewMode::Config => Self::render_config_view(model, frame, chunks[1]),
            ViewMode::Help => Self::render_help_view(model, frame, chunks[1]),
        }
        
        // Render status/input bar
        Self::render_status_bar(model, frame, chunks[2]);
        
        // Render overlays (errors, messages, etc.)
        Self::render_overlays(model, frame, size);
    }
    
    /// Render the title bar
    fn render_title_bar(model: &TuiModel, frame: &mut Frame, area: Rect) {
        let title = match &model.mode {
            ViewMode::RepoList => "GitaGrip - Repository List",
            ViewMode::Organize => "GitaGrip - Organize Repositories",
            ViewMode::RepoDetails { .. } => "GitaGrip - Repository Details",
            ViewMode::CommitLog { .. } => "GitaGrip - Commit Log",
            ViewMode::Config => "GitaGrip - Configuration",
            ViewMode::Help => "GitaGrip - Help",
        };
        
        let scanning_indicator = if model.projection.scanning {
            " [SCANNING...]"
        } else {
            ""
        };
        
        let refreshing_indicator = if model.projection.refreshing_status {
            " [REFRESHING...]"
        } else {
            ""
        };
        
        let title_text = format!("{}{}{}", title, scanning_indicator, refreshing_indicator);
        
        let title_paragraph = Paragraph::new(title_text)
            .style(Style::default().fg(Color::White).bg(Color::Blue))
            .alignment(Alignment::Center);
            
        frame.render_widget(title_paragraph, area);
    }
    
    /// Render the main repository list view
    fn render_repo_list(model: &TuiModel, frame: &mut Frame, area: Rect) {
        let display_repos = model.get_display_repositories();
        
        if display_repos.is_empty() {
            let empty_msg = if model.projection.scanning {
                "Scanning for repositories..."
            } else {
                "No repositories found. Press 's' to scan a directory."
            };
            
            let paragraph = Paragraph::new(empty_msg)
                .style(Style::default().fg(Color::Yellow))
                .alignment(Alignment::Center)
                .wrap(Wrap { trim: true });
                
            frame.render_widget(paragraph, area);
            return;
        }
        
        let mut items = Vec::new();
        let mut item_index = 0;
        
        for (group_name, repos) in display_repos {
            if let Some(group) = group_name {
                // Group header
                let expanded = model.is_group_expanded(&group);
                let expand_indicator = if expanded { "▼" } else { "▶" };
                let group_line = Line::from(vec![
                    Span::styled(expand_indicator, Style::default().fg(Color::Blue)),
                    Span::raw(" "),
                    Span::styled(group.clone(), Style::default().fg(Color::Blue).add_modifier(Modifier::BOLD)),
                    Span::raw(format!(" ({} repos)", repos.len())),
                ]);
                
                let style = if item_index == model.ui_state.cursor_position {
                    Style::default().bg(Color::DarkGray)
                } else {
                    Style::default()
                };
                
                items.push(ListItem::new(group_line).style(style));
                item_index += 1;
                
                // Group repositories (if expanded)
                if expanded {
                    for repo in repos {
                        let status_indicator = Self::get_repo_status_indicator(model, repo);
                        let mark_indicator = if let Some(repo_id) = model.projection.repositories.iter()
                            .find(|(_, meta)| meta.path == repo.path)
                            .map(|(id, _)| id)
                            .and_then(|id| if model.is_repo_marked(id) { Some("✓") } else { None })
                        {
                            repo_id
                        } else {
                            " "
                        };
                        
                        let repo_line = Line::from(vec![
                            Span::raw("  "), // Indent for group member
                            Span::raw(mark_indicator),
                            Span::raw(" "),
                            Span::styled(status_indicator, Self::get_status_color(model, repo)),
                            Span::raw(" "),
                            Span::styled(&repo.name, Style::default()),
                        ]);
                        
                        let style = if item_index == model.ui_state.cursor_position {
                            Style::default().bg(Color::DarkGray)
                        } else {
                            Style::default()
                        };
                        
                        items.push(ListItem::new(repo_line).style(style));
                        item_index += 1;
                    }
                }
            }
        }
        
        let list = List::new(items)
            .block(Block::default().borders(Borders::ALL).title("Repositories"))
            .highlight_style(Style::default().add_modifier(Modifier::BOLD));
            
        frame.render_widget(list, area);
    }
    
    /// Render the organize view
    fn render_organize_view(model: &TuiModel, frame: &mut Frame, area: Rect) {
        let chunks = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Percentage(70), Constraint::Percentage(30)])
            .split(area);
        
        // Left side: repository list (similar to main view but with selection)
        Self::render_repo_list(model, frame, chunks[0]);
        
        // Right side: organization panel
        Self::render_organization_panel(model, frame, chunks[1]);
    }
    
    /// Render the organization panel
    fn render_organization_panel(model: &TuiModel, frame: &mut Frame, area: Rect) {
        let mut lines = vec![
            Line::from(Span::styled("Organization", Style::default().add_modifier(Modifier::BOLD))),
            Line::from(""),
            Line::from("Controls:"),
            Line::from("  Space - Mark/unmark repository"),
            Line::from("  n - Create new group"),
            Line::from("  r - Rename group"),
            Line::from("  d - Delete group"),
            Line::from("  q - Exit organize mode"),
            Line::from(""),
        ];
        
        // Show marked repositories
        if !model.ui_state.marked_repos.is_empty() {
            lines.push(Line::from(Span::styled("Marked repositories:", Style::default().add_modifier(Modifier::BOLD))));
            for repo_id in &model.ui_state.marked_repos {
                if let Some(repo_meta) = model.projection.repositories.get(repo_id) {
                    lines.push(Line::from(format!("  • {}", repo_meta.name)));
                }
            }
        }
        
        let paragraph = Paragraph::new(lines)
            .block(Block::default().borders(Borders::ALL).title("Organize"))
            .wrap(Wrap { trim: true });
            
        frame.render_widget(paragraph, area);
    }
    
    /// Render repository details view
    fn render_repo_details(model: &TuiModel, frame: &mut Frame, area: Rect, repo_id: &gitagrip_core::domain::RepoId) {
        let repo_meta = model.projection.repositories.get(repo_id);
        let repo_status = model.projection.statuses.get(repo_id);
        
        let mut lines = Vec::new();
        
        if let Some(meta) = repo_meta {
            lines.push(Line::from(Span::styled(&meta.name, Style::default().add_modifier(Modifier::BOLD))));
            lines.push(Line::from(format!("Path: {}", meta.path.display())));
            lines.push(Line::from(format!("Group: {}", meta.auto_group)));
            lines.push(Line::from(""));
        }
        
        if let Some(status) = repo_status {
            lines.push(Line::from(Span::styled("Git Status:", Style::default().add_modifier(Modifier::BOLD))));
            
            if let Some(branch) = &status.branch_name {
                lines.push(Line::from(format!("Branch: {}", branch)));
            }
            
            let status_text = if status.is_dirty {
                "Dirty"
            } else {
                "Clean"
            };
            let status_color = if status.is_dirty { Color::Red } else { Color::Green };
            lines.push(Line::from(vec![
                Span::raw("Status: "),
                Span::styled(status_text, Style::default().fg(status_color)),
            ]));
            
            if status.ahead_count > 0 || status.behind_count > 0 {
                lines.push(Line::from(format!("Ahead: {} | Behind: {}", status.ahead_count, status.behind_count)));
            }
            
            if !status.last_commit_summary.is_empty() {
                lines.push(Line::from(""));
                lines.push(Line::from("Last commit:"));
                lines.push(Line::from(format!("  {}", status.last_commit_summary)));
            }
        }
        
        lines.push(Line::from(""));
        lines.push(Line::from("Controls:"));
        lines.push(Line::from("  l - Show commit log"));
        lines.push(Line::from("  f - Fetch from remote"));
        lines.push(Line::from("  o - Open in external app"));
        lines.push(Line::from("  b - Back to list"));
        
        let paragraph = Paragraph::new(lines)
            .block(Block::default().borders(Borders::ALL).title("Repository Details"))
            .wrap(Wrap { trim: true });
            
        frame.render_widget(paragraph, area);
    }
    
    /// Render commit log view
    fn render_commit_log(_model: &TuiModel, frame: &mut Frame, area: Rect, _repo_id: &gitagrip_core::domain::RepoId) {
        // TODO: Implement commit log rendering
        let placeholder = Paragraph::new("Commit log view not implemented yet.\n\nPress 'b' to go back.")
            .block(Block::default().borders(Borders::ALL).title("Commit Log"))
            .alignment(Alignment::Center)
            .wrap(Wrap { trim: true });
            
        frame.render_widget(placeholder, area);
    }
    
    /// Render config view
    fn render_config_view(_model: &TuiModel, frame: &mut Frame, area: Rect) {
        let placeholder = Paragraph::new("Configuration view not implemented yet.\n\nPress 'b' to go back.")
            .block(Block::default().borders(Borders::ALL).title("Configuration"))
            .alignment(Alignment::Center)
            .wrap(Wrap { trim: true });
            
        frame.render_widget(placeholder, area);
    }
    
    /// Render help view
    fn render_help_view(_model: &TuiModel, frame: &mut Frame, area: Rect) {
        let help_text = vec![
            Line::from(Span::styled("GitaGrip Help", Style::default().add_modifier(Modifier::BOLD))),
            Line::from(""),
            Line::from(Span::styled("Navigation:", Style::default().add_modifier(Modifier::UNDERLINED))),
            Line::from("  ↑/k - Move up"),
            Line::from("  ↓/j - Move down"),
            Line::from("  ←/h - Collapse group / Go back"),
            Line::from("  →/l - Expand group / Enter"),
            Line::from(""),
            Line::from(Span::styled("Actions:", Style::default().add_modifier(Modifier::UNDERLINED))),
            Line::from("  r - Refresh repository status"),
            Line::from("  f - Fetch all repositories"),
            Line::from("  F - Fetch all repositories (with prune)"),
            Line::from("  s - Scan directory for repositories"),
            Line::from("  / - Search repositories"),
            Line::from("  o - Enter organize mode"),
            Line::from(""),
            Line::from(Span::styled("Organize Mode:", Style::default().add_modifier(Modifier::UNDERLINED))),
            Line::from("  Space - Mark/unmark repository"),
            Line::from("  n - Create new group"),
            Line::from("  q - Exit organize mode"),
            Line::from(""),
            Line::from(Span::styled("Global:", Style::default().add_modifier(Modifier::UNDERLINED))),
            Line::from("  ? - Show this help"),
            Line::from("  Ctrl+C / Esc / q - Quit"),
            Line::from("  F5 - Refresh all"),
            Line::from(""),
            Line::from("Press any key to close help..."),
        ];
        
        let help = Paragraph::new(help_text)
            .block(Block::default().borders(Borders::ALL).title("Help"))
            .wrap(Wrap { trim: true });
            
        frame.render_widget(help, area);
    }
    
    /// Render the status/input bar at the bottom
    fn render_status_bar(model: &TuiModel, frame: &mut Frame, area: Rect) {
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Length(1), Constraint::Length(1), Constraint::Length(1)])
            .split(area);
        
        // Input line
        if model.input.mode != InputMode::None {
            let input_text = format!("{} {}", model.input.prompt, model.input.text);
            let input_paragraph = Paragraph::new(input_text)
                .style(Style::default().fg(Color::Yellow));
            frame.render_widget(input_paragraph, chunks[0]);
        } else {
            frame.render_widget(Paragraph::new(""), chunks[0]);
        }
        
        // Status line
        let status_text = Self::build_status_text(model);
        let status_paragraph = Paragraph::new(status_text)
            .style(Style::default().fg(Color::White).bg(Color::DarkGray));
        frame.render_widget(status_paragraph, chunks[1]);
        
        // Key hints
        let hints = Self::get_key_hints(model);
        let hints_paragraph = Paragraph::new(hints)
            .style(Style::default().fg(Color::Gray));
        frame.render_widget(hints_paragraph, chunks[2]);
    }
    
    /// Build status text for the status bar
    fn build_status_text(model: &TuiModel) -> String {
        let repo_count = model.projection.repositories.len();
        let group_count = model.projection.repositories_by_auto_group().len();
        
        let mut status_parts = vec![
            format!("{} repos", repo_count),
            format!("{} groups", group_count),
        ];
        
        if !model.ui_state.marked_repos.is_empty() {
            status_parts.push(format!("{} marked", model.ui_state.marked_repos.len()));
        }
        
        if let Some(selected_repo) = &model.ui_state.selected_repo {
            if let Some(repo_meta) = model.projection.repositories.get(selected_repo) {
                status_parts.push(format!("Selected: {}", repo_meta.name));
            }
        }
        
        status_parts.join(" | ")
    }
    
    /// Get key hints for current mode
    fn get_key_hints(model: &TuiModel) -> String {
        match &model.mode {
            ViewMode::RepoList => "? Help | o Organize | r Refresh | f Fetch | s Scan | q Quit",
            ViewMode::Organize => "Space Mark | n New Group | q Exit Organize",
            ViewMode::RepoDetails { .. } => "l Log | f Fetch | o Open | b Back",
            ViewMode::CommitLog { .. } => "j/k Navigate | b Back",
            ViewMode::Config => "b Back",
            ViewMode::Help => "Any key to close",
        }.to_string()
    }
    
    /// Render overlay messages (errors, notifications, etc.)
    fn render_overlays(model: &TuiModel, frame: &mut Frame, area: Rect) {
        // Render error messages
        if !model.errors.is_empty() {
            Self::render_error_overlay(model, frame, area);
        }
        
        // TODO: Render other overlays (notifications, confirmations, etc.)
    }
    
    /// Render error overlay
    fn render_error_overlay(model: &TuiModel, frame: &mut Frame, area: Rect) {
        let popup_area = Self::centered_rect(60, 20, area);
        
        frame.render_widget(Clear, popup_area);
        
        let error_text: Vec<Line> = model.errors.iter()
            .map(|error| Line::from(error.as_str()))
            .collect();
        
        let error_popup = Paragraph::new(error_text)
            .block(Block::default().borders(Borders::ALL).title("Errors"))
            .style(Style::default().fg(Color::Red))
            .wrap(Wrap { trim: true });
            
        frame.render_widget(error_popup, popup_area);
    }
    
    /// Helper to create centered rectangle
    fn centered_rect(percent_x: u16, percent_y: u16, r: Rect) -> Rect {
        let popup_layout = Layout::default()
            .direction(Direction::Vertical)
            .constraints([
                Constraint::Percentage((100 - percent_y) / 2),
                Constraint::Percentage(percent_y),
                Constraint::Percentage((100 - percent_y) / 2),
            ])
            .split(r);
        
        Layout::default()
            .direction(Direction::Horizontal)
            .constraints([
                Constraint::Percentage((100 - percent_x) / 2),
                Constraint::Percentage(percent_x),
                Constraint::Percentage((100 - percent_x) / 2),
            ])
            .split(popup_layout[1])[1]
    }
    
    /// Get repository status indicator character
    fn get_repo_status_indicator(model: &TuiModel, repo: &gitagrip_core::domain::RepoMeta) -> &'static str {
        // Find the repository status
        if let Some((repo_id, _)) = model.projection.repositories.iter()
            .find(|(_, meta)| meta.path == repo.path)
        {
            if let Some(status) = model.projection.statuses.get(repo_id) {
                if status.is_dirty {
                    "●" // Dirty
                } else if status.ahead_count > 0 || status.behind_count > 0 {
                    "⋯" // Ahead/behind
                } else {
                    "✓" // Clean and up to date
                }
            } else {
                "?" // Status unknown
            }
        } else {
            "?" // Repository not found
        }
    }
    
    /// Get color for repository status
    fn get_status_color(_model: &TuiModel, _repo: &gitagrip_core::domain::RepoMeta) -> Style {
        // TODO: Implement status-based coloring
        Style::default()
    }
}