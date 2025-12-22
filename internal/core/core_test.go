package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"audiosort/pkg/models"
)

// === Scanner Tests ===

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name            string
		workers         int
		expectedWorkers int
	}{
		{
			name:            "positive workers",
			workers:         8,
			expectedWorkers: 8,
		},
		{
			name:            "zero workers defaults to 4",
			workers:         0,
			expectedWorkers: 4,
		},
		{
			name:            "negative workers defaults to 4",
			workers:         -1,
			expectedWorkers: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.workers)
			if scanner == nil {
				t.Fatal("NewScanner() returned nil")
			}
			if scanner.workers != tt.expectedWorkers {
				t.Errorf("workers = %d, want %d", scanner.workers, tt.expectedWorkers)
			}
		})
	}
}

func TestScanner_Scan_InvalidPath(t *testing.T) {
	scanner := NewScanner(4)
	_, err := scanner.Scan(context.Background(), "/nonexistent/path")
	if err == nil {
		t.Error("Scan() should return error for nonexistent path")
	}
}

func TestScanner_Scan_NotADirectory(t *testing.T) {
	// Create a temp file
	tempFile, err := os.CreateTemp("", "test-file")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	scanner := NewScanner(4)
	_, err = scanner.Scan(context.Background(), tempFile.Name())
	if err == nil {
		t.Error("Scan() should return error for file path")
	}
}

func TestScanner_Scan_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	scanner := NewScanner(4)
	results, err := scanner.Scan(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	count := 0
	for range results {
		count++
	}

	if count != 0 {
		t.Errorf("Scan() found %d audiobooks, want 0 in empty directory", count)
	}
}

