# gitagrip

[![GitHub Downloads](https://img.shields.io/github/downloads/darksworm/gitagrip/total?style=flat-square&label=github+downloads)](https://github.com/darksworm/gitagrip/releases/latest)
[![Latest Release](https://img.shields.io/github/v/release/darksworm/gitagrip?style=flat-square)](https://github.com/darksworm/gitagrip/releases/latest)
[![License](https://img.shields.io/github/license/darksworm/gitagrip?style=flat-square)](./LICENSE)
[![Tests](https://github.com/darksworm/gitagrip/actions/workflows/test.yml/badge.svg)](https://github.com/darksworm/gitagrip/actions/workflows/test.yml)

> [!IMPORTANT] 
> This project is currently in DEVELOPMENT and is not stable

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
# Run with current directory
docker run --rm -it -v $(pwd):/repos ghcr.io/darksworm/gitagrip:latest

# Run with specific directory
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

# Scan specific directory (using flag)
gitagrip -dir /path/to/repos
gitagrip -d /path/to/repos  # shorthand

# Scan specific directory (as argument)
gitagrip /path/to/repos
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
Note: Enter integration requires lazygit to be installed and available in PATH.
- `Enter` - Open lazygit for the selected repository
- `H` - View git log
- `D` - View git diff
- `r` - Refresh repository status
- `f` - Fetch from remote
- `p` - Pull from remote
- `i` - Show repository info
- `I` - View repository command logs (pager)

### Group Management
- `z` - Toggle group expansion
- `N` - Create new group (with selection)
- `m` - Move repositories to group
- `Shift+R` - Rename group
- `Shift+J/K` - Move group up/down
- `d` - Delete group (when on group header)

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

## Lazygit Integration

To enable Enter → lazygit, install lazygit:
- macOS (Homebrew): `brew install lazygit`
- Linux: `sudo pacman -S lazygit` (Arch) or see release binaries at https://github.com/jesseduffield/lazygit
- Go install (latest): `go install github.com/jesseduffield/lazygit@latest`

You can override the lazygit binary path via the `GITAGRIP_LAZYGIT_BIN` environment variable for testing.

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

### Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please follow conventional commits for your commit messages.

## 📄 License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Bubble](https://github.com/charmbracelet/bubbles) - TUI components
- And the amazing Go community 💙
