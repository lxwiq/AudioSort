package metadata

import (
	"context"

	"audiosort/pkg/models"
)

// MetadataSource defines the interface for metadata providers
type MetadataSource interface {
	// Name returns the identifier of the source
	Name() string

	// Search queries the source for book metadata
	Search(ctx context.Context, query string) ([]models.BookMetadata, error)

	// Priority determines the order in which sources are queried (lower is higher priority)
	Priority() int

	// Supports checks if the source supports a given language
	Supports(language string) bool
}
