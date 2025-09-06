Gitagrip TUI – Architectural Rewrite Plan
Objectives and Requirements

Preserve Features & Async Behavior: The new design must retain all current features of Gitagrip (background Git repository scanning, repository grouping, config persistence, etc.) and keep the event-driven asynchronous behavior (e.g. non-blocking background updates).

Idiomatic, Maintainable Go: We will use Go modules and idiomatic patterns for clean, testable code. This means clear package structures, minimal global state, and leveraging Go’s concurrency features (goroutines, channels, context) in a controlled manner.

Separation of Concerns: Strongly decouple different parts of the app:

Domain logic (e.g. grouping rules, repository status handling)

Git I/O (interfacing with Git repositories, running Git commands or parsing Git data)

Filesystem scanning (finding Git repos in the filesystem)

Config management (loading/saving user config and state)

TUI/Presentation (Bubble Tea UI and Bubbles components)
Each concern should reside in its own package or module, interacting via well-defined interfaces or an event bus, rather than through tight coupling.

Event/Message Bus: Introduce a central publish-subscribe event bus to relay messages between components. Domain events (e.g. RepoDiscovered, StatusUpdated, GroupAdded) and UI intents (user actions) will be dispatched on this bus. This decouples producers from consumers and mirrors an event-driven design.

Concurrency & Error Handling: Use Go’s concurrency primitives for background work:

Run background tasks (scanning, Git status checks) in goroutines, communicating results via channels or the event bus.

Use context.Context to manage cancellation (e.g. stop scanning when the app exits).

Handle errors by logging them and converting them into events/messages that the UI can display (so issues bubble up instead of crashing the app). For example, a filesystem permission error during scanning might be logged and also sent as an ErrorOccurred event to show in the UI’s status bar.

Simplicity and Clarity: Keep the design as simple as possible while meeting the above goals. Avoid over-engineering with too many layers or abstractions. The code structure should be easy for new contributors to navigate and extend. We’ll follow common Go project structure conventions (like using a cmd/ directory for the main program and internal/ packages for core logic) but only to the extent that it adds clarity
github.com
. Every abstraction should have a clear purpose.

Proposed Project Structure

Organize the project into modules/directories that reflect the separated concerns. Below is a recommended layout with each package’s purpose:

gitagrip/                   (Go module root)
├── cmd/                    
│   └── gitagrip/           (Main application package)
│       └── main.go         (Application entry point)
└── internal/               (Private application code)
    ├── config/             (Config management)
    │   ├── config.go       (Loading/saving config, config struct definitions)
    │   └── config_test.go 
    ├── domain/             (Core domain models and events)
    │   ├── models.go       (e.g. Repository, Group structs)
    │   ├── events.go       (Domain event definitions like RepoDiscovered, StatusUpdated)
    │   └── domain_test.go
    ├── discovery/          (Filesystem scanning for git repos)
    │   ├── discovery.go    (DiscoveryService implementation)
    │   └── discovery_test.go
    ├── git/                (Git I/O operations)
    │   ├── gitservice.go   (GitService implementation for repo status, etc.)
    │   └── gitservice_test.go
    ├── groups/             (Group management logic)
    │   ├── groups.go       (GroupManager implementation, managing grouping of repos)
    │   └── groups_test.go
    ├── eventbus/           (Central event bus implementation)
    │   └── eventbus.go     (Publish-subscribe hub for events, used across services)
    └── ui/                 (TUI using Bubbletea and Bubbles)
        ├── model.go        (Bubbletea Model, message handling, UI update logic)
        ├── view.go         (UI rendering with Bubbles components, styles via Lipgloss)
        └── ui_test.go


Notes on structure: We use the internal/ directory to clearly indicate that these packages are internal implementation details not meant to be imported by external projects. This helps encapsulate our code. The number of packages is kept reasonable to avoid undue complexity – each corresponds to a major concern of the app. For example, discovery handles only finding repositories on disk and knows nothing of how they’ll be displayed, while ui knows how to present data but nothing about how scanning is done. This clear separation will make the code easier to maintain and test. (If the project is small, you could even start with fewer packages and only split out as it grows, since an overly rigid structure from the start can be overkill
github.com
.)

Core Components and Responsibilities

Each internal package/component has a well-defined responsibility:

ConfigService (internal/config): Loads configuration (such as persisted group definitions, user preferences, or last-known repository list) from a file (e.g. JSON or YAML in the user’s home directory). Exposes methods to retrieve config and save changes. The config struct might include things like list of directories to scan, defined groups and their member repo paths, UI settings, etc. This service should be initialized at startup (before the UI launches) to provide initial data. It can emit a ConfigLoaded event after reading config (containing info like which base paths or repos to scan, and group data) so other components can act on it. If config changes (e.g. user adds a new group), it should handle writing to disk and possibly emit a ConfigSaved or relevant events.

