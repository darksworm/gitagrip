## GEMINI.md

### Project Overview

`gitagrip` is a terminal-based UI application written in Go for managing multiple Git repositories. It provides a quick and efficient way to discover, group, and inspect the status of repositories within a specified directory. The user interface is built using the `charmbracelet/bubbletea` framework, providing a responsive and interactive experience in the terminal.

The application is designed with a decoupled architecture, utilizing an event bus for communication between different services such as repository discovery, Git operations, and configuration management. This allows for non-blocking UI updates and a more robust and extensible system.

### Building and Running

To build and run the project, you need to have Go installed.

**Build:**

```sh
go build -o gitagrip ./cmd/gitagrip
```

**Run:**

You can run the application with an optional directory to scan. If no directory is provided, it will scan the current directory.

```sh
# Scan the current directory
./gitagrip

# Scan a specific directory
./gitagrip --dir /path/to/your/projects
```

**Testing:**

The project does not have a clear, top-level test command. To run tests, you would likely need to run `go test` within each package directory.

```sh
go test ./...
```

### Development Conventions

*   **Architecture:** The project follows an event-driven architecture with a central event bus for communication between services. This promotes loose coupling and modularity.
*   **UI:** The user interface is built using the `charmbracelet/bubbletea` library, which is based on The Elm Architecture (Model-View-Update). The main UI state is managed in the `internal/ui/model.go` file.
*   **Configuration:** Application configuration is handled by the `internal/config` package and is stored in a `.gitagrip.toml` file in the scanned directory.
*   **Dependencies:** The project uses Go modules for dependency management. Key dependencies include `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, and `pelletier/go-toml/v2`.
*   **Code Structure:** The code is organized into `cmd` for the main application entry point and `internal` for all the core logic, following standard Go project layout. The `internal` directory is further subdivided by feature, such as `discovery`, `git`, `ui`, etc.
