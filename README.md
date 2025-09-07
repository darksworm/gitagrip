# GitaGrip

[![GitHub Downloads](https://img.shields.io/github/downloads/darksworm/gitagrip/total?style=flat-square&label=github+downloads)](https://github.com/darksworm/gitagrip/releases/latest)
[![Latest Release](https://img.shields.io/github/v/release/darksworm/gitagrip?style=flat-square)](https://github.com/darksworm/gitagrip/releases/latest)
[![License](https://img.shields.io/github/license/darksworm/gitagrip?style=flat-square)](./LICENSE)
[![Tests](https://github.com/darksworm/gitagrip/actions/workflows/test.yml/badge.svg)](https://github.com/darksworm/gitagrip/actions/workflows/test.yml)

A fast, keyboard-driven terminal UI for discovering, grouping, and inspecting multiple Git repositories. Written in Go using Bubble Tea for a responsive, efficient experience.

## ✨ Features

- 🚀 **Fast Repository Discovery**: Automatically scans directories to find all Git repositories
- 📁 **Smart Grouping**: Organize repositories by directory structure or custom groups
- 📊 **Rich Git Status**: Shows branch names, dirty status, ahead/behind counts, and more
- 🎨 **Colored Branch Display**: Main/master branches in bold green, others get consistent colors
- ⚡ **Non-blocking UI**: Background operations keep the interface responsive
- 🔍 **Search & Filter**: Quickly find repositories with powerful search and filtering
- 📦 **Zero Dependencies**: Single binary with no runtime requirements except git

## 🚀 Installation

<details>
  <summary><strong>Quick Install (Linux/macOS)</strong></summary>

```bash
curl -sSL https://raw.githubusercontent.com/darksworm/gitagrip/main/install.sh | sh
```

Install a specific version:
```bash
curl -sSL https://raw.githubusercontent.com/darksworm/gitagrip/main/install.sh | sh -s -- v0.1.0
```
</details>

<details>
  <summary><strong>Homebrew (macOS/Linux)</strong></summary>

```bash
brew tap darksworm/homebrew-tap
brew install gitagrip
```
</details>

<details>
  <summary><strong>AUR (Arch Linux)</strong></summary>

```bash
yay -S gitagrip-bin
# or
paru -S gitagrip-bin
```
</details>

<details>
  <summary><strong>Docker</strong></summary>

```bash
# Run with current directory as base
docker run --rm -it -v $(pwd):/repos ghcr.io/darksworm/gitagrip:latest

# Scan specific directory
docker run --rm -it -v /path/to/repos:/repos ghcr.io/darksworm/gitagrip:latest
```
</details>

<details>
  <summary><strong>Go Install</strong></summary>

```bash
go install github.com/darksworm/gitagrip@latest
```
</details>

<details>
  <summary><strong>Download Binary</strong></summary>

Download the appropriate binary for your platform from the [releases page](https://github.com/darksworm/gitagrip/releases/latest).

**Linux/macOS:**
```bash
# Download (replace VERSION and PLATFORM)
curl -LO https://github.com/darksworm/gitagrip/releases/download/vVERSION/gitagrip-VERSION-PLATFORM.tar.gz

# Extract
tar -xzf gitagrip-VERSION-PLATFORM.tar.gz

# Install
sudo mv gitagrip /usr/local/bin/
```

**Windows:**
Download the `.zip` file, extract, and add to your PATH.
</details>

## 📖 Usage

```bash
# Scan current directory
gitagrip

# Scan specific directory
gitagrip --base-dir /path/to/repos

# Use custom config file
gitagrip --config ~/.config/gitagrip/custom.json
```

## ⌨️ Keyboard Shortcuts

### Navigation
- `↑/↓`, `j/k` - Navigate up/down
- `←/→`, `h/l` - Collapse/expand groups
- `PgUp/PgDn` - Page up/down
- `gg/G` - Go to top/bottom

### Selection
- `Space` - Toggle selection
- `a/A` - Select/deselect all
- `Esc` - Clear selection

### Repository Actions
- `r` - Refresh repository status
- `f` - Fetch from remote
- `p` - Pull from remote
- `l` - View git log
- `i` - Show repository info

### Group Management
- `z` - Toggle group expansion
- `N` - Create new group (with selection)
- `m` - Move repositories to group
- `Shift+R` - Rename group
- `Shift+J/K` - Move group up/down
- `d` - Delete group (when on group header)
- `H` - Hide selected repositories

### Search & Filter
- `/` - Search repositories
- `n` - Next search result
- `Shift+N` - Previous search result
- `F` - Filter repositories
- `s` - Sort options

### Other
- `?` - Show help
- `q` - Quit

### Filter Examples
- `status:dirty` - Show only repositories with uncommitted changes
- `status:clean` - Show only clean repositories  
- `status:ahead` - Show repositories ahead of remote

## 📁 Configuration

GitaGrip stores its configuration at:
- **Linux**: `~/.config/gitagrip/config.json`
- **macOS**: `~/Library/Application Support/gitagrip/config.json`
- **Windows**: `%APPDATA%\gitagrip\config.json`

### Example Configuration

```json
{
  "version": 1,
  "base_dir": "/home/user/code",
  "ui": {
    "show_ahead_behind": true,
    "autosave_on_exit": true
  },
  "groups": {
    "Work": [
      "/home/user/code/project1",
      "/home/user/code/project2"
    ],
    "Personal": [
      "/home/user/code/dotfiles"
    ]
  }
}
```

## 🖥️ Interface

```
gitagrip                                              ↻ Refreshing 2  ↓ Fetching 1

▼ Work (2)
  ● project-api (feature/auth ↑2↓1)
  ✓ project-web (main)

▼ Personal (3)
  ✓ dotfiles (master)
  ● blog (draft-post)
  ⚠ old-project (cleanup ⚠)

▶ Archived (5)

Press ? for help
```

### Status Indicators
- `✓` Clean repository
- `●` Dirty repository (uncommitted changes)
- `⚠` Repository with errors
- `⋯` Loading status
- `?` Unknown status

### Branch Colors
- **Bold Green**: main/master branches
- **Various Colors**: Other branches get consistent colors based on name

## 🛠️ Development

### Prerequisites
- Go 1.21 or later
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/darksworm/gitagrip.git
cd gitagrip

# Build
go build

# Run tests
go test ./...

# Install locally
go install
```

### Architecture

GitaGrip follows an event-driven architecture:

```
cmd/gitagrip/         - Application entry point
internal/
  ├── config/         - Configuration management
  ├── domain/         - Domain models and events
  ├── discovery/      - Repository discovery service
  ├── eventbus/       - Event-driven communication
  ├── git/            - Git operations service
  ├── groups/         - Group management service
  └── ui/             - Bubble Tea UI implementation
```

### Key Principles

- **Event-Driven**: Services communicate via a central event bus
- **Non-Blocking**: All I/O operations run on background goroutines
- **Clean Architecture**: Separation of concerns with clear boundaries
- **Comprehensive Testing**: Both unit and integration tests

### Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please follow conventional commits for your commit messages.

## 📋 Roadmap

- [x] Core TUI with repository discovery
- [x] Git status integration
- [x] Repository grouping
- [x] Fetch/pull operations
- [x] Search and filtering
- [ ] Push operations
- [ ] Bulk operations
- [ ] Git worktree support
- [ ] Repository templates
- [ ] Plugin system

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Bubble](https://github.com/charmbracelet/bubbles) - TUI components
- And the amazing Go community 💙