DiscoveryService (internal/discovery): Handles filesystem scanning to discover Git repositories. This will run asynchronously in the background. For example, it might walk through specified directories (from config) to find any folder containing a .git/ subdirectory (indicating a repo). When it finds a new repository, it publishes a RepoDiscovered event with details (e.g. repo path, name). The DiscoveryService could continuously watch for changes (perhaps using a ticker or fsnotify) or run on demand (e.g. when user triggers a refresh) – either way, it should not block the UI. This service knows nothing about UI; it just finds repos and fires events. It likely uses goroutines for file I/O, and channels or the event bus to report findings.

GitService (internal/git): Responsible for interacting with Git repositories (reading statuses, branches, counts of uncommitted changes, ahead/behind, etc.). When a repository is discovered or when a periodic refresh is due, the GitService will perform Git I/O (possibly by calling git CLI or using a Go Git library) to get the current status of that repo. It then emits events like StatusUpdated (including repo identifier and new status info) or RepoScanned. This service could also handle continuous monitoring: e.g. it might maintain a list of known repos and periodically refresh their status in the background. Internally it can use goroutines (one per repo or a worker pool) to parallelize Git commands, communicating results via the event bus. All Git operations errors should be caught; on error, log the issue and send an ErrorOccurred event (with context, like “Failed to get status of repo X”) rather than stopping the whole app.

GroupManager (internal/groups): Encapsulates the logic for repository grouping. It might maintain data structures mapping group names to lists of repositories (or repo IDs). It provides functions to add/remove a group, assign a repo to a group, etc. The UI can call this component (or send events to it) when the user creates or modifies groups. GroupManager will update the in-memory state (and possibly trigger a save via ConfigService) and then emit events like GroupAdded, GroupRemoved, or RepoMovedToGroup so the UI (and other services if needed) can react. This separation ensures group logic (like ensuring no duplicate group names, etc.) is handled in one place, making it easy to test independently.

Event Bus (internal/eventbus): A lightweight publish-subscribe hub that connects all the above components and the UI. The event bus allows components to broadcast domain events without needing direct references to each other. For example:

DiscoveryService publishes a RepoDiscoveredEvent on the bus.

The GitService and UI (which are subscribed to that event type) will receive it. The GitService may respond by immediately checking that repo’s status (and then publishing a StatusUpdated event), while the UI will add the new repo to its list view.

Similarly, if the user triggers a “refresh” in the UI, the UI can send a ScanRequested event on the bus, which the DiscoveryService listens for to start a new scan (or the UI could call a DiscoveryService method directly – both approaches are possible, but using an event keeps the pattern consistent).

The event bus can be implemented idiomatically using Go channels: e.g. have each event type or topic correspond to a channel, or use a single channel carrying an Event interface with type information. A simple implementation could maintain a map of subscribers (channels or callback functions) by event type.

For simplicity, one might create an interface like:

type EventBus interface {
    Publish(event DomainEvent)
    Subscribe(eventType EventType, handler func(DomainEvent))
}


and have an internal loop or select on channels to dispatch events to subscribers. This avoids tight coupling and makes testing easier (you can inject a mock EventBus that just records events, for example).

UI (internal/ui): The Terminal User Interface built with Bubble Tea and Bubbles components. This holds the application state needed for presentation (lists of repos, grouping info, current selection, loading spinners, etc.) and defines how to render it. The UI layer should only handle presentation logic and user interaction, delegating actual work to the domain. It will subscribe to relevant domain events (via the event bus) and convert them into UI messages that Bubble Tea’s Update function can process. Conversely, when the user presses a key or invokes a command in the UI, the UI layer will emit a corresponding event or call a service. The UI’s Bubbletea Model could internally maintain references to needed services or an event bus handle (passed in during initialization) so that on certain keypresses it can do things like bus.Publish(UserRequestedRefresh{}) or call DiscoveryService.StartScan(). The UI will use Bubbles components (like lists, tables, status bar, etc.) to display data. For instance, a list component can show repositories grouped by category (perhaps using a custom delegate to style group headers), an spinner bubble can indicate background activity for scanning, etc.

Importantly, the UI must remain de-coupled from business logic – it should not perform Git or filesystem operations directly, in line with the separation principle. This approach aligns with guidance from experienced Bubble Tea users: keeping business logic out of the Bubble Tea update/view and instead interacting through clear APIs or events leads to a more maintainable design
reddit.com
reddit.com
.

