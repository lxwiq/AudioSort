package core

import (
	"os"
	"path/filepath"
	"strings"

	"audiosort/pkg/models"
)

// Pattern represents an organization pattern for audiobook folders
type Pattern string

const (
	PatternAuthorTitle       Pattern = "author/title"
	PatternAuthorSeriesTitle Pattern = "author/series/title"
	PatternTitle             Pattern = "title"
	PatternSeriesTitle       Pattern = "series/title"
)

// Detector analyzes destination folders to detect organization patterns
type Detector struct{}

// NewDetector creates a new pattern detector
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes the destination path to determine the existing organization pattern
// Returns the detected pattern or a default pattern if none can be determined
func (d *Detector) Detect(destPath string) Pattern {
	// Check if destination exists
	info, err := os.Stat(destPath)
	if err != nil || !info.IsDir() {
		// Default to author/title pattern for new directories
		return PatternAuthorTitle
	}

	// Analyze existing structure
	entries, err := os.ReadDir(destPath)
	if err != nil {
		return PatternAuthorTitle
	}

	// Track pattern votes
	patterns := make(map[Pattern]int)

	// Analyze up to 20 directories to determine pattern
	analyzed := 0
	for _, entry := range entries {
		if !entry.IsDir() || analyzed >= 20 {
			continue
		}

		authorPath := filepath.Join(destPath, entry.Name())
		pattern := d.analyzeAuthorFolder(authorPath)
		if pattern != "" {
			patterns[pattern]++
			analyzed++
		}
	}

	// Return the most common pattern
	if len(patterns) == 0 {
		return PatternAuthorTitle
	}

	maxVotes := 0
	detectedPattern := PatternAuthorTitle
	for pattern, votes := range patterns {
		if votes > maxVotes {
			maxVotes = votes
			detectedPattern = pattern
		}
	}

	return detectedPattern
}

// analyzeAuthorFolder examines a folder to determine its pattern
func (d *Detector) analyzeAuthorFolder(authorPath string) Pattern {
	entries, err := os.ReadDir(authorPath)
	if err != nil {
		return ""
	}

	// Check if there are subdirectories
	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			subdirs = append(subdirs, entry.Name())
		}
	}

	if len(subdirs) == 0 {
		// No subdirectories - could be title-only pattern
		if d.hasAudioFiles(authorPath) {
			return PatternTitle
		}
		return ""
	}

	// Check first subdirectory for pattern
	firstSubdir := filepath.Join(authorPath, subdirs[0])
	entries, err = os.ReadDir(firstSubdir)
	if err != nil {
		return PatternAuthorTitle
	}

	// Check if this is a series folder (contains title subfolders) or a title folder (contains audio)
	hasSubdirs := false
	hasAudio := false

	for _, entry := range entries {
		if entry.IsDir() {
			hasSubdirs = true
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if audioExtensions[ext] {
			hasAudio = true
		}
	}

	if hasSubdirs && !hasAudio {
		// Series folder pattern: author/series/title
		return PatternAuthorSeriesTitle
	} else if hasAudio {
		// Title folder pattern: author/title
		return PatternAuthorTitle
	}

	return PatternAuthorTitle
}

// hasAudioFiles checks if a directory contains audio files
func (d *Detector) hasAudioFiles(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if audioExtensions[ext] {
			return true
		}
	}
	return false
}

// BuildPath constructs the destination path based on the pattern and metadata
func (d *Detector) BuildPath(pattern Pattern, metadata *models.BookMetadata) string {
	if metadata == nil {
		return ""
	}

	// Sanitize strings for filesystem paths
	author := sanitizePath(metadata.PrimaryAuthor())
	title := sanitizePath(metadata.Title)
	series := sanitizePath(metadata.Series)

	switch pattern {
	case PatternAuthorSeriesTitle:
		if series != "" {
			return filepath.Join(author, series, title)
		}
		// Fallback to author/title if no series
		return filepath.Join(author, title)

	case PatternSeriesTitle:
		if series != "" {
			return filepath.Join(series, title)
		}
		// Fallback to title if no series
		return title

	case PatternTitle:
		return title

	case PatternAuthorTitle:
		fallthrough
	default:
		return filepath.Join(author, title)
	}
}

// sanitizePath removes or replaces characters that are invalid in filesystem paths
func sanitizePath(s string) string {
	if s == "" {
		return "Unknown"
	}

	// Replace problematic characters
	replacements := map[rune]string{
		'/':  "-",
		'\\': "-",
		':':  "-",
		'*':  "",
		'?':  "",
		'"':  "",
		'<':  "",
		'>':  "",
		'|':  "",
	}

	var result strings.Builder
	for _, r := range s {
		if replacement, ok := replacements[r]; ok {
			result.WriteString(replacement)
		} else {
			result.WriteRune(r)
		}
	}

	// Trim spaces and dots from the ends
	cleaned := strings.TrimSpace(result.String())
	cleaned = strings.Trim(cleaned, ".")

	if cleaned == "" {
		return "Unknown"
	}

	return cleaned
}
