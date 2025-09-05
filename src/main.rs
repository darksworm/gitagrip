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
    Frame, Terminal,
};
use std::io;
use std::time::Duration;
use tracing::{error, info};

mod cli;
mod config;
mod git;
mod scan;
mod app;

use cli::CliArgs;
use config::Config;
use scan::ScanEvent;
use git::StatusEvent;
use app::App;

impl App {
    fn run<B: Backend>(
        &mut self, 
        terminal: &mut Terminal<B>,
        scan_receiver: Receiver<ScanEvent>,
        status_receiver: Receiver<StatusEvent>,
        status_sender: crossbeam_channel::Sender<StatusEvent>
    ) -> Result<()> {
        let mut git_status_started = false;
        let mut needs_redraw = true; // Initial draw needed
        
        loop {
            // Only redraw if something changed
            if needs_redraw {
                terminal.draw(|f| self.ui(f))?;
                needs_redraw = false;
            }

            // Check for scan events (non-blocking)
            let mut events_received = false;
            while let Ok(event) = scan_receiver.try_recv() {
                events_received = true;
                match event {
                    ScanEvent::RepoDiscovered(repo) => {
                        info!("Discovered repository: {}", repo.name);
                        self.repositories.push(repo);
                    }
                    ScanEvent::ScanCompleted => {
                        info!("Repository scan completed");
                        self.scan_complete = true;
                        // Start git status loading once repository scan is complete
                        if !self.repositories.is_empty() && !git_status_started {
                            self.git_status_loading = true;
                            git_status_started = true;
                            let repos_for_status = self.repositories.clone();
                            let status_sender_clone = status_sender.clone();
                            std::thread::spawn(move || {
                                if let Err(e) = git::compute_statuses_with_events(&repos_for_status, status_sender_clone) {
                                    error!("Background git status failed: {}", e);
                                }
                            });
                        }
                    }
                    ScanEvent::ScanError(err) => {
                        error!("Scan error: {}", err);
                    }
                }
            }
            
            // Check for git status events (non-blocking)
            while let Ok(event) = status_receiver.try_recv() {
                events_received = true;
                match event {
                    StatusEvent::StatusUpdated { repository, status } => {
                        info!("Git status updated for repository: {}", repository);
                        self.git_statuses.insert(repository, status);
                    }
                    StatusEvent::StatusScanCompleted => {
                        info!("Git status scan completed");
                        self.git_status_loading = false;
                    }
                    StatusEvent::StatusError { repository, error } => {
                        error!("Git status error for {}: {}", repository, error);
                    }
                }
            }
            
            // If we received any events, we need to redraw
            if events_received {
                needs_redraw = true;
            }

            // Handle user input with timeout to allow UI updates
            if event::poll(Duration::from_millis(100))? {
                let event = event::read()?;
                match event {
                    Event::Key(key) => {
                        if key.kind == KeyEventKind::Press {
                        // Check if we're in input mode first
                        if self.get_input_mode() != app::InputMode::None {
                            // In input mode - handle text input and special keys
                            match key.code {
                                KeyCode::Char(c) => {
                                    // Add character to input
                                    self.handle_text_input(&c.to_string())?;
                                    needs_redraw = true;
                                }
                                KeyCode::Backspace => {
                                    // Remove last character
                                    let mut current_text = self.get_current_input_text();
                                    current_text.pop();
                                    self.clear_input();
                                    self.handle_text_input(&current_text)?;
                                    needs_redraw = true;
                                }
                                // Let mode-specific handler deal with Enter/Esc in input mode
                                _ => {
                                    if self.current_mode() == app::AppMode::Organize {
                                        // Use simplified organize key handler
                                        if self.handle_organize_key(key.code)? {
                                            needs_redraw = true;
                                        }
                                    } else {
                                        // Use old handler for normal mode
                                        if self.handle_mode_specific_key(key.code)? {
                                            needs_redraw = true;
                                        }
                                    }
                                }
                            }
                        } else {
                            // Not in input mode - handle normal keys
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
                                    // Only quit with Esc if not in organize mode
                                    if self.current_mode() == app::AppMode::Normal {
                                        info!("Escape pressed, quitting");
                                        self.should_quit = true;
                                    } else {
                                        // In organize mode, let simplified handler deal with Esc
                                        if self.handle_organize_key(key.code)? {
                                            needs_redraw = true;
                                        }
                                    }
                                }
                                KeyCode::Char('o') => {
                                    info!("Mode toggle requested");
                                    self.toggle_mode();
                                    needs_redraw = true;
                                }
                                // Handle mode-specific keys
                                _ => {
                                    if self.current_mode() == app::AppMode::Organize {
                                        // Use simplified organize key handler
                                        if self.handle_organize_key(key.code)? {
                                            needs_redraw = true;
                                        }
                                    } else {
                                        // Use old handler for normal mode
                                        if self.handle_mode_specific_key(key.code)? {
                                            needs_redraw = true;
                                        }
                                    }
                                }
                            }
                        }
                    }
                    }
                    Event::Resize(_width, _height) => {
                        // Terminal was resized, force a redraw
                        needs_redraw = true;
                    }
                    _ => {
                        // Other events (mouse, etc.) - ignore for now
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
        // Delegate to the new ui_with_git_status method
        self.ui_with_git_status(f);
    }
}

fn main() -> Result<()> {
    // Initialize tracing with env filter
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    info!("Starting GitaGrip");

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
    let mut app = App::new(config.clone(), None); // Use default config path
    
    // Setup background repository scanning
    let (scan_sender, scan_receiver) = crossbeam_channel::unbounded();
    let (status_sender, status_receiver) = crossbeam_channel::unbounded();
    let base_dir = config.base_dir.clone();
    
    // Spawn background scan
    let scan_sender_clone = scan_sender.clone();
    std::thread::spawn(move || {
        if let Err(e) = scan::scan_repositories_background(base_dir, scan_sender_clone) {
            error!("Background scan failed: {}", e);
        }
    });
    
    // We'll trigger git status loading from within the main loop after scan completes
    // This avoids the competing receiver problem
    
    let res = app.run(&mut terminal, scan_receiver, status_receiver, status_sender);

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

    info!("GitaGrip shut down cleanly");
    Ok(())
}

