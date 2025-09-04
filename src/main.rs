use anyhow::Result;
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
use tracing::{error, info};

struct App {
    should_quit: bool,
}

impl Default for App {
    fn default() -> App {
        App { should_quit: false }
    }
}

impl App {
    fn run<B: Backend>(&mut self, terminal: &mut Terminal<B>) -> Result<()> {
        loop {
            terminal.draw(|f| self.ui(f))?;

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

        // Title
        let title = Paragraph::new("YARG - Yet Another Repo Grouper")
            .block(Block::default().borders(Borders::ALL))
            .style(Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD));
        f.render_widget(title, chunks[0]);

        // Main content placeholder
        let main_content = Paragraph::new("Repository discovery and status will appear here...")
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

    // Setup terminal
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    // Create app and run it
    let mut app = App::default();
    let res = app.run(&mut terminal);

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
    fn test_app_default() {
        let app = App::default();
        assert!(!app.should_quit);
    }

    #[test]
    fn test_app_can_quit() {
        let mut app = App::default();
        app.should_quit = true;
        assert!(app.should_quit);
    }

    #[test] 
    fn test_app_state_transitions() {
        let mut app = App::default();
        
        // Initially should not quit
        assert!(!app.should_quit);
        
        // Can set to quit state
        app.should_quit = true;
        assert!(app.should_quit);
        
        // Can reset quit state
        app.should_quit = false;
        assert!(!app.should_quit);
    }

    // Note: Testing the actual TUI rendering and key handling would require 
    // more complex integration tests with mock terminals, which we'll add 
    // as we build more functionality
}
