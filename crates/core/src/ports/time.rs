/// Clock abstraction for testability
pub trait Clock: Send + Sync {
    /// Get current Unix timestamp
    fn now(&self) -> i64;
}

/// System clock implementation
#[derive(Debug, Default)]
pub struct SystemClock;

impl Clock for SystemClock {
    fn now(&self) -> i64 {
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i64
    }
}