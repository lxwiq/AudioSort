package core

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"audiosort/pkg/models"

	"github.com/dhowden/tag"
)

var audioExtensions = map[string]bool{
	".mp3":  true,
	".m4a":  true,
	".m4b":  true,
	".flac": true,
	".ogg":  true,
	".wma":  true,
}

// Scanner recursively scans directories for audiobook folders
type Scanner struct {
	workers int
}

// NewScanner creates a new scanner with the specified number of workers
func NewScanner(workers int) *Scanner {
	if workers <= 0 {
		workers = 4
	}
	return &Scanner{workers: workers}
}

// Scan recursively scans the root directory for audiobook folders
// Returns a channel that streams found audiobooks
func (s *Scanner) Scan(ctx context.Context, root string) (<-chan models.Audiobook, error) {
	// Validate root path exists
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, models.ErrInvalidPath
	}

	results := make(chan models.Audiobook)

	go func() {
		defer close(results)
		s.scanDirectory(ctx, root, results)
	}()

	return results, nil
}

// scanDirectory walks the directory tree and identifies audiobook folders
func (s *Scanner) scanDirectory(ctx context.Context, root string, results chan<- models.Audiobook) {
	// Use a semaphore to limit concurrent directory processing
	semaphore := make(chan struct{}, s.workers)
	var wg sync.WaitGroup

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}

		if err != nil {
			return nil // Skip directories with errors
		}

		if !d.IsDir() {
			return nil
		}

		// Check if this directory contains audio files
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(dirPath string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			audiobook := s.analyzeDirectory(ctx, dirPath)
			if audiobook != nil {
				select {
				case results <- *audiobook:
				case <-ctx.Done():
					return
				}
			}
		}(path)

		return nil
	})

	// Wait for all workers to complete
	wg.Wait()
}

// analyzeDirectory checks if a directory is an audiobook folder and extracts file info
func (s *Scanner) analyzeDirectory(ctx context.Context, dirPath string) *models.Audiobook {
	// Check context
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil
	}

	var audioFiles []models.AudioFile

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if audioExtensions[ext] {
			fullPath := filepath.Join(dirPath, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			audioFiles = append(audioFiles, models.AudioFile{
				Path:     fullPath,
				Size:     info.Size(),
				Duration: 0, // Duration will be extracted later if needed
			})
		}
	}

	// Return this directory as an audiobook if it contains audio files
	if len(audioFiles) > 0 {
		return &models.Audiobook{
			Path:   dirPath,
			Files:  audioFiles,
			Probe:  readProbe(audioFiles[0].Path),
			Status: models.StatusPending,
		}
	}

	return nil
}

// readProbe reads embedded tags from a single audio file to derive hints about
// the book. It reads only the tag header (not the whole file). Returns nil when
// no useful tags are present.
func readProbe(path string) *models.Probe {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil
	}

	// For audiobooks the book title is usually the album (each file is a
	// chapter/part); the author is usually the album-artist.
	title := m.Album()
	if title == "" {
		title = m.Title()
	}
	author := m.AlbumArtist()
	if author == "" {
		author = m.Artist()
	}

	year := ""
	if y := m.Year(); y > 0 {
		year = strconv.Itoa(y)
	}

	if title == "" && author == "" {
		return nil
	}
	return &models.Probe{Title: title, Author: author, Year: year}
}
