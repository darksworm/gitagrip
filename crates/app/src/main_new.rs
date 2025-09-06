// New main.rs implementing the hexagonal architecture with MVU TUI
// This is the composition root that wires everything together

use anyhow::Result;
use clap::Parser;
use crossterm::{
    event::{self, Event, KeyEventKind},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use gitagrip_core::ports::AppConfig;
use gitagrip_core::app::Command;
use ratatui::{
    backend::CrosstermBackend,
    Terminal,
};
use std::io;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;
use tracing::{error, info};

// Import all the components we've built
use crate::adapters::{
    git::GitAdapter,
    discovery::FsDiscoveryAdapter,
    persistence::FileConfigStore,
};
use crate::services::app_service::AppService;
use crate::tui::{TuiModel, TuiView, TuiUpdate, TuiMessage};
use crate::cli::CliArgs;

/// The main application struct that coordinates everything
/// This follows the hexagonal architecture pattern
pub struct GitaGripApp {
    /// The core application service (hexagonal core)
    app_service: AppService,
    
    /// The TUI model (MVU pattern)
    tui_model: TuiModel,
    
    /// Terminal for rendering
    terminal: Terminal<CrosstermBackend<io::Stdout>>,
    
    /// Event receiver from the app service
    event_rx: mpsc::UnboundedReceiver<gitagrip_core::domain::Event>,
    
    /// Command sender to the app service
    command_tx: mpsc::UnboundedSender<gitagrip_core::app::Command>,
    
    /// App configuration
    config: AppConfig,
}

impl GitaGripApp {
    /// Create a new GitaGrip application instance
    /// This is the composition root - where dependency injection happens
    pub async fn new() -> Result<Self> {
        info!("Initializing GitaGrip application");
        
        // Parse CLI arguments
        let cli_args = CliArgs::parse();
        
        // Create adapters (dependency injection)
        let git_adapter: Arc<dyn gitagrip_core::ports::GitPort> = Arc::new(GitAdapter::new());
        let discovery_adapter: Arc<dyn gitagrip_core::ports::DiscoveryPort> = Arc::new(FsDiscoveryAdapter::new());
        let config_store: Arc<dyn gitagrip_core::ports::ConfigStore> = Arc::new(FileConfigStore::new()?);
        
        // Load configuration
        let config = if let Some(base_dir) = cli_args.base_dir {
            let mut config = config_store.load()?;
            config.base_dir = base_dir; // CLI overrides config file
            config
        } else {
            config_store.load()?
        };
        
        info!("Loaded config with base_dir: {}", config.base_dir.display());
        
        // Create the application service (hexagonal core) with event receiver
        let (app_service, event_rx, command_tx) = AppService::new(git_adapter, discovery_adapter, config_store);
        
        // Initialize terminal
        enable_raw_mode()?;
        let mut stdout = io::stdout();
        execute!(stdout, EnterAlternateScreen)?;
        let backend = CrosstermBackend::new(stdout);
        let terminal = Terminal::new(backend)?;
        
        // Create TUI model and initialize with scan command
        let tui_model = TuiModel::new();
        
        // Trigger initial repository scan
        info!("Starting initial repository scan of {}", config.base_dir.display());
        // This will be handled by the app service once it starts
        
        Ok(Self {
            app_service,
            tui_model,
            terminal,
            event_rx,
            command_tx,
            config,
        })
    }
    
    /// Run the application
    pub async fn run(self) -> Result<()> {
        info!("Starting GitaGrip application");
        
        // Destructure self to take ownership of parts
        let GitaGripApp {
            app_service,
            mut tui_model,
            mut terminal,
            event_rx,
            command_tx,
            config,
        } = self;
        
        // Start the application service in the background
        let mut app_service = app_service;
        let app_service_handle = tokio::spawn(async move {
            app_service.start(config).await
        });
        
        // Run the main application loop
        let result = run_main_loop(&mut tui_model, &mut terminal, event_rx, command_tx).await;
        
        // Clean shutdown
        shutdown(&mut terminal).await?;
        
        // Wait for app service to finish
        if let Err(e) = app_service_handle.await {
            error!("App service task failed: {:?}", e);
        }
        
        result
    }
}

/// Main application loop - coordinates TUI and app service
async fn run_main_loop(
    tui_model: &mut TuiModel,
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    mut event_rx: mpsc::UnboundedReceiver<gitagrip_core::domain::Event>,
    command_tx: mpsc::UnboundedSender<Command>,
) -> Result<()> {
        let mut last_render = std::time::Instant::now();
        let render_interval = Duration::from_millis(16); // ~60 FPS
        let mut needs_redraw = true;
        
        loop {
            // Handle events from the app service
            let mut events_received = false;
            while let Ok(event) = event_rx.try_recv() {
                info!("Received event from app service: {:?}", event);
                tui_model.apply_event(&event);
                events_received = true;
            }
            
            // If we received events, we need to redraw
            if events_received {
                needs_redraw = true;
            }
            
            // Handle user input
            if event::poll(Duration::from_millis(10))? {
                if let Event::Key(key_event) = event::read()? {
                    if key_event.kind == KeyEventKind::Press {
                        let message = TuiUpdate::handle_key(
                            tui_model,
                            key_event.code,
                            key_event.modifiers,
                        )?;
                        
                        // Process the message from TUI
                        match message {
                            TuiMessage::Command(cmd) => {
                                info!("Sending command to app service: {:?}", cmd);
                                
                                // Check for quit command
                                if matches!(cmd, Command::Quit) {
                                    tui_model.should_quit = true;
                                } else {
                                    // Send command to app service via channel
                                    if let Err(e) = command_tx.send(cmd) {
                                        error!("Failed to send command: {}", e);
                                    }
                                }
                            }
                            TuiMessage::Event(event) => {
                                info!("Processing TUI event: {:?}", event);
                                tui_model.apply_event(&event);
                            }
                            TuiMessage::None => {
                                // No action needed
                            }
                        }
                        
                        needs_redraw = true;
                    }
                }
            }
            
            // Handle terminal resize
            if let Ok(size) = terminal.size() {
                let _ = TuiUpdate::handle_resize(tui_model, size.width, size.height)?;
            }
            
            // Check if we should quit
            if tui_model.should_quit {
                info!("Quit requested, exiting main loop");
                break;
            }
            
            // Render at regular intervals or when needed
            if needs_redraw || last_render.elapsed() >= render_interval {
                render(terminal, tui_model)?;
                last_render = std::time::Instant::now();
                needs_redraw = false;
            }
            
            // Small sleep to prevent busy waiting
            tokio::time::sleep(Duration::from_millis(1)).await;
        }
        
        Ok(())
    }
/// Render the TUI
fn render(
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    tui_model: &TuiModel,
) -> Result<()> {
    terminal.draw(|frame| {
        TuiView::render(tui_model, frame);
    })?;
    
    Ok(())
}

/// Clean shutdown
async fn shutdown(terminal: &mut Terminal<CrosstermBackend<io::Stdout>>) -> Result<()> {
    info!("Shutting down GitaGrip");
    
    // Restore terminal
    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;
    
    Ok(())
}

/// The new main function using the hexagonal architecture
pub async fn main_new() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    info!("Starting GitaGrip with hexagonal architecture");
    
    // Create and run the application
    let app = GitaGripApp::new().await?;
    
    if let Err(e) = app.run().await {
        error!("Application error: {}", e);
        eprintln!("Error: {}", e);
        std::process::exit(1);
    }
    
    info!("GitaGrip shut down cleanly");
    Ok(())
}