Event-Driven Data Flow

The components interact through events and messages. Below is an overview of the high-level data flow in the application:

Figure: High-level architecture and event flow in Gitagrip. Domain services (Discovery, Git, Config, GroupManager) communicate via a central Event Bus, and the UI (Bubbletea) also connects to this bus. Domain events (solid arrows) like RepoDiscovered or StatusUpdated are published by services and received by both the UI and other services (e.g. GitService reacts to a new repo event). User actions (dashed arrow) in the UI are sent as messages or events (e.g. ScanRequested, GroupCreated) to the bus, where appropriate services handle them. This decoupled flow ensures the UI doesn’t call services directly and domain logic doesn’t depend on UI code, making the system easier to understand and extend.

From startup to runtime, a typical sequence might be:

Application Start: main.go initializes the EventBus and each service, passing the EventBus into each so they can publish/subscribe to events. It then loads config via ConfigService. Once config is loaded (say it contains directories to scan and any existing groups), ConfigService publishes a ConfigLoaded event (or simply returns the config data to main).

Initial Scan: Upon config load, the main function (or the UI, after startup) triggers an initial repository scan. This could be done by publishing a ScanRequested event on the bus or by directly calling a DiscoveryService.StartScan() method. In either case, the DiscoveryService begins scanning in a goroutine (using a context from main to allow cancellation). The UI can show a loading spinner indicating “scanning in progress”.

Repo Discovery: As the scanner finds Git repositories, for each repo it publishes a RepoDiscovered event (containing at least the repo path, and perhaps an initial stub of repo info). The EventBus broadcasts this to all subscribers:

The UI subscribed to RepoDiscovered will receive the event (likely via a channel or callback) and create a new entry in its list of repositories (initially with placeholder status like “Loading…”). In Bubbletea, this could be handled by sending a custom message RepoDiscoveredMsg to the UI’s Update method which then updates the model’s state (e.g. adds a new item to the list).

The GitService also subscribes to RepoDiscovered; when it gets the event, it spawns a goroutine or issues a command (perhaps via a bounded worker pool) to fetch the Git status of that new repo (e.g. run git status or use a library to get number of untracked files, etc.). This operation, being I/O heavy, happens asynchronously.

Git Status Update: Once the GitService has gathered the status info for a repo (or if the status changes later on), it publishes a StatusUpdated event with details (repo identifier and status data such as branch, ahead/behind counts, dirty flag, etc.). The EventBus again distributes this:

The UI receives StatusUpdated and updates the display for that repository (e.g. removes the spinner and shows the repo’s status: “main * (2↑ 1↓)” meaning 2 commits ahead, 1 behind, for example). This is done by sending a StatusUpdatedMsg to the Bubbletea model which triggers a re-render of that repo’s list item.

(If any other component cares about statuses, they would subscribe as well. For instance, maybe GroupManager might listen if it automatically organizes repos by status or something, though in our design grouping is manual.)

User Interaction: The user can navigate and trigger actions in the TUI (thanks to Bubbletea’s message handling of keypresses). For example:

If the user presses “r” to refresh, the UI’s Update function will intercept that key and publish a ScanRequested event on the bus (or call DiscoveryService directly). The DiscoveryService, on receiving this, can re-scan the filesystem (perhaps clearing previous results or updating them). This will generate new RepoDiscovered events for any new repos found (and possibly events for removed repos if needed).

If the user opens a menu to create a new group and selects some repositories for it, the UI would call GroupManager.CreateGroup(name, repoIDs) or publish a GroupAdded event with that info. GroupManager will handle creating the group (and persisting via ConfigService) then emit a GroupAdded event. The UI, on receiving that, will add the new group to its group list view and possibly reorganize the repository list under the new group heading.

If the user quits (e.g. presses “q”), the UI’s Update will return a tea.Quit command to Bubbletea to exit the main loop. We also need to ensure background goroutines stop: using the context passed to services, we would cancel it on exit (perhaps main can cancel a root context in a defer once the Bubbletea program finishes). This will signal DiscoveryService or others to stop any ongoing work gracefully.

Persistent Updates: Throughout runtime, the event bus continues to mediate between background tasks and UI. For example, DiscoveryService might periodically rescan in the background (timer-driven), or GitService might poll for repo status changes every few minutes – each time producing events that update the UI. The UI never directly calls these services’ internals or manipulates domain state; it only reacts to events and sends out user intent events. This makes the app reactive and easier to reason about, since state changes flow in one direction (from domain to UI via events), and user intents flow in the opposite direction (from UI to domain via events or service calls).

