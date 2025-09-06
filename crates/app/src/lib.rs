//! GitaGrip application library
//! 
//! This exposes the public API of the GitaGrip application for testing and external usage.

// Old architecture modules (will be phased out)
pub mod app;
pub mod cli; 
pub mod config;
pub mod git;
pub mod scan;

// New hexagonal architecture modules
pub mod adapters;
pub mod services; 
pub mod tui;
pub mod main_new;