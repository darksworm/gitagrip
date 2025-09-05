# GitaGrip

A fast terminal UI for discovering, grouping, and inspecting multiple Git repositories.

## Features

- **Fast Repository Discovery**: Automatically scans directories to find Git repositories
- **Intelligent Grouping**: Groups repositories by directory structure 
- **Rich Git Status**: Shows branch names, dirty status, and ahead/behind counts
- **Colored Branch Display**: Main/master branches in bold green, others get consistent colors
- **Non-blocking UI**: Background scanning keeps the interface responsive
- **Comprehensive Testing**: 53 tests ensure reliability

## Installation

```bash
# Build from source
git clone https://github.com/darksworm/gitagrip.git
cd gitagrip
cargo build --release
./target/release/gitagrip
```

## Usage

```bash
# Scan current directory
gitagrip

# Scan specific directory
gitagrip --base-dir /path/to/repos

# Use custom config
gitagrip --config ~/.config/gitagrip/custom.toml
```

## Configuration

GitaGrip looks for configuration at:
- `~/.config/gitagrip/gitagrip.toml` (Linux/macOS)
- `~/Library/Application Support/gitagrip/gitagrip.toml` (macOS)

Example config:
```toml
version = 1
base_dir = "/home/user/code"

[ui]
show_ahead_behind = true
autosave_on_exit = false

[groups.Work]
repos = [
  "/home/user/code/project1",
  "/home/user/code/project2",
]
```

## Interface

```
┌─ GitaGrip    /home/user/code ──────────────────────────┐
│ ▼ Auto: work                                           │
│   ✓ project1 (main)                                    │
│   ● project2 (feature-branch ↑2)                       │
│ ▼ Auto: personal                                       │
│   ✓ dotfiles (master)                                  │ 
│ ▼ Ungrouped                                            │
│   ⋯ new-repo                                           │
└────────────────────────────────────────────────────────┘
```

**Status Indicators:**
- `✓` Clean repository
- `●` Dirty repository (uncommitted changes)
- `?` Git status unknown
- `⋯` Loading git status

**Branch Colors:**
- **Bold Green**: main, master
- **Various Colors**: Other branches (consistent per name)

## Controls

- `q` / `Ctrl+C` / `Esc`: Quit

## Development

GitaGrip is built using **Outside-In Test-Driven Development** with comprehensive integration tests that run the real application.

### Key Documents

- **[STRATEGY.md](STRATEGY.md)** - Our development philosophy and core principles
- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Practical lessons learned and best practices

### Development Setup

```bash
# Install Rust if not already installed
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Clone and test
git clone https://github.com/darksworm/gitagrip.git
cd gitagrip
cargo test
cargo run -- --base-dir .
```

### Architecture

- **Outside-in TDD**: Integration tests drive implementation
- **Milestone-based**: Clear deliverable increments (M0→M1→M2→M3)
- **Event-driven**: Background threads communicate via channels
- **Real-world testing**: Tests use actual git repositories

### Contributing

1. Read [STRATEGY.md](STRATEGY.md) and [DEVELOPMENT.md](DEVELOPMENT.md)
2. Write a "guiding star" integration test for new features
3. Implement to make tests pass
4. Ensure all 53 tests pass
5. Manual verification of user-facing changes

## Milestones

- **M0: Bootstrap** ✅ - Basic TUI structure and test framework
- **M1: Config + CLI** ✅ - Configuration loading and command-line interface  
- **M2: Repository Discovery** ✅ - Find and group git repositories
- **M3: Git Status Integration** ✅ - Display git status with colored branches
- **M4: Interactive Navigation** (next) - Navigate and perform operations

## License

MIT License - see LICENSE file for details.

## Credits

Built with:
- [Rust](https://www.rust-lang.org/)
- [Ratatui](https://github.com/ratatui-org/ratatui) for terminal UI
- [git2](https://github.com/rust-lang/git2-rs) for Git operations
- Love for developer productivity tools ❤️