This event-driven approach not only preserves the asynchronous behavior of the original app, but also compartmentalizes it. We can add new event types or subscribers in the future without modifying all components (Open/Closed principle in practice).

Concurrency Model and Error Handling

Concurrency: We leverage goroutines and channels in a structured way:

Each service that performs background work does so in its own goroutines. For example, DiscoveryService.StartScan() might spawn a goroutine that walks the filesystem. That goroutine can use a channel (internal to DiscoveryService) to send back each discovered repo, or it can directly publish events to the EventBus for each repo. We avoid using shared mutable data across goroutines – instead, communicate via events/messages which is Go’s recommended approach to concurrency (share memory by communicating).

We use context.Context to coordinate cancellation and timeouts. A context (likely created in main with context.Background() and then context.WithCancel) is passed into long-lived goroutines. If the user exits or if we need to stop a background process, canceling the context will unblock those goroutines (they should be checking ctx.Done() periodically or on blocking operations).

The Bubbletea UI runs on the main thread (the Bubbletea library handles its internal concurrency for I/O). We must ensure that any goroutine that wants to update the UI does so by sending a message into the Bubbletea event loop (never directly modifying the UI model state from outside). Our EventBus subscribers for the UI will need to use Bubbletea’s thread-safe message passing. Bubbletea provides Program.Send(msg) or allows returning tea.Cmd from the Update to inject messages. One approach is to give the EventBus a reference to a function to forward events to the UI. For instance, when initializing the UI, you could subscribe to certain domain events by providing a handler that does prog.Send(RepoDiscoveredMsg{...}). That way, the UI messages enter the normal Update loop and are handled there, keeping UI updates on the main thread.

Error Handling: Each service should capture and handle its own errors, converting them into log entries and/or events rather than panicking:

Use Go’s log (or a structured logger) to record errors for debugging. For example, if git status command fails, log the error with details of which repo.

Also inform the UI of issues via an event. Define an event type like ErrorEvent with fields Message string and maybe Severity. If, say, GitService cannot read a repository (perhaps permission denied), it can publish ErrorEvent{Message: "Failed to read repo X: permission denied"}. The UI, on receiving an ErrorEvent, could display it in a status line or a popup dialog. This is the “message bubbling” approach – errors are propagated up to the UI/user level in a controlled manner.

For non-critical errors, the system can continue running (e.g. one repo failed to scan, but others are fine). By logging internally and showing a brief notice to the user, we ensure the app doesn’t crash and the user is aware of any issues.

If an error is critical (say, config file is completely unreadable), the ConfigService might publish an event or return an error to main, and the app could handle it by showing a message and exiting or falling back to defaults. The key is to handle it gracefully and document that in the architecture (e.g. “ConfigService will return an error if config is corrupt, which main() should handle by logging and maybe using a default config or prompting the user”).

Example Interfaces and Types

To clarify the boundaries between components, here are a few example interface definitions and data structures in the design:

// Domain models (in internal/domain/models.go)
type Repository struct {
    Path   string
    Name   string
    Group  string   // group name it belongs to ("" if ungrouped)
    Status RepoStatus
}
type RepoStatus struct {
    Branch          string
    AheadCount      int
    BehindCount     int
    Uncommitted     int    // number of unstaged/uncommitted changes
    UnpushedCommits int    // commits ahead of remote
    // ... other git status info as needed
}

// Domain events (in internal/domain/events.go)
type EventType string
const (
    EventRepoDiscovered EventType = "RepoDiscovered"
    EventStatusUpdated  EventType = "StatusUpdated"
    EventError          EventType = "Error"
    EventGroupAdded     EventType = "GroupAdded"
    // etc...
)
type DomainEvent interface {
    Type() EventType
}
type RepoDiscoveredEvent struct {
    Repo Repository
}
func (e RepoDiscoveredEvent) Type() EventType { return EventRepoDiscovered }

type StatusUpdatedEvent struct {
    RepoPath string
    Status   RepoStatus
}
func (e StatusUpdatedEvent) Type() EventType { return EventStatusUpdated }

type ErrorEvent struct {
    Message string
    Err     error    // underlying error
}
func (e ErrorEvent) Type() EventType { return EventError }


The above are simple struct implementations of a DomainEvent interface for each event type, which can be published on the bus. The UI will likely convert these into Bubbletea message types (perhaps reuse them directly as tea.Msg since any type can be a message, or wrap them in UI-specific types). Now, example service interfaces:

// Service interfaces (in their respective packages):

// DiscoveryService finds git repositories (could be an interface if we want to mock it in tests).
type DiscoveryService interface {
    StartScan(ctx context.Context, roots []string) error 
    // Perhaps also methods to subscribe to new discoveries if not using global bus, e.g.:
    // SubscribeFound(chan<- string)  -- not needed if using event bus globally.
}

// GitService handles git repo status checks.
type GitService interface {
    RefreshRepo(ctx context.Context, repoPath string) (RepoStatus, error)
    RefreshAll(ctx context.Context, repos []Repository)
    // Could either actively return results or just publish events internally.
}

// GroupManager handles grouping of repositories.
type GroupManager interface {
    CreateGroup(name string) error
    AddRepoToGroup(repoPath string, groupName string) error
    RemoveGroup(name string) error
    // ... other methods as needed (or this could be done via events as well)
}

// ConfigService manages persistent configuration.
type ConfigService interface {
    Load() (Config, error)
    Save(config Config) error
}
type Config struct {
    ScanPaths []string        // directories to scan for repos
    Groups    map[string][]string  // group name to list of repo paths
    // ... other config (UI settings etc.)
}


These interfaces will have concrete implementations inside the packages (e.g. discovery.service struct implementing DiscoveryService). The use of interfaces will help in testing – for instance, you can create a fake DiscoveryService that emits a known set of repos for a test, or a fake GitService that returns preset statuses, allowing you to test the UI in isolation.

Using Go’s internal packages and interfaces this way ensures loose coupling and easier maintenance. If in the future we need to change how we scan (say, use fd or another tool instead of our own scanner) or how we get Git info (e.g. integrate libgit2), we can do so by swapping out the implementation of GitService without impacting the UI or other parts of the code.

Main Loop and Message Routing

The main.go (in cmd/gitagrip) will wire everything together and launch the UI. Here’s what the high-level main function workflow might look like:

Initialize Logger (optional): Set up logging to file or stderr for debug info.

Load Config: Use config.Load() to read the configuration file. Handle errors (if it fails, maybe create a default config or alert the user). If successful, we now have initial data like directories to scan and any saved grouping.

Create Event Bus: Instantiate the event bus (for example, bus := eventbus.New()). Possibly also create a top-level context, cancel := context.WithCancel(context.Background()) to pass into services.

Instantiate Services: Create the concrete implementations of DiscoveryService, GitService, GroupManager, etc., providing them with necessary dependencies (like the event bus and config). For example:

bus := eventbus.New()
discovery := discovery.NewDiscoveryService(bus)
gitSvc    := git.NewGitService(bus)
groups    := groups.NewGroupManager(bus, cfg.Groups) // perhaps pass initial groups from config
configSvc := config.NewConfigService(configPath, bus)


These constructors would subscribe the service to relevant events. For instance, NewDiscoveryService(bus) might do bus.Subscribe(EventScanRequested, svc.handleScanRequest). Similarly, GroupManager might subscribe to events like RepoDiscovered if it wants to auto-assign repos or just to update internal state.

Initialize UI Model: Create the Bubbletea model, providing it with references or closures for interactions. For instance:

uiModel := ui.NewModel(bus, cfg)  
// NewModel sets up initial state (maybe empty list with “Loading…” message) and subscribes to needed event types.


The UI model may subscribe to domain events via the bus. Alternatively, the UI model could simply rely on Bubbletea messages that we send via the bus – design choice. One way is to give the UI model a channel (e.g. eventsChan := make(chan DomainEvent)) and subscribe that channel to the EventBus for all relevant events. The UI’s Bubbletea program can then have an Init or goroutine reading from eventsChan and sending corresponding tea.Msg into the program. For example:

// In NewModel:
bus.Subscribe(EventRepoDiscovered, func(e DomainEvent) {
    if evt, ok := e.(RepoDiscoveredEvent); ok {
        p.Send(RepoDiscoveredMsg{Repo: evt.Repo})  // p is the tea.Program
    }
})


If directly injecting Program p isn’t feasible at model construction time, another approach is to use a Bubbletea command to continuously read from the event channel. (Bubbletea allows a model’s Init to return a tea.Cmd; you can create a command that waits on the event channel and returns an appropriate Msg for each event.)

Start Initial Processes: Possibly trigger initial events. For example, after loading config:

If config had some saved repository list or groups, we might publish events like GroupAdded for each existing group so UI can render them. Or simply call uiModel.SetGroups(cfg.Groups) to initialize the state (since those are not asynchronous results, just initial state).

