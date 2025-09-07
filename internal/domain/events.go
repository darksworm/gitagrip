package domain

// EventType represents the type of domain event
type EventType string

// Event types
const (
	EventRepoDiscovered EventType = "RepoDiscovered"
	EventStatusUpdated  EventType = "StatusUpdated"
	EventError          EventType = "Error"
	EventGroupAdded     EventType = "GroupAdded"
	EventGroupRemoved   EventType = "GroupRemoved"
	EventRepoMoved      EventType = "RepoMoved"
	EventScanStarted    EventType = "ScanStarted"
	EventScanCompleted  EventType = "ScanCompleted"
	EventScanRequested  EventType = "ScanRequested"
	EventStatusRefreshRequested EventType = "StatusRefreshRequested"
	EventFetchRequested EventType = "FetchRequested"
	EventPullRequested  EventType = "PullRequested"
	EventFetchCompleted EventType = "FetchCompleted"
	EventPullCompleted  EventType = "PullCompleted"
	EventConfigLoaded   EventType = "ConfigLoaded"
	EventConfigSaved    EventType = "ConfigSaved"
	EventConfigChanged  EventType = "ConfigChanged"
	EventAppReady       EventType = "AppReady"
)

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	Type() EventType
}

// RepoDiscoveredEvent is emitted when a new repository is found
type RepoDiscoveredEvent struct {
	Repo Repository
}

func (e RepoDiscoveredEvent) Type() EventType { return EventRepoDiscovered }

// StatusUpdatedEvent is emitted when a repository's status is updated
type StatusUpdatedEvent struct {
	RepoPath string
	Status   RepoStatus
}

func (e StatusUpdatedEvent) Type() EventType { return EventStatusUpdated }

// ErrorEvent is emitted when an error occurs
type ErrorEvent struct {
	Message string
	Err     error
}

func (e ErrorEvent) Type() EventType { return EventError }

// GroupAddedEvent is emitted when a new group is created
type GroupAddedEvent struct {
	Name string
}

func (e GroupAddedEvent) Type() EventType { return EventGroupAdded }

// GroupRemovedEvent is emitted when a group is deleted
type GroupRemovedEvent struct {
	Name string
}

func (e GroupRemovedEvent) Type() EventType { return EventGroupRemoved }

// RepoMovedEvent is emitted when a repository is moved to a different group
type RepoMovedEvent struct {
	RepoPath string
	FromGroup string
	ToGroup   string
}

func (e RepoMovedEvent) Type() EventType { return EventRepoMoved }

// ScanStartedEvent is emitted when repository scanning begins
type ScanStartedEvent struct {
	Paths []string
}

func (e ScanStartedEvent) Type() EventType { return EventScanStarted }

// ScanCompletedEvent is emitted when repository scanning completes
type ScanCompletedEvent struct {
	ReposFound int
}

func (e ScanCompletedEvent) Type() EventType { return EventScanCompleted }

// ScanRequestedEvent is emitted to request a new scan
type ScanRequestedEvent struct {
	Paths []string
}

func (e ScanRequestedEvent) Type() EventType { return EventScanRequested }

// ConfigLoadedEvent is emitted when configuration is loaded
type ConfigLoadedEvent struct {
	BaseDir string
	Groups  map[string][]string
}

func (e ConfigLoadedEvent) Type() EventType { return EventConfigLoaded }

// ConfigSavedEvent is emitted when configuration is saved
type ConfigSavedEvent struct{}

func (e ConfigSavedEvent) Type() EventType { return EventConfigSaved }

// ConfigChangedEvent is emitted when configuration needs to be saved
type ConfigChangedEvent struct {
	Groups map[string][]string // Current group configuration
}

func (e ConfigChangedEvent) Type() EventType { return EventConfigChanged }

// StatusRefreshRequestedEvent is emitted to request status refresh for specific repositories
type StatusRefreshRequestedEvent struct {
	RepoPaths []string // Empty means refresh all
}

func (e StatusRefreshRequestedEvent) Type() EventType { return EventStatusRefreshRequested }

// FetchRequestedEvent is emitted to request git fetch for specific repositories
type FetchRequestedEvent struct {
	RepoPaths []string // Empty means fetch all
}

func (e FetchRequestedEvent) Type() EventType { return EventFetchRequested }

// PullRequestedEvent is emitted to request git pull for specific repositories
type PullRequestedEvent struct {
	RepoPaths []string // Empty means pull all
}

func (e PullRequestedEvent) Type() EventType { return EventPullRequested }

// FetchCompletedEvent is emitted when git fetch operation completes
type FetchCompletedEvent struct {
	RepoPath string
	Success  bool
	Error    error
}

func (e FetchCompletedEvent) Type() EventType { return EventFetchCompleted }

// PullCompletedEvent is emitted when git pull operation completes
type PullCompletedEvent struct {
	RepoPath string
	Success  bool
	Error    error
}

func (e PullCompletedEvent) Type() EventType { return EventPullCompleted }

// AppReadyEvent is emitted when the app is fully initialized and ready
type AppReadyEvent struct {
	HasExistingConfig bool
}

func (e AppReadyEvent) Type() EventType { return EventAppReady }