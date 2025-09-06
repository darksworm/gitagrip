use anyhow::Result;
use gitagrip_core::app::{Command, ReadProjection};
use gitagrip_core::domain::{Event, RepoId};
use gitagrip_core::ports::{AppConfig, ConfigStore, DiscoverReq, DiscoveryPort, GitPort};
use std::sync::Arc;
use tokio::sync::mpsc;
use tokio::task::JoinSet;
use tracing::{error, info};

/// The main application service that coordinates all operations
/// This is the heart of the hexagonal architecture - it uses ports (traits)
/// and communicates via the event bus
pub struct AppService {
    // Ports (dependency injection)
    git_port: Arc<dyn GitPort>,
    discovery_port: Arc<dyn DiscoveryPort>,
    config_store: Arc<dyn ConfigStore>,
    
    // Event bus
    event_tx: mpsc::UnboundedSender<Event>,
    event_rx: mpsc::UnboundedReceiver<Event>,
    
    // External event sender (for TUI)
    event_tx_external: mpsc::UnboundedSender<Event>,
    
    // Command receiver
    command_rx: mpsc::UnboundedReceiver<Command>,
    
    // Read projection for queries
    projection: ReadProjection,
    
    // Background task management
    tasks: JoinSet<Result<()>>,
}

impl AppService {
    pub fn new(
        git_port: Arc<dyn GitPort>,
        discovery_port: Arc<dyn DiscoveryPort>,
        config_store: Arc<dyn ConfigStore>,
    ) -> (Self, mpsc::UnboundedReceiver<Event>, mpsc::UnboundedSender<Command>) {
        let (event_tx, event_rx) = mpsc::unbounded_channel();
        let (event_tx_external, event_rx_external) = mpsc::unbounded_channel(); 
        let (command_tx, command_rx) = mpsc::unbounded_channel();
        
        let service = Self {
            git_port,
            discovery_port,
            config_store,
            event_tx,
            event_rx,
            event_tx_external,
            command_rx,
            projection: ReadProjection::default(),
            tasks: JoinSet::new(),
        };
        
        (service, event_rx_external, command_tx)
    }
    
    /// Get a clone of the event sender for external use (TUI, etc.)
    pub fn event_sender(&self) -> mpsc::UnboundedSender<Event> {
        self.event_tx.clone()
    }
    
    /// Get the current read projection (for UI queries)
    pub fn projection(&self) -> &ReadProjection {
        &self.projection
    }
    
    /// Start the application service
    pub async fn start(&mut self, config: AppConfig) -> Result<()> {
        info!("Starting AppService");
        
        // Start background repository discovery
        self.start_discovery(&config).await?;
        
        // Run the main event loop
        self.run_event_loop().await
    }
    
    /// Handle a command (CQRS Command side)
    pub async fn handle_command(&mut self, cmd: Command) -> Result<()> {
        match cmd {
            Command::Rescan { base } => {
                info!("Starting rescan of {}", base.display());
                self.start_discovery_for_path(base).await?;
            }
            Command::RefreshStatus { ids } => {
                info!("Refreshing status for {} repositories", ids.len());
                for id in ids {
                    self.refresh_repository_status(id).await?;
                }
            }
            Command::FetchAll { prune } => {
                info!("Fetching all repositories (prune: {})", prune);
                let repo_ids: Vec<_> = self.projection.repositories.keys().cloned().collect();
                for id in repo_ids {
                    self.fetch_repository(id, "origin".to_string(), prune).await?;
                }
            }
            Command::OpenRepo { id } => {
                info!("Opening repository {} in external application", id.0);
                // TODO: Implement external app opening
                let _ = self.event_tx.send(Event::Error {
                    id: Some(id),
                    msg: "External app opening not implemented yet".to_string(),
                });
            }
            Command::ToggleGroup { name } => {
                info!("Toggling group: {}", name);
                // TODO: Implement group toggling
            }
            Command::ShowLog { id, range, limit } => {
                info!("Loading log for repository {} (limit: {})", id.0, limit);
                let range_str = range.unwrap_or_else(|| "HEAD".to_string());
                self.load_repository_log(id, range_str, limit).await?;
            }
            Command::Quit => {
                info!("Quit command received");
                let _ = self.event_tx.send(Event::QuitRequested);
            }
        }
        Ok(())
    }
    
    /// Start repository discovery in background
    async fn start_discovery(&mut self, config: &AppConfig) -> Result<()> {
        self.start_discovery_for_path(config.base_dir.clone()).await
    }
    
