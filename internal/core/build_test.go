package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"audiosort/internal/config"
	"audiosort/pkg/models"
)

func TestProcessBooksHonoursSelectionAndStreamsProgress(t *testing.T) {
	p := NewPipeline(PipelineOptions{DryRun: true, Workers: 2})

	books := []models.Audiobook{
		{Path: "/tmp/a"},
		{Path: "/tmp/b"},
		{Path: "/tmp/c"},
	}

	progress := make(chan ProgressUpdate, len(books)+1)
	summary := p.ProcessBooks(context.Background(), books, progress)

	if summary.Total != 3 || summary.Processed != 3 {
		t.Fatalf("expected 3 books processed, got total=%d processed=%d", summary.Total, summary.Processed)
	}

	// The channel must be closed by ProcessBooks; the final update is 3/3.
	var last ProgressUpdate
	count := 0
	for u := range progress {
		last = u
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 progress updates, got %d", count)
	}
	if last.Done != 3 || last.Total != 3 {
		t.Fatalf("expected final progress 3/3, got %d/%d", last.Done, last.Total)
	}
}

func TestProcessBooksRespectsCancellation(t *testing.T) {
	p := NewPipeline(PipelineOptions{DryRun: true, Workers: 1})

	books := make([]models.Audiobook, 100)
	for i := range books {
		books[i].Path = fmt.Sprintf("/tmp/%d", i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before processing starts

	progress := make(chan ProgressUpdate, len(books)+1)
	done := make(chan *models.Summary, 1)
	go func() { done <- p.ProcessBooks(ctx, books, progress) }()

	select {
	case summary := <-done:
		if summary.Total > len(books) {
			t.Fatalf("processed more books than provided: %d", summary.Total)
		}
		// Draining the (closed) progress channel must not block.
		for range progress {
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ProcessBooks did not return after cancellation (possible deadlock)")
	}
}

func TestMetadataSourcesRespectConfigOrder(t *testing.T) {
	cfg := &config.Config{Sources: []string{"googlebooks", "bookinfo"}}
	sources := MetadataSources(cfg)
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].Name() != "googlebooks" || sources[1].Name() != "bookinfo" {
		t.Fatalf("sources should follow config order, got %s, %s", sources[0].Name(), sources[1].Name())
	}
}

func TestMetadataSourcesFallbackIncludesBookInfo(t *testing.T) {
	cfg := &config.Config{Sources: nil}
	sources := MetadataSources(cfg)
	if len(sources) != len(config.AvailableSources) {
		t.Fatalf("expected fallback to %d sources, got %d", len(config.AvailableSources), len(sources))
	}
	// "bookinfo first" canonical order.
	if sources[0].Name() != "bookinfo" {
		t.Fatalf("fallback should query bookinfo first, got %s", sources[0].Name())
	}
}
