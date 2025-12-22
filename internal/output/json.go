package output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"audiosort/pkg/models"
)

// JSONWriter writes metadata.json files
type JSONWriter struct {
	basePath string
}

// NewJSONWriter creates a new JSON metadata writer
func NewJSONWriter(basePath string) *JSONWriter {
	return &JSONWriter{basePath: basePath}
}

// Write generates and writes a JSON metadata file
func (w *JSONWriter) Write(ctx context.Context, book *models.Audiobook, destPath string) error {
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

	// Marshal metadata to JSON with indentation
	jsonData, err := json.MarshalIndent(book.Metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Build full destination path
	fullPath := filepath.Join(w.basePath, destPath)

	// Create destination directory if needed
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	jsonPath := filepath.Join(fullPath, "metadata.json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}