Kick off the first scan: If the design uses an event to initiate scanning, do bus.Publish(ScanRequestedEvent{Paths: cfg.ScanPaths}). If not using an event, call go discovery.StartScan(ctx, cfg.ScanPaths). In both cases, the scanning will begin and events will start flowing for discovered repos.

Run the Bubbletea Program: Set up the TUI program:

p := tea.NewProgram(uiModel)
if err := p.Start(); err != nil {
    log.Fatalf("Failed to start UI: %v", err)
}


Once p.Start() is called, Bubbletea takes over the terminal and starts its internal event loop. The UI now responds to user input and incoming messages. Under the hood, our model’s Update(msg) will be called for each message:

For keypresses: we interpret keys and possibly publish events or call services. E.g., if user presses "q", Update returns tea.Quit. If user presses "a" to add a group, Update might capture a name (maybe via a text input bubble) and then publish a GroupAdded event and call groups.CreateGroup(name) to update domain.

For our custom domain events delivered as tea.Msg: e.g. when RepoDiscoveredMsg arrives (sent via event bus), the Update method will handle it by adding the repo to the model’s list and maybe returning a command (like a tea.Cmd to trigger a Git status check command, though we chose to have GitService handle status externally in this design).

The View() method of the model will render the current state into text UI using Bubbles components (like rendering groups and repos in a list, showing a spinner if scanning, etc.).

Shutdown: After p.Start() returns (which happens when the user quits the TUI), the program can perform cleanup. This is where we call cancel() on the context to stop background goroutines. We also ensure to save any updated config (for example, if the user created new groups during the session, we should call configSvc.Save(updatedCfg) before exiting, or have GroupManager or ConfigService handle that via events when changes occur).

Throughout the running of the app, the message routing is largely handled by Bubbletea and our EventBus:

Bubbletea takes care of delivering user input (keys, window resize, etc.) to our Update method as messages.

Our event bus takes care of delivering domain events to whoever is interested. The UI model’s Update can receive those events as messages (via the bridging we set up).

We maintain a single source of truth for state in the UI model. The domain services don’t directly mutate UI state; they only send events. The UI decides how to apply those events to its state representation. This follows the Elm architecture style (unidirectional data flow: domain -> UI via messages, and user -> domain via messages) which Bubbletea encourages.

Since the UI is decoupled, we could even run parts of the system headlessly for testing (for instance, simulate DiscoveryService and GitService running, feed events to an in-memory EventBus, and verify the GroupManager or ConfigService behave correctly, all without a real TUI).

First Steps of Implementation

To tackle this architectural rewrite, here’s a step-by-step plan to proceed in an incremental and testable way:

Skeleton Setup: Create the basic project structure with Go modules. Set up the cmd/gitagrip/main.go and internal/ package folders as outlined. Write a trivial main.go that prints a hello or launches a dummy Bubbletea model (for sanity check). Run go mod init gitagrip (or appropriate module path) and get the basic project building. This ensures all package imports and module paths are correctly set before adding complexity.

Define Domain Types & Interfaces: Start by defining the core domain structs and event types (in internal/domain). Write down Repository, RepoStatus, etc., and event structs like RepoDiscoveredEvent, StatusUpdatedEvent, etc., as shown in the examples. Define interface types for services (DiscoveryService, GitService, etc.). At this stage, you can also define the EventBus interface and maybe a simple implementation (even if it’s not fully asynchronous yet – could start with a synchronous stub that immediately calls handlers, then refine).

Implement the EventBus: In internal/eventbus, create a concrete implementation of the publish-subscribe mechanism. This might involve:

An internal map of subscribers: map[EventType][]chan DomainEvent or map[EventType][]EventHandler.

Subscribe(eventType, handler) method that registers a handler (you may decide to use channels: e.g., return a channel that the subscriber can range over, or accept a callback function).

Publish(event) method that sends the event to all subscribers. Important: ensure this does not block indefinitely if no subscriber is listening. Using buffered channels or doing publishes in a new goroutine can prevent deadlocks if a subscriber is slow. Alternatively, a simple solution is to handle subscription by always using buffered channels or have a dedicated dispatch goroutine inside EventBus (which listens on a channel of incoming events and fans them out).

Write tests for the EventBus: e.g., subscribe a handler to an event, publish that event, and verify the handler was called with the right data.

Build ConfigService: Implement internal/config next. Decide on a format (say JSON in ~/.gitagrip.json). Implement Load() to read the file (or create defaults if not exists) and Save() to write. Keep it simple (you can add more fields as needed later). After loading, emit a ConfigLoaded event (via EventBus) with the data, or simply return the data to main. Test that loading and saving round-trip correctly using a temp file in config_test.go.

