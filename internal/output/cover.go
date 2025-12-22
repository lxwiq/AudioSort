package output

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"audiosort/pkg/models"
)

// CoverWriter downloads and saves cover images
type CoverWriter struct {
	basePath string
	client   *http.Client
}

// NewCoverWriter creates a new cover image writer
func NewCoverWriter(basePath string, client *http.Client) *CoverWriter {
	if client == nil {
		client = &http.Client{}
	}
	return &CoverWriter{
		basePath: basePath,
		client:   client,
	}
}

// Write downloads and saves a cover image
func (w *CoverWriter) Write(ctx context.Context, book *models.Audiobook, destPath string) error {
	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate input
	if book == nil || book.Metadata == nil {
		return fmt.Errorf("book or metadata is nil")
	}

	// Check if cover URL is available
	if book.Metadata.CoverURL == "" {
		return nil // Not an error, just no cover available
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", book.Metadata.CoverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid some rate limiting
	req.Header.Set("User-Agent", "AudioSort/1.0")

	// Download the image
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download cover: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download cover: HTTP %d", resp.StatusCode)
	}

	// Build full destination path
	fullPath := filepath.Join(w.basePath, destPath)

	// Create destination directory if needed
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the cover file
	coverPath := filepath.Join(fullPath, "cover.jpg")
	coverFile, err := os.Create(coverPath)
	if err != nil {
		return fmt.Errorf("failed to create cover file: %w", err)
	}

	// Use a cleanup flag to handle errors
	success := false
	defer func() {
		coverFile.Close()
		if !success {
			os.Remove(coverPath) // Clean up partial file on error
		}
	}()

	// Copy the image data
	if _, err := io.Copy(coverFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write cover file: %w", err)
	}

	// Sync to ensure data is written
	if err := coverFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync cover file: %w", err)
	}

	success = true
	return nil
}
