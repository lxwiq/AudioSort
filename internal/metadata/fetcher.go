package metadata

import (
	"context"
	"net/http"
	"sync"
	"time"

	"audiosort/internal/cache"
	"audiosort/pkg/models"
)

// Fetcher orchestrates metadata retrieval from multiple sources
type Fetcher struct {
	sources []MetadataSource
	cache   *cache.Store
	client  *http.Client
}

// NewFetcher creates a new metadata fetcher with the given sources and cache
func NewFetcher(sources []MetadataSource, cacheStore *cache.Store) *Fetcher {
	return &Fetcher{
		sources: sources,
		cache:   cacheStore,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Fetch retrieves metadata for the given query
// It checks the cache first, then queries all sources in parallel
// and returns the first successful result
func (f *Fetcher) Fetch(ctx context.Context, query string) (*models.BookMetadata, error) {
	// Check cache first
	if f.cache != nil {
		if cached, ok := f.cache.GetMetadata(query); ok {
			return &cached.Metadata, nil
		}
	}

	// Fetch from all sources in parallel
	results := make(chan *models.BookMetadata, len(f.sources))
	var wg sync.WaitGroup

	for _, source := range f.sources {
		wg.Add(1)
		go func(s MetadataSource) {
			defer wg.Done()
			books, err := s.Search(ctx, query)
			if err == nil && len(books) > 0 {
				results <- &books[0]
			}
		}(source)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Return first result
	for result := range results {
		if f.cache != nil {
			f.cache.SetMetadata(query, *result)
		}
		return result, nil
	}

	return nil, models.ErrNoMetadataFound
}
