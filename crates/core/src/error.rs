use thiserror::Error;

/// Core domain errors
#[derive(Error, Debug)]
pub enum CoreError {
    #[error("Repository not found: {id}")]
    RepositoryNotFound { id: String },
    
    #[error("Invalid repository path: {path}")]
    InvalidPath { path: String },
    
    #[error("Group not found: {name}")]
    GroupNotFound { name: String },
    
    #[error("Duplicate group name: {name}")]
    DuplicateGroup { name: String },
    
    #[error("Invalid command: {reason}")]
    InvalidCommand { reason: String },
    
    #[error("Port error: {source}")]
    Port { source: anyhow::Error },
}

pub type Result<T> = std::result::Result<T, CoreError>;