Stub out Services: Create initial scaffolding for DiscoveryService, GitService, and GroupManager:

For now, they can be minimal: e.g. DiscoveryService might have a StartScan that immediately returns (or does a very simple directory listing without recursion, just to test wiring). GitService’s RefreshRepo can just return a dummy RepoStatus. The idea is to get the plumbing working before the heavy logic.

Ensure these services register with the EventBus for the events they need to handle or emit. For example, DiscoveryService should subscribe to a ScanRequested event (if using that) so that when UI requests a scan, it kicks off. Or if you prefer direct calls, maybe skip that and call StartScan directly from UI in the initial version.

GroupManager can start with just storing groups in memory (perhaps initialize from config’s Groups map) and have a no-op CreateGroup that just adds to its map and publishes GroupAdded event.

By stubbing, you can already integrate them with UI and test the flow without full functionality.

Basic UI Integration: Begin implementing the internal/ui package with a simple Bubbletea model:

Use the Bubbletea framework to create a Model struct that holds, say, a list of repository names (initially empty or “No repos yet”) and possibly a status message.

Implement the Init() method of the model to perhaps send a command or initial message. For example, you might return a tea.Cmd that sends a “start scanning” message, or you can directly call the DiscoveryService to start (via event bus or direct method).

Implement a basic Update(msg tea.Msg) that can handle two things: key presses for quitting (and maybe a refresh key), and the custom domain events (RepoDiscoveredMsg, StatusUpdatedMsg, etc.). At first, just handle quitting and maybe a dummy message to prove it works.

Implement View() to display some static content (like a title "Gitagrip" and an empty placeholder list).

Launch this UI from main.go and verify that you can open the TUI, see the placeholder, and quit with 'q'. This ensures the Bubbletea loop is running. With this in place, you have a foundation to build on.

Connect UI with EventBus: Now wire the real events into the UI. For example:

When a RepoDiscovered event is published on EventBus, the UI needs to get a RepoDiscoveredMsg. One approach: in main.go, after creating the tea Program (but before starting it), subscribe the UI to events. If we have bus.Subscribe(EventRepoDiscovered, handler), we can make the handler do prog.Send(RepoDiscoveredMsg{...}). Since prog.Send is thread-safe, this will inject the message into Bubbletea. (If the program isn’t started yet, you might start the program in a goroutine or structure differently. Alternatively, bubbletea allows you to supply an initial command via InitialModel.Init() so you could pass the bus to the model and have the model spawn a goroutine that reads from bus channels.)

Another simpler way: inside the UI’s Update, handle domain events that might be stored on some channel. For example, you could have a global or package-level channel that the EventBus uses for UI events. But a global channel is less clean than using EventBus callbacks.

Choose a method and implement it so that when DiscoveryService publishes RepoDiscovered, the UI’s Update actually receives a RepoDiscoveredMsg. Then fill in the Update logic to add the repo to the model’s list (you might maintain []Repository in the model, or separate lists per group).

Similarly, handle StatusUpdatedMsg to update the status of a repo in the model’s state (finding it by path or ID and updating its status field).

You can print logs or temporary text on the UI to confirm that these messages are being received and processed (for debugging).

Implement DiscoveryService logic: Now that the skeleton is in place, flesh out the DiscoveryService to actually scan the filesystem:

Use filepath.WalkDir or similar to traverse the user-specified directories (from config) and find any directory containing a .git subfolder. For each such directory, create a Repository struct (set Path, maybe Name as the folder name, and default Group if any logic for that – likely ungrouped initially).

Whenever a repo is found, publish a RepoDiscovered event via EventBus. (To avoid flooding too fast, ensure the EventBus can handle bursts or consider introducing a slight delay if needed – but probably fine as is.)

This scanning should run in a goroutine so the UI isn’t blocked. It might be useful to signal scanning progress via an event or set a flag in the UI state. For example, send a ScanStarted event when beginning and ScanCompleted when done, so the UI can show/hide a loading indicator.

Also consider: if a scan is triggered while another is running, either cancel the previous (use context) or skip if one in progress. Manage concurrent scans carefully (maybe a simple mutex or atomic flag in DiscoveryService to ensure only one runs at a time, or queue them).

Test the Discovery logic independently if possible (e.g. using a temporary directory structure with some fake git dirs).

Implement GitService logic: Similarly, develop the GitService:

Decide how to get repo status. Easiest might be to call the git executable (using os/exec) and parse its output for branch and status. Alternatively, use a Go library like go-git for programmatic access. For simplicity, shelling out to git status --porcelain and git rev-parse for branch might suffice.

Ensure these calls are done asynchronously. You might have GitService maintain a worker goroutine or use go func(){ ... }() whenever a RepoDiscovered comes in. But be mindful of too many concurrent git processes if a huge number of repos – maybe limit concurrency (e.g., use a buffered channel as a semaphore).

When a status is obtained, publish StatusUpdated event. Also consider periodic updates: maybe set a timer to re-check each repo every N minutes, or provide a manual refresh action in UI (user presses "R" for refresh all, which triggers GitService to refresh all known repos).

Also implement logic for when a repository is first discovered: likely you want an immediate status check (the RepoDiscovered event triggers it as described). Ensure to handle errors (if a repo is in an odd state or git not installed, etc.).

Group Management & Config Persistence: Now use the GroupManager to handle grouping commands from UI:

When the user creates or deletes a group in the UI, call the GroupManager’s methods (or send events that GroupManager subscribes to, e.g., a CreateGroupRequested event). The GroupManager should update its internal state (e.g., add new group to its map) and then:

Publish a GroupAdded event (with group name and maybe initial members if any).

Trigger a config save. This could be done by simply calling configService.Save() with the updated Config (GroupManager can hold a reference to ConfigService or better, emit an event ConfigChanged that ConfigService listens to in order to perform the save).

The UI upon receiving GroupAdded event will add a new section in its view for that group. If the event includes which repos belong (if any), it can arrange them. Initially, new group might be empty until user assigns repos.

Provide a way in UI to move a repo to a group (e.g., select a repo, press some key to choose a target group). That action would result in calling GroupManager (or an event). GroupManager would update its mapping (perhaps moving the repo’s path from one group list to another) and emit RepoMovedToGroup (or simply another GroupAdded/Removed combination). The UI can then update its list accordingly. Also save config (so that grouping is persisted).

Test that group creation and deletion events properly reflect in UI and config file.

Polish the UI: Now that the core logic is done, focus on improving the UI/UX:

Use the Bubbles library components for a better interface. For example, use bubbles/list to display repositories. You could have one list per group, or one list that is sectioned by group. Alternatively, use a bubbles/table if tabular display of status is desired. Since Bubbles components can be combined, you might structure the UI Model to contain multiple sub-models: e.g., a model for the list of repos, a model for a text input when adding group name, etc.

Implement navigation keys (arrow up/down to move through list, maybe left/right to collapse groups if you implement that).

Show a status bar or footer (maybe with help menu from bubbles/help showing key bindings).

Show an indicator for background tasks: e.g., a spinner (from bubbles/spinner) that is active when scanning or refreshing is in progress. You can tie this to an internal counter or boolean (e.g., increment when scan starts, decrement when scan finishes; if >0, show spinner).

Ensure resizing the terminal reflows content nicely (Bubbletea handles basic resizing if you use the viewport or flexbox-like layouts from Lipgloss).

No need to do all polish at once, but a clear separation means this can be done last without affecting the underlying logic.

Testing: As you implement each piece, add tests:

Services: test that DiscoveryService correctly finds known repos (you can simulate by creating temp dirs with dummy “.git” folders).

EventBus: already tested earlier, ensure no race conditions (maybe use go test -race).

GroupManager: test that adding/removing groups updates internal state and triggers events (you might inject a mock event bus that records events).

UI: testing TUIs is tricky, but you can test the model’s Update function in isolation. Bubbletea’s teatest package or simply calling Update with sample messages can assert that state changes as expected (e.g., send a RepoDiscoveredMsg to the model and check that model’s repo list length increased).

By following these steps, you will incrementally build up the new Gitagrip architecture. At each stage, you’ll have a running application (even if with limited functionality), which helps avoid big bang rewrites that don’t run until the end. The end result will be a Go TUI application that is idiomatic, robust, and much easier to maintain or extend. New features (for example, sorting repositories, or integrating a fetch/pull command) can be added by introducing new events and handlers in the relevant service, without having to tangle through monolithic spaghetti code. Each package has a single responsibility and minimal knowledge of the others, primarily interacting through the event bus and shared domain models.

This design should make it straightforward for contributors to dive in: they can, for instance, work on improving internal/git (knowing it only affects how statuses are retrieved and events emitted) or tweak the internal/ui (knowing they can rely on events for data). The clear boundaries and event-driven flow will help prevent unintended side effects when making changes, fulfilling the goal of an easy-to-understand and change architecture.
