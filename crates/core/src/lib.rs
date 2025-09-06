//! GitaGrip Core - Pure domain logic with no external dependencies
//!
//! This crate contains the business logic, domain types, and ports (interfaces)
//! for GitaGrip. It has no dependencies on UI frameworks, Git libraries, or
//! filesystem operations - those are handled by adapters.

pub mod domain;
pub mod ports;
pub mod app;
pub mod error;

// Re-exports for ergonomics
pub use domain::*;
pub use error::*;