    /// Start discovery for a specific path
    async fn start_discovery_for_path(&mut self, base_dir: std::path::PathBuf) -> Result<()> {
        let discovery_port = self.discovery_port.clone();
        let event_tx = self.event_tx.clone();
        
        self.tasks.spawn(async move {
            info!("Discovery task starting for {}", base_dir.display());
            
            // Run discovery in spawn_blocking since it's CPU-intensive
            let result = tokio::task::spawn_blocking(move || {
                let req = DiscoverReq { base: base_dir };
                discovery_port.scan(req)
            }).await;
            
            match result {
                Ok(Ok(repos)) => {
                    info!("Discovery found {} repositories", repos.len());
                    
                    // Send discovery events
                    for (id, meta) in repos {
                        if event_tx.send(Event::RepoDiscovered { id, meta }).is_err() {
                            error!("Event receiver dropped during discovery");
                            break;
                        }
                    }
                    
                    // Send completion event
                    let _ = event_tx.send(Event::ScanCompleted);
                }
                Ok(Err(e)) => {
                    error!("Discovery failed: {}", e);
                    let _ = event_tx.send(Event::Error {
                        id: None,
                        msg: format!("Discovery failed: {}", e),
                    });
                }
                Err(e) => {
                    error!("Discovery task panicked: {}", e);
                    let _ = event_tx.send(Event::Error {
                        id: None,
                        msg: format!("Discovery task failed: {}", e),
                    });
                }
            }
            
            Ok(())
        });
        
        Ok(())
    }
    
    /// Refresh git status for all known repositories
    async fn refresh_all_statuses(&mut self) -> Result<()> {
        let repo_ids: Vec<_> = self.projection.repositories.keys().cloned().collect();
        
        for repo_id in repo_ids {
            self.refresh_repository_status(repo_id).await?;
        }
        
        Ok(())
    }
    
    /// Refresh git status for a specific repository
    async fn refresh_repository_status(&mut self, id: RepoId) -> Result<()> {
        let git_port = self.git_port.clone();
        let event_tx = self.event_tx.clone();
        let id_clone = id.clone();
        
        self.tasks.spawn(async move {
            let id_for_error = id_clone.clone();
            let result = tokio::task::spawn_blocking(move || {
                git_port.status(&id_clone)
            }).await;
            
            match result {
                Ok(Ok(status)) => {
                    let _ = event_tx.send(Event::StatusUpdated { id, status });
                }
                Ok(Err(e)) => {
                    error!("Failed to get status for {}: {}", id_for_error.0, e);
                    let _ = event_tx.send(Event::Error {
                        id: Some(id_for_error.clone()),
                        msg: format!("Status update failed: {}", e),
                    });
                }
                Err(e) => {
                    error!("Status task panicked for {}: {}", id_for_error.0, e);
                    let _ = event_tx.send(Event::Error {
                        id: Some(id_for_error),
                        msg: format!("Status task failed: {}", e),
                    });
                }
            }
            
            Ok(())
        });
        
        Ok(())
    }
    
    /// Fetch a repository from its remote
    async fn fetch_repository(&mut self, id: RepoId, remote: String, prune: bool) -> Result<()> {
        let git_port = self.git_port.clone();
        let event_tx = self.event_tx.clone();
        let id_clone = id.clone();
        let remote_clone = remote.clone();
        
        self.tasks.spawn(async move {
            let id_for_error = id_clone.clone();
            let remote_for_msg = remote_clone.clone();
            let result = tokio::task::spawn_blocking(move || {
                git_port.fetch(&id_clone, &remote_clone, prune)
            }).await;
            
            match result {
                Ok(Ok(())) => {
                    let _ = event_tx.send(Event::RepoFetched {
                        id,
                        ok: true,
                        msg: Some(format!("Successfully fetched from {}", remote_for_msg)),
                    });
                }
                Ok(Err(e)) => {
                    error!("Failed to fetch {}: {}", id_for_error.0, e);
                    let _ = event_tx.send(Event::RepoFetched {
                        id: id_for_error.clone(),
                        ok: false,
                        msg: Some(format!("Fetch failed: {}", e)),
                    });
                }
                Err(e) => {
                    error!("Fetch task panicked for {}: {}", id_for_error.0, e);
                    let _ = event_tx.send(Event::RepoFetched {
                        id: id_for_error,
                        ok: false,
                        msg: Some(format!("Fetch task failed: {}", e)),
                    });
                }
            }
            
            Ok(())
        });
        
        Ok(())
    }
    
    /// Load commit log for a repository
    async fn load_repository_log(&mut self, id: RepoId, range: String, limit: usize) -> Result<()> {
        let git_port = self.git_port.clone();
        let event_tx = self.event_tx.clone();
        let id_clone = id.clone();
        let range_clone = range.clone();
        
        self.tasks.spawn(async move {
            let id_for_error = id_clone.clone();
            let result = tokio::task::spawn_blocking(move || {
                git_port.log(&id_clone, &range_clone, limit)
            }).await;
            
            match result {
                Ok(Ok(commits)) => {
                    let _ = event_tx.send(Event::LogLoaded { id, commits });
                }
                Ok(Err(e)) => {
                    error!("Failed to load log for {}: {}", id_for_error.0, e);
                    let _ = event_tx.send(Event::Error {
                        id: Some(id_for_error.clone()),
                        msg: format!("Log loading failed: {}", e),
                    });
                }
                Err(e) => {
                    error!("Log loading task panicked for {}: {}", id_for_error.0, e);
                    let _ = event_tx.send(Event::Error {
                        id: Some(id_for_error),
                        msg: format!("Log loading task failed: {}", e),
                    });
                }
            }
            
            Ok(())
        });
        
        Ok(())
    }
    
