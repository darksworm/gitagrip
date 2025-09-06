package query

import (
	"gitagrip/internal/domain"
)

// IndexType represents what type of item is at an index
type IndexType int

const (
	IndexTypeGroup IndexType = iota
	IndexTypeRepository
	IndexTypeUngroupedHeader
)

// IndexInfo contains information about what's at a specific index
type IndexInfo struct {
	Type       IndexType
	GroupName  string // For groups and repos in groups
	Repository *domain.Repository
	Path       string // Repository path
}