package domain

// Repository represents a git repository
type Repository struct {
	Path        string
	Name        string
	DisplayName string // Name shown in UI, may include path for duplicates
	Group       string // group name it belongs to ("" if ungrouped)
	Status      RepoStatus
}

// RepoStatus represents the current status of a repository
type RepoStatus struct {
	Branch          string
	AheadCount      int
	BehindCount     int
	Uncommitted     int // number of unstaged/uncommitted changes
	UnpushedCommits int // commits ahead of remote
	IsDirty         bool
	HasUntracked    bool
	StashCount      int    // number of stashed changes
	Error           string // error message if status check failed
}

// Group represents a collection of repositories
type Group struct {
	Name  string
	Repos []string // repository paths
}

// ScanProgress represents the current scanning state
type ScanProgress struct {
	IsScanning   bool
	ReposFound   int
	CurrentPath  string
}