    /// Save configuration
    async fn save_configuration(&mut self, config: AppConfig) -> Result<()> {
        let config_store = self.config_store.clone();
        let event_tx = self.event_tx.clone();
        
        self.tasks.spawn(async move {
            let result = tokio::task::spawn_blocking(move || {
                config_store.save(&config)
            }).await;
            
            match result {
                Ok(Ok(())) => {
                    info!("Configuration saved successfully");
                }
                Ok(Err(e)) => {
                    error!("Failed to save configuration: {}", e);
                    let _ = event_tx.send(Event::Error {
                        id: None,
                        msg: format!("Config save failed: {}", e),
                    });
                }
                Err(e) => {
                    error!("Config save task panicked: {}", e);
                    let _ = event_tx.send(Event::Error {
                        id: None,
                        msg: format!("Config save task failed: {}", e),
                    });
                }
            }
            
            Ok(())
        });
        
        Ok(())
    }
    
    /// Main event processing loop
    async fn run_event_loop(&mut self) -> Result<()> {
        info!("Starting event loop");
        
        loop {
            tokio::select! {
                // Handle commands from the TUI
                command = self.command_rx.recv() => {
                    match command {
                        Some(cmd) => {
                            if let Err(e) = self.handle_command(cmd).await {
                                error!("Error handling command: {}", e);
                            }
                        }
                        None => {
                            info!("Command channel closed");
                        }
                    }
                }
                
                // Handle events from the event bus
                event = self.event_rx.recv() => {
                    match event {
                        Some(event) => {
                            if let Err(e) = self.handle_event(event).await {
                                error!("Error handling event: {}", e);
                            }
                        }
                        None => {
                            info!("Event channel closed, stopping event loop");
                            break;
                        }
                    }
                }
                
                // Handle completed background tasks
                task_result = self.tasks.join_next(), if !self.tasks.is_empty() => {
                    if let Some(result) = task_result {
                        match result {
                            Ok(Ok(())) => {
                                // Task completed successfully
                            }
                            Ok(Err(e)) => {
                                error!("Background task failed: {}", e);
                            }
                            Err(e) => {
                                error!("Background task panicked: {}", e);
                            }
                        }
                    }
                }
                
                // Handle quit signals, etc.
                else => {
                    break;
                }
            }
        }
        
        // Wait for all background tasks to complete
        info!("Shutting down background tasks");
        self.tasks.abort_all();
        
        Ok(())
    }
    
    /// Handle a single event and update the read projection
    async fn handle_event(&mut self, event: Event) -> Result<()> {
        // Forward event to external listeners (TUI)
        let _ = self.event_tx_external.send(event.clone());
        
        match &event {
            Event::RepoDiscovered { id, meta } => {
                info!("Repository discovered: {}", id.0);
                self.projection.apply(&event);
                
                // Register repository path with GitAdapter for future operations
                if let Some(git_adapter) = self.git_port.as_any().downcast_ref::<crate::adapters::git::GitAdapter>() {
                    git_adapter.register_repo(id.clone(), meta.path.clone());
                }
                
                // Automatically start status refresh for new repos
                self.refresh_repository_status(id.clone()).await?;
            }
            
            Event::ScanCompleted => {
                info!("Repository scan completed");
                self.projection.apply(&event);
            }
            
            Event::StatusUpdated { id, status: _ } => {
                info!("Status updated for repository: {}", id.0);
                self.projection.apply(&event);
            }
            
            Event::FetchProgress { id, done, total } => {
                info!("Fetch progress for {}: {}/{}", id.0, done, total);
                self.projection.apply(&event);
            }
            
            Event::RepoFetched { id, ok, msg } => {
                if *ok {
                    info!("Repository {} fetched successfully", id.0);
                    // Refresh status after successful fetch
                    self.refresh_repository_status(id.clone()).await?;
                } else {
                    error!("Repository {} fetch failed: {:?}", id.0, msg);
                }
                self.projection.apply(&event);
            }
            
            Event::LogLoaded { id, commits } => {
                info!("Log loaded for {}: {} commits", id.0, commits.len());
                self.projection.apply(&event);
            }
            
            Event::Error { id, msg } => {
                if let Some(repo_id) = id {
                    error!("Repository error for {}: {}", repo_id.0, msg);
                } else {
                    error!("Application error: {}", msg);
                }
                self.projection.apply(&event);
            }
            
            Event::QuitRequested => {
                info!("Quit requested via event");
                return Ok(()); // This will break the event loop
            }
        }
        
        Ok(())
    }
}

impl Drop for AppService {
    fn drop(&mut self) {
        // Abort all background tasks when the service is dropped
        self.tasks.abort_all();
    }
}