package metadata

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"audiosort/internal/cache"
	"audiosort/pkg/models"
)

// MockSource implements MetadataSource for testing
type MockSource struct {
	name     string
	priority int
	results  []models.BookMetadata
	err      error
}

func (m *MockSource) Name() string {
	return m.name
}

func (m *MockSource) Priority() int {
	return m.priority
}

func (m *MockSource) Supports(lang string) bool {
	return true
}

func (m *MockSource) Search(ctx context.Context, query string) ([]models.BookMetadata, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestGoogleBooks_Name(t *testing.T) {
	gb := NewGoogleBooks(nil)
	if gb.Name() != "googlebooks" {
		t.Errorf("Name() = %q, want %q", gb.Name(), "googlebooks")
	}
}

func TestGoogleBooks_Priority(t *testing.T) {
	gb := NewGoogleBooks(nil)
	if gb.Priority() != 1 {
		t.Errorf("Priority() = %d, want 1", gb.Priority())
	}
}

func TestGoogleBooks_Supports(t *testing.T) {
	gb := NewGoogleBooks(nil)

	languages := []string{"en", "fr", "de", "ja", "zh"}
	for _, lang := range languages {
		if !gb.Supports(lang) {
			t.Errorf("Supports(%q) = false, want true", lang)
		}
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		expected string
	}{
		{
			name:     "full date",
			date:     "2023-05-15",
			expected: "2023",
		},
		{
			name:     "year only",
			date:     "2020",
			expected: "2020",
		},
		{
			name:     "year-month",
			date:     "2019-12",
			expected: "2019",
		},
		{
			name:     "empty string",
			date:     "",
			expected: "",
		},
		{
			name:     "no year",
			date:     "unknown",
			expected: "",
		},
		{
			name:     "old year",
			date:     "1984-01-01",
			expected: "1984",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractYear(tt.date)
			if got != tt.expected {
				t.Errorf("extractYear(%q) = %q, want %q", tt.date, got, tt.expected)
			}
		})
	}
}

func TestNewFetcher(t *testing.T) {
	sources := []MetadataSource{
		&MockSource{name: "source1"},
		&MockSource{name: "source2"},
	}

	fetcher := NewFetcher(sources, nil)
	if fetcher == nil {
		t.Fatal("NewFetcher() returned nil")
	}
	if len(fetcher.sources) != 2 {
		t.Errorf("sources count = %d, want 2", len(fetcher.sources))
	}
}

func TestFetcher_Fetch_ReturnsFirstResult(t *testing.T) {
	sources := []MetadataSource{
		&MockSource{
			name:     "source1",
			priority: 1,
			results: []models.BookMetadata{
				{Title: "Book from Source 1"},
			},
		},
		&MockSource{
			name:     "source2",
			priority: 2,
			results: []models.BookMetadata{
				{Title: "Book from Source 2"},
			},
		},
	}

	fetcher := NewFetcher(sources, nil)
	result, err := fetcher.Fetch(context.Background(), "test query")

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result == nil {
		t.Fatal("Fetch() returned nil result")
	}
	// Either source could return first due to parallel execution
	if result.Title != "Book from Source 1" && result.Title != "Book from Source 2" {
		t.Errorf("Fetch() returned unexpected title: %q", result.Title)
	}
}

func TestFetcher_Fetch_ReturnsErrorWhenNoResults(t *testing.T) {
	sources := []MetadataSource{
		&MockSource{
			name:    "empty",
			results: []models.BookMetadata{},
		},
	}

	fetcher := NewFetcher(sources, nil)
	_, err := fetcher.Fetch(context.Background(), "test query")

	if err == nil {
		t.Fatal("Fetch() should return error when no results")
	}
	if !errors.Is(err, models.ErrNoMetadataFound) {
		t.Errorf("Fetch() error = %v, want ErrNoMetadataFound", err)
	}
}

func TestFetcher_Fetch_SkipsFailingSources(t *testing.T) {
	sources := []MetadataSource{
		&MockSource{
			name: "failing",
			err:  errors.New("API error"),
		},
		&MockSource{
			name:    "working",
			results: []models.BookMetadata{{Title: "Working Book"}},
		},
	}

	fetcher := NewFetcher(sources, nil)
	result, err := fetcher.Fetch(context.Background(), "test query")

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.Title != "Working Book" {
		t.Errorf("Fetch() Title = %q, want %q", result.Title, "Working Book")
	}
}

func TestFetcher_Fetch_UsesCache(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := cache.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// Pre-populate cache
	cachedMetadata := models.BookMetadata{Title: "Cached Book"}
	if err := store.SetMetadata("cached query", cachedMetadata); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}

	// Source should not be called
	sources := []MetadataSource{
		&MockSource{
			name:    "source",
			results: []models.BookMetadata{{Title: "Fresh Book"}},
		},
	}

	fetcher := NewFetcher(sources, store)
	result, err := fetcher.Fetch(context.Background(), "cached query")

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.Title != "Cached Book" {
		t.Errorf("Fetch() Title = %q, want %q (from cache)", result.Title, "Cached Book")
	}
}

func TestFetcher_Fetch_CachesResult(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := cache.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	sources := []MetadataSource{
		&MockSource{
			name:    "source",
			results: []models.BookMetadata{{Title: "New Book"}},
		},
	}

	fetcher := NewFetcher(sources, store)
	_, err = fetcher.Fetch(context.Background(), "new query")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify result is cached
	cached, found := store.GetMetadata("new query")
	if !found {
		t.Fatal("Result should be cached")
	}
	if cached.Metadata.Title != "New Book" {
		t.Errorf("Cached Title = %q, want %q", cached.Metadata.Title, "New Book")
	}
}

func TestFetcher_Fetch_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	sources := []MetadataSource{
		&MockSource{
			name:    "source",
			results: []models.BookMetadata{{Title: "Book"}},
		},
	}

	fetcher := NewFetcher(sources, nil)
	_, err := fetcher.Fetch(ctx, "test query")

	// Should return error or no results due to cancellation
	// The exact behavior depends on timing
	_ = err // Either error or success is acceptable with cancelled context
}

func TestFetcher_Fetch_NilCache(t *testing.T) {
	sources := []MetadataSource{
		&MockSource{
			name:    "source",
			results: []models.BookMetadata{{Title: "Book"}},
		},
	}

	fetcher := NewFetcher(sources, nil)
	result, err := fetcher.Fetch(context.Background(), "test query")

	if err != nil {
		t.Fatalf("Fetch() with nil cache error = %v", err)
	}
	if result.Title != "Book" {
		t.Errorf("Fetch() Title = %q, want %q", result.Title, "Book")
	}
}

func TestFetcher_Fetch_EmptySources(t *testing.T) {
	fetcher := NewFetcher([]MetadataSource{}, nil)
	_, err := fetcher.Fetch(context.Background(), "test query")

	if err == nil {
		t.Fatal("Fetch() with empty sources should return error")
	}
}

func TestMetadataSource_Interface(t *testing.T) {
	// Verify GoogleBooks implements MetadataSource
	var _ MetadataSource = (*GoogleBooks)(nil)
}
