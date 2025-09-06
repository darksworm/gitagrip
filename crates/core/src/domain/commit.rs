use serde::{Deserialize, Serialize};

/// Git commit information
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Commit {
    pub id: String,
    pub author: Author,
    pub message: String,
    pub timestamp: Timestamp,
}

/// Commit author information
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Author {
    pub name: String,
    pub email: String,
}

/// Commit timestamp (Unix timestamp with timezone offset in minutes)
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Timestamp {
    pub seconds: i64,
    pub offset_minutes: i32,
}

impl Timestamp {
    pub fn new(seconds: i64, offset_minutes: i32) -> Self {
        Self {
            seconds,
            offset_minutes,
        }
    }
}

impl std::fmt::Display for Timestamp {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        // Simple display - adapters can do more sophisticated formatting
        write!(f, "{}", self.seconds)
    }
}