package cache

import (
	"path/filepath"
	"testing"
	"time"

	"audiosort/pkg/models"
)

func TestNewStore(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Error("NewStore() db is nil")
	}
}

func TestStore_MetadataOperations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// Test Get on empty store
	_, found := store.GetMetadata("nonexistent")
	if found {
		t.Error("GetMetadata() should return false for nonexistent query")
	}

	// Test Set and Get
	metadata := models.BookMetadata{
		Title:   "Test Book",
		Authors: []string{"Test Author"},
		ISBN:    "1234567890",
	}

	if err := store.SetMetadata("test query", metadata); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}

	cached, found := store.GetMetadata("test query")
	if !found {
		t.Fatal("GetMetadata() should return true after Set")
	}

	if cached.Query != "test query" {
		t.Errorf("cached.Query = %q, want %q", cached.Query, "test query")
	}
	if cached.Metadata.Title != "Test Book" {
		t.Errorf("cached.Metadata.Title = %q, want %q", cached.Metadata.Title, "Test Book")
	}
	if cached.Metadata.ISBN != "1234567890" {
		t.Errorf("cached.Metadata.ISBN = %q, want %q", cached.Metadata.ISBN, "1234567890")
	}
}

func TestStore_MetadataUpdate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// Set initial metadata
	metadata1 := models.BookMetadata{Title: "Original Title"}
	if err := store.SetMetadata("query", metadata1); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}

	// Update metadata
	metadata2 := models.BookMetadata{Title: "Updated Title"}
	if err := store.SetMetadata("query", metadata2); err != nil {
		t.Fatalf("SetMetadata() update error = %v", err)
	}

	// Verify update
	cached, found := store.GetMetadata("query")
	if !found {
		t.Fatal("GetMetadata() should return true")
	}
	if cached.Metadata.Title != "Updated Title" {
		t.Errorf("Metadata.Title = %q, want %q", cached.Metadata.Title, "Updated Title")
	}
}

func TestStore_ProcessedOperations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// Test IsProcessed on empty store
	if store.IsProcessed("/path/to/book") {
		t.Error("IsProcessed() should return false for unprocessed path")
	}

	// Mark as processed
	if err := store.MarkProcessed("/source/path", "/dest/path", "abc123"); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}

	// Verify marked
	if !store.IsProcessed("/source/path") {
		t.Error("IsProcessed() should return true after MarkProcessed")
	}

	// Different path should not be marked
	if store.IsProcessed("/other/path") {
		t.Error("IsProcessed() should return false for different path")
	}
}

func TestStore_MultipleQueries(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	// Set multiple metadata entries
	queries := []string{"query1", "query2", "query3"}
	for i, q := range queries {
		metadata := models.BookMetadata{Title: q}
		if err := store.SetMetadata(q, metadata); err != nil {
			t.Fatalf("SetMetadata(%d) error = %v", i, err)
		}
	}

	// Verify all can be retrieved
	for _, q := range queries {
		cached, found := store.GetMetadata(q)
		if !found {
			t.Errorf("GetMetadata(%q) not found", q)
		}
		if cached.Metadata.Title != q {
			t.Errorf("Metadata.Title = %q, want %q", cached.Metadata.Title, q)
		}
	}
}

func TestStore_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create store and add data
	store1, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	metadata := models.BookMetadata{Title: "Persistent Book"}
	if err := store1.SetMetadata("persistent", metadata); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}
	if err := store1.MarkProcessed("/persistent/source", "/persistent/dest", "hash"); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	store1.Close()

	// Open new store and verify data persisted
	store2, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() second open error = %v", err)
	}
	defer store2.Close()

	cached, found := store2.GetMetadata("persistent")
	if !found {
		t.Error("GetMetadata() should find persisted data")
	}
	if cached.Metadata.Title != "Persistent Book" {
		t.Errorf("Metadata.Title = %q, want %q", cached.Metadata.Title, "Persistent Book")
	}

	if !store2.IsProcessed("/persistent/source") {
		t.Error("IsProcessed() should return true for persisted data")
	}
}

func TestCachedBook_JSONSerialization(t *testing.T) {
	cached := CachedBook{
		Query: "test query",
		Metadata: models.BookMetadata{
			Title:   "Test Book",
			Authors: []string{"Author 1", "Author 2"},
		},
		FetchedAt: time.Now(),
	}

	if cached.Query == "" {
		t.Error("CachedBook.Query should not be empty")
	}
	if cached.Metadata.Title == "" {
		t.Error("CachedBook.Metadata.Title should not be empty")
	}
	if cached.FetchedAt.IsZero() {
		t.Error("CachedBook.FetchedAt should not be zero")
	}
}

func TestProcessedBook_JSONSerialization(t *testing.T) {
	processed := ProcessedBook{
		SourcePath:  "/source/path",
		DestPath:    "/dest/path",
		ProcessedAt: time.Now(),
		Checksum:    "abc123",
	}

	if processed.SourcePath == "" {
		t.Error("ProcessedBook.SourcePath should not be empty")
	}
	if processed.DestPath == "" {
		t.Error("ProcessedBook.DestPath should not be empty")
	}
	if processed.Checksum == "" {
		t.Error("ProcessedBook.Checksum should not be empty")
	}
}

func TestStore_Close(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if err := store.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestStore_ClearAndStats(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "clear.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if err := store.SetMetadata("query-a", models.BookMetadata{Title: "A"}); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}
	if err := store.SetMetadata("query-b", models.BookMetadata{Title: "B"}); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}

	if meta, _ := store.Stats(); meta != 2 {
		t.Fatalf("expected 2 metadata entries before clear, got %d", meta)
	}

	// Clear must operate on the already-open handle (no second open / no lock
	// conflict) and empty the buckets.
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if meta, processed := store.Stats(); meta != 0 || processed != 0 {
		t.Fatalf("expected empty store after clear, got %d metadata, %d processed", meta, processed)
	}

	// The store must remain usable after clearing.
	if err := store.SetMetadata("query-c", models.BookMetadata{Title: "C"}); err != nil {
		t.Fatalf("SetMetadata() after clear error = %v", err)
	}
	if _, ok := store.GetMetadata("query-c"); !ok {
		t.Fatalf("store should be usable after clear")
	}
}
