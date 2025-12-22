package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"audiosort/pkg/models"
)

// Organizer handles moving or copying audiobook files to their destination
type Organizer struct {
	destPath  string
	copyMode  bool
	skipExist bool
	pattern   Pattern
}

// NewOrganizer creates a new organizer with the specified configuration
func NewOrganizer(destPath string, copyMode, skipExist bool, pattern Pattern) *Organizer {
	return &Organizer{
		destPath:  destPath,
		copyMode:  copyMode,
		skipExist: skipExist,
		pattern:   pattern,
	}
}

// Organize processes a single audiobook, moving or copying files to the destination
func (o *Organizer) Organize(ctx context.Context, book *models.Audiobook) models.Result {
	start := time.Now()

	// Check context
	select {
	case <-ctx.Done():
		return models.Result{
			Audiobook: book,
			Error:     ctx.Err(),
			Duration:  time.Since(start),
		}
	default:
	}

	// Validate audiobook
	if book == nil || book.Path == "" {
		return models.Result{
			Audiobook: book,
			Error:     models.ErrInvalidPath,
			Duration:  time.Since(start),
		}
	}

	// Check if metadata is available
	if book.Metadata == nil {
		book.Status = models.StatusError
		book.Error = "no metadata available"
		return models.Result{
			Audiobook: book,
			Error:     fmt.Errorf("no metadata available for %s", book.Path),
			Duration:  time.Since(start),
		}
	}

	// Build destination path using pattern
	detector := NewDetector()
	relativePath := detector.BuildPath(o.pattern, book.Metadata)
	destDir := filepath.Join(o.destPath, relativePath)

	// Check if destination exists
	if _, err := os.Stat(destDir); err == nil {
		if o.skipExist {
			book.Status = models.StatusSkipped
			book.Error = "destination already exists"
			return models.Result{
				Audiobook: book,
				Error:     nil, // Not an error, just skipped
				Duration:  time.Since(start),
			}
		}
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		book.Status = models.StatusError
		book.Error = err.Error()
		return models.Result{
			Audiobook: book,
			Error:     fmt.Errorf("failed to create destination directory: %w", err),
			Duration:  time.Since(start),
		}
	}

	// Process each audio file
	book.Status = models.StatusProcessing
	var errors []error

	for _, audioFile := range book.Files {
		select {
		case <-ctx.Done():
			book.Status = models.StatusError
			book.Error = "cancelled"
			return models.Result{
				Audiobook: book,
				Error:     ctx.Err(),
				Duration:  time.Since(start),
			}
		default:
		}

		// Get relative path within source directory
		relPath, err := filepath.Rel(book.Path, audioFile.Path)
		if err != nil {
			relPath = filepath.Base(audioFile.Path)
		}

		destFile := filepath.Join(destDir, relPath)

		// Create subdirectories if needed
		destFileDir := filepath.Dir(destFile)
		if err := os.MkdirAll(destFileDir, 0755); err != nil {
			errors = append(errors, fmt.Errorf("failed to create directory for %s: %w", relPath, err))
			continue
		}

		// Copy or move the file
		var fileErr error
		if o.copyMode {
			fileErr = o.copyFile(audioFile.Path, destFile)
		} else {
			fileErr = o.moveFile(audioFile.Path, destFile)
		}

		if fileErr != nil {
			errors = append(errors, fmt.Errorf("failed to process %s: %w", relPath, fileErr))
		}
	}

	// Also handle any non-audio files (covers, metadata, etc.)
	if err := o.copyAdditionalFiles(ctx, book.Path, destDir); err != nil {
		errors = append(errors, err)
	}

	// Update book status
	if len(errors) > 0 {
		book.Status = models.StatusError
		book.Error = fmt.Sprintf("%d errors occurred", len(errors))
		return models.Result{
			Audiobook: book,
			Error:     fmt.Errorf("processing completed with errors: %v", errors),
			Duration:  time.Since(start),
		}
	}

	book.Status = models.StatusDone
	return models.Result{
		Audiobook: book,
		Error:     nil,
		Duration:  time.Since(start),
	}
}

// copyFile copies a file from src to dst
func (o *Organizer) copyFile(src, dst string) error {
	// Check if destination exists
	if _, err := os.Stat(dst); err == nil {
		if o.skipExist {
			return nil // Skip existing files
		}
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Sync to ensure data is written
	if err := dstFile.Sync(); err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// moveFile moves a file from src to dst
func (o *Organizer) moveFile(src, dst string) error {
	// Check if destination exists
	if _, err := os.Stat(dst); err == nil {
		if o.skipExist {
			// Remove source if skip mode is on (consider it moved)
			return os.Remove(src)
		}
	}

	// Try atomic rename first (works if src and dst are on same filesystem)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename fails, fall back to copy + delete
	if err := o.copyFile(src, dst); err != nil {
		return err
	}

	return os.Remove(src)
}

// copyAdditionalFiles copies non-audio files from source to destination
func (o *Organizer) copyAdditionalFiles(ctx context.Context, srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.IsDir() {
			continue
		}

		// Skip audio files (already processed)
		ext := filepath.Ext(entry.Name())
		if audioExtensions[ext] {
			continue
		}

		// Copy metadata files, covers, etc.
		srcFile := filepath.Join(srcDir, entry.Name())
		dstFile := filepath.Join(dstDir, entry.Name())

		// Only copy certain file types
		if o.shouldCopyAdditionalFile(entry.Name()) {
			if err := o.copyFile(srcFile, dstFile); err != nil {
				// Don't fail the whole operation for additional files
				continue
			}
		}
	}

	return nil
}

// shouldCopyAdditionalFile determines if a non-audio file should be copied
func (o *Organizer) shouldCopyAdditionalFile(filename string) bool {
	// Common metadata and cover file extensions
	additionalExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".opf":  true,
		".txt":  true,
		".nfo":  true,
		".json": true,
		".xml":  true,
		".cue":  true,
	}

	ext := filepath.Ext(filename)
	return additionalExts[ext]
}