func TestScanner_Scan_FindsAudiobooks(t *testing.T) {
	tempDir := t.TempDir()

	// Create audiobook folder structure
	bookDir := filepath.Join(tempDir, "TestBook")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Create audio files
	audioFiles := []string{"chapter1.mp3", "chapter2.mp3"}
	for _, f := range audioFiles {
		filePath := filepath.Join(bookDir, f)
		if err := os.WriteFile(filePath, []byte("fake audio data"), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	scanner := NewScanner(4)
	results, err := scanner.Scan(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	var audiobooks []models.Audiobook
	for book := range results {
		audiobooks = append(audiobooks, book)
	}

	if len(audiobooks) != 1 {
		t.Fatalf("Scan() found %d audiobooks, want 1", len(audiobooks))
	}

	if audiobooks[0].Path != bookDir {
		t.Errorf("Audiobook.Path = %q, want %q", audiobooks[0].Path, bookDir)
	}

	if len(audiobooks[0].Files) != 2 {
		t.Errorf("Audiobook.Files count = %d, want 2", len(audiobooks[0].Files))
	}
}

func TestScanner_Scan_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	scanner := NewScanner(4)
	results, err := scanner.Scan(ctx, tempDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// Should complete quickly without error
	for range results {
		// Drain channel
	}
}

func TestScanner_Scan_MultipleFormats(t *testing.T) {
	tempDir := t.TempDir()

	bookDir := filepath.Join(tempDir, "MultiFormat")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Create files with different audio extensions
	formats := []string{"test.mp3", "test.m4a", "test.m4b", "test.flac", "test.ogg", "test.wma"}
	for _, f := range formats {
		filePath := filepath.Join(bookDir, f)
		if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	scanner := NewScanner(4)
	results, err := scanner.Scan(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	var audiobooks []models.Audiobook
	for book := range results {
		audiobooks = append(audiobooks, book)
	}

	if len(audiobooks) != 1 {
		t.Fatalf("Scan() found %d audiobooks, want 1", len(audiobooks))
	}

	if len(audiobooks[0].Files) != len(formats) {
		t.Errorf("Found %d audio files, want %d", len(audiobooks[0].Files), len(formats))
	}
}

// === Detector Tests ===

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

func TestDetector_Detect_NonexistentPath(t *testing.T) {
	detector := NewDetector()
	pattern := detector.Detect("/nonexistent/path")
	if pattern != PatternAuthorTitle {
		t.Errorf("Detect() = %q, want %q for nonexistent path", pattern, PatternAuthorTitle)
	}
}

func TestDetector_Detect_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	detector := NewDetector()
	pattern := detector.Detect(tempDir)
	if pattern != PatternAuthorTitle {
		t.Errorf("Detect() = %q, want %q for empty directory", pattern, PatternAuthorTitle)
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "Unknown",
		},
		{
			name:     "normal string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "forward slash",
			input:    "Hello/World",
			expected: "Hello-World",
		},
		{
			name:     "backslash",
			input:    "Hello\\World",
			expected: "Hello-World",
		},
		{
			name:     "colon",
			input:    "Hello: World",
			expected: "Hello- World",
		},
		{
			name:     "asterisk",
			input:    "Hello*World",
			expected: "HelloWorld",
		},
		{
			name:     "question mark",
			input:    "Hello?",
			expected: "Hello",
		},
		{
			name:     "quotes",
			input:    "\"Hello\"",
			expected: "Hello",
		},
		{
			name:     "angle brackets",
			input:    "<Hello>",
			expected: "Hello",
		},
		{
			name:     "pipe",
			input:    "Hello|World",
			expected: "HelloWorld",
		},
		{
			name:     "trailing dots",
			input:    "Hello...",
			expected: "Hello",
		},
		{
			name:     "leading spaces",
			input:    "  Hello",
			expected: "Hello",
		},
		{
			name:     "only special chars",
			input:    "***???",
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePath(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDetector_BuildPath(t *testing.T) {
	tests := []struct {
		name     string
		pattern  Pattern
		metadata *models.BookMetadata
		expected string
	}{
		{
			name:    "author/title pattern",
			pattern: PatternAuthorTitle,
			metadata: &models.BookMetadata{
				Title:   "Test Book",
				Authors: []string{"Test Author"},
			},
			expected: filepath.Join("Test Author", "Test Book"),
		},
		{
			name:    "author/series/title pattern with series",
			pattern: PatternAuthorSeriesTitle,
			metadata: &models.BookMetadata{
				Title:   "Test Book",
				Authors: []string{"Test Author"},
				Series:  "Test Series",
			},
			expected: filepath.Join("Test Author", "Test Series", "Test Book"),
		},
		{
			name:    "author/series/title pattern without series",
			pattern: PatternAuthorSeriesTitle,
			metadata: &models.BookMetadata{
				Title:   "Test Book",
				Authors: []string{"Test Author"},
			},
			expected: filepath.Join("Test Author", "Unknown", "Test Book"),
		},
		{
			name:    "title pattern",
			pattern: PatternTitle,
			metadata: &models.BookMetadata{
				Title:   "Test Book",
				Authors: []string{"Test Author"},
			},
			expected: "Test Book",
		},
		{
			name:    "series/title pattern with series",
			pattern: PatternSeriesTitle,
			metadata: &models.BookMetadata{
				Title:  "Test Book",
				Series: "Test Series",
			},
			expected: filepath.Join("Test Series", "Test Book"),
		},
		{
			name:    "series/title pattern without series",
			pattern: PatternSeriesTitle,
			metadata: &models.BookMetadata{
				Title: "Test Book",
			},
			expected: filepath.Join("Unknown", "Test Book"),
		},
		{
			name:     "nil metadata",
			pattern:  PatternAuthorTitle,
			metadata: nil,
			expected: "",
		},
		{
			name:    "unknown author",
			pattern: PatternAuthorTitle,
			metadata: &models.BookMetadata{
				Title:   "Test Book",
				Authors: []string{},
			},
			expected: filepath.Join(models.UnknownAuthor, "Test Book"),
		},
	}

	detector := NewDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.BuildPath(tt.pattern, tt.metadata)
			if got != tt.expected {
				t.Errorf("BuildPath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPattern_Constants(t *testing.T) {
	tests := []struct {
		pattern  Pattern
		expected string
	}{
		{PatternAuthorTitle, "author/title"},
		{PatternAuthorSeriesTitle, "author/series/title"},
		{PatternTitle, "title"},
		{PatternSeriesTitle, "series/title"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.pattern) != tt.expected {
				t.Errorf("Pattern = %q, want %q", tt.pattern, tt.expected)
			}
		})
	}
}

// === Organizer Tests ===

func TestNewOrganizer(t *testing.T) {
	organizer := NewOrganizer("/dest", true, false, PatternAuthorTitle)
	if organizer == nil {
		t.Fatal("NewOrganizer() returned nil")
	}
	if organizer.destPath != "/dest" {
		t.Errorf("destPath = %q, want %q", organizer.destPath, "/dest")
	}
	if organizer.copyMode != true {
		t.Errorf("copyMode = %v, want true", organizer.copyMode)
	}
	if organizer.skipExist != false {
		t.Errorf("skipExist = %v, want false", organizer.skipExist)
	}
	if organizer.pattern != PatternAuthorTitle {
		t.Errorf("pattern = %q, want %q", organizer.pattern, PatternAuthorTitle)
	}
}

func TestOrganizer_Organize_NilBook(t *testing.T) {
	organizer := NewOrganizer("/dest", false, false, PatternAuthorTitle)
	result := organizer.Organize(context.Background(), nil)

	if result.Error == nil {
		t.Error("Organize() should return error for nil book")
	}
}

func TestOrganizer_Organize_NoMetadata(t *testing.T) {
	tempDir := t.TempDir()
	organizer := NewOrganizer(tempDir, false, false, PatternAuthorTitle)

	book := &models.Audiobook{
		Path:     "/test/path",
		Metadata: nil,
	}

	result := organizer.Organize(context.Background(), book)
	if result.Error == nil {
		t.Error("Organize() should return error for book without metadata")
	}
	if book.Status != models.StatusError {
		t.Errorf("book.Status = %q, want %q", book.Status, models.StatusError)
	}
}

func TestOrganizer_Organize_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	organizer := NewOrganizer(tempDir, false, false, PatternAuthorTitle)
	book := &models.Audiobook{
		Path: "/test/path",
		Metadata: &models.BookMetadata{
			Title:   "Test",
			Authors: []string{"Author"},
		},
	}

	result := organizer.Organize(ctx, book)
	if result.Error == nil {
		t.Error("Organize() should return error for cancelled context")
	}
}

func TestOrganizer_shouldCopyAdditionalFile(t *testing.T) {
	organizer := NewOrganizer("/dest", false, false, PatternAuthorTitle)

	tests := []struct {
		filename string
		expected bool
	}{
		{"cover.jpg", true},
		{"cover.jpeg", true},
		{"cover.png", true},
		{"cover.gif", true},
		{"cover.webp", true},
		{"metadata.opf", true},
		{"info.txt", true},
		{"info.nfo", true},
		{"metadata.json", true},
		{"metadata.xml", true},
		{"playlist.cue", true},
		{"audio.mp3", false},
		{"audio.m4b", false},
		{"unknown.xyz", false},
		{"document.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := organizer.shouldCopyAdditionalFile(tt.filename)
			if got != tt.expected {
				t.Errorf("shouldCopyAdditionalFile(%q) = %v, want %v", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestOrganizer_Organize_CopyMode(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create source audiobook
	bookDir := filepath.Join(srcDir, "TestBook")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	audioFile := filepath.Join(bookDir, "chapter1.mp3")
	if err := os.WriteFile(audioFile, []byte("audio content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	organizer := NewOrganizer(destDir, true, false, PatternAuthorTitle)
	book := &models.Audiobook{
		Path: bookDir,
		Files: []models.AudioFile{
			{Path: audioFile, Size: 13},
		},
		Metadata: &models.BookMetadata{
			Title:   "Test Book",
			Authors: []string{"Test Author"},
		},
	}

	result := organizer.Organize(context.Background(), book)
	if result.Error != nil {
		t.Fatalf("Organize() error = %v", result.Error)
	}

	// Verify destination exists
	expectedDest := filepath.Join(destDir, "Test Author", "Test Book", "chapter1.mp3")
	if _, err := os.Stat(expectedDest); err != nil {
		t.Errorf("Destination file should exist: %v", err)
	}

	// Verify source still exists (copy mode)
	if _, err := os.Stat(audioFile); err != nil {
		t.Errorf("Source file should still exist in copy mode: %v", err)
	}

	if book.Status != models.StatusDone {
		t.Errorf("book.Status = %q, want %q", book.Status, models.StatusDone)
	}
}

func TestOrganizer_Organize_SkipExisting(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create source
	bookDir := filepath.Join(srcDir, "TestBook")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	audioFile := filepath.Join(bookDir, "chapter1.mp3")
	if err := os.WriteFile(audioFile, []byte("audio"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Pre-create destination
	existingDest := filepath.Join(destDir, "Test Author", "Test Book")
	if err := os.MkdirAll(existingDest, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	organizer := NewOrganizer(destDir, true, true, PatternAuthorTitle)
	book := &models.Audiobook{
		Path: bookDir,
		Files: []models.AudioFile{
			{Path: audioFile},
		},
		Metadata: &models.BookMetadata{
			Title:   "Test Book",
			Authors: []string{"Test Author"},
		},
	}

	result := organizer.Organize(context.Background(), book)
	if result.Error != nil {
		t.Fatalf("Organize() error = %v", result.Error)
	}

	if book.Status != models.StatusSkipped {
		t.Errorf("book.Status = %q, want %q", book.Status, models.StatusSkipped)
	}
}

func TestOrganizer_copyFile(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "source.txt")
	content := "test content"
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	destFile := filepath.Join(destDir, "dest.txt")
	organizer := NewOrganizer("/", false, false, PatternAuthorTitle)

	if err := organizer.copyFile(srcFile, destFile); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify content
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != content {
		t.Errorf("File content = %q, want %q", string(data), content)
	}

	// Verify source still exists
	if _, err := os.Stat(srcFile); err != nil {
		t.Error("Source file should still exist after copy")
	}
}

func TestOrganizer_Result_Duration(t *testing.T) {
	organizer := NewOrganizer("/dest", false, false, PatternAuthorTitle)
	book := &models.Audiobook{
		Path: "",
	}

	result := organizer.Organize(context.Background(), book)

	if result.Duration <= 0 {
		t.Errorf("Result.Duration = %v, should be positive", result.Duration)
	}
	if result.Duration > time.Second {
		t.Errorf("Result.Duration = %v, should be less than 1 second for invalid input", result.Duration)
	}
}

func TestAudioExtensions(t *testing.T) {
	expected := map[string]bool{
		".mp3":  true,
		".m4a":  true,
		".m4b":  true,
		".flac": true,
		".ogg":  true,
		".wma":  true,
	}

	for ext, val := range expected {
		if audioExtensions[ext] != val {
			t.Errorf("audioExtensions[%q] = %v, want %v", ext, audioExtensions[ext], val)
		}
	}

	// Verify non-audio extensions are not present
	nonAudio := []string{".txt", ".jpg", ".pdf", ".doc"}
	for _, ext := range nonAudio {
		if audioExtensions[ext] {
			t.Errorf("audioExtensions[%q] should be false", ext)
		}
	}
}
