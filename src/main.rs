use anyhow::Result;
use clap::Parser;
use crossbeam_channel::Receiver;
use crossterm::{
    event::{self, DisableMouseCapture, EnableMouseCapture, Event, KeyCode, KeyEventKind, KeyModifiers},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{
    backend::{Backend, CrosstermBackend},
    layout::{Constraint, Direction, Layout},
    prelude::Stylize,
    style::{Color, Modifier, Style},
    text::Line,
    widgets::{Block, Borders, Paragraph},
    Frame, Terminal,
};
use std::io;
use std::time::Duration;
use tracing::{error, info};

mod cli;
mod config;
mod scan;

use cli::CliArgs;
use config::Config;
use scan::{Repository, ScanEvent};

struct App {
    should_quit: bool,
    config: Config,
    repositories: Vec<Repository>,
    scan_complete: bool,
}

impl App {
    fn new(config: Config) -> App {
        App { 
            should_quit: false,
            config,
            repositories: Vec::new(),
            scan_complete: false,
        }
    }
}

impl App {
    fn run<B: Backend>(
        &mut self, 
        terminal: &mut Terminal<B>,
        scan_receiver: Receiver<ScanEvent>
    ) -> Result<()> {
        loop {
            terminal.draw(|f| self.ui(f))?;

            // Check for scan events (non-blocking)
            while let Ok(event) = scan_receiver.try_recv() {
                match event {
                    ScanEvent::RepoDiscovered(repo) => {
                        info!("Discovered repository: {}", repo.name);
                        self.repositories.push(repo);
                    }
                    ScanEvent::ScanCompleted => {
                        info!("Repository scan completed");
                        self.scan_complete = true;
                    }
                    ScanEvent::ScanError(err) => {
                        error!("Scan error: {}", err);
                    }
                }
            }

            // Handle user input with timeout to allow UI updates
            if event::poll(Duration::from_millis(100))? {
                if let Event::Key(key) = event::read()? {
                    if key.kind == KeyEventKind::Press {
                        match key.code {
                            KeyCode::Char('q') => {
                                info!("Quit requested by user");
                                self.should_quit = true;
                            }
                            KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                                info!("Ctrl+C pressed, quitting");
                                self.should_quit = true;
                            }
                            KeyCode::Esc => {
                                info!("Escape pressed, quitting");
                                self.should_quit = true;
                            }
                            _ => {}
                        }
                    }
                }
            }

            if self.should_quit {
                break;
            }
        }
        Ok(())
    }

    fn ui(&self, f: &mut Frame) {
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

        // Main content - show repositories
        let content_text = if self.repositories.is_empty() {
            if self.scan_complete {
                "No Git repositories found in base directory.".to_string()
            } else {
                "Scanning for repositories...".to_string()
            }
        } else {
            let grouped_repos = scan::group_repositories(&self.repositories);
            let mut text = Vec::new();
            
            for (group_name, repos) in grouped_repos {
                text.push(format!("▼ {}", group_name));
                for repo in repos {
                    text.push(format!("  {} ({})", repo.name, repo.path.display()));
                }
                text.push("".to_string()); // Empty line between groups
            }
            
            if !self.scan_complete {
                text.push("Scanning for more repositories...".to_string());
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

fn main() -> Result<()> {
    // Initialize tracing with env filter
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    info!("Starting YARG - Yet Another Repo Grouper");

    // Parse CLI arguments
    let cli_args = CliArgs::parse();
    
    // Load config (with CLI overrides)
    let config = Config::from_cli_and_file(cli_args, None)?;
    info!("Loaded config with base_dir: {}", config.base_dir.display());
    
    // Setup terminal
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    // Create app and background scanning
    let mut app = App::new(config.clone());
    
    // Setup background repository scanning
    let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();
    let base_dir = config.base_dir.clone();
    
    // Spawn background scan
    std::thread::spawn(move || {
        if let Err(e) = scan::scan_repositories_background(base_dir, scan_sender) {
            error!("Background scan failed: {}", e);
        }
    });
    
    let res = app.run(&mut terminal, scan_receiver);

    // Restore terminal
    disable_raw_mode()?;
    execute!(
        terminal.backend_mut(),
        LeaveAlternateScreen,
        DisableMouseCapture
    )?;
    terminal.show_cursor()?;

    if let Err(err) = res {
        error!("Application error: {}", err);
        println!("Error: {}", err);
    }

    info!("YARG shut down cleanly");
    Ok(())
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

    // Note: Testing the actual TUI rendering and key handling would require 
    // more complex integration tests with mock terminals, which we'll add 
    // as we build more functionality
}
