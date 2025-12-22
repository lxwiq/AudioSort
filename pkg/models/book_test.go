package models

import (
	"testing"
	"time"
)

func TestBookMetadata_PrimaryAuthor(t *testing.T) {
	tests := []struct {
		name     string
		authors  []string
		expected string
	}{
		{
			name:     "returns first author when multiple authors",
			authors:  []string{"Author One", "Author Two", "Author Three"},
			expected: "Author One",
		},
		{
			name:     "returns single author",
			authors:  []string{"Solo Author"},
			expected: "Solo Author",
		},
		{
			name:     "returns UnknownAuthor when no authors",
			authors:  []string{},
			expected: UnknownAuthor,
		},
		{
			name:     "returns UnknownAuthor when authors is nil",
			authors:  nil,
			expected: UnknownAuthor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := BookMetadata{Authors: tt.authors}
			got := book.PrimaryAuthor()
			if got != tt.expected {
				t.Errorf("PrimaryAuthor() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBookMetadata_PrimaryNarrator(t *testing.T) {
	tests := []struct {
		name      string
		narrators []string
		expected  string
	}{
		{
			name:      "returns first narrator when multiple narrators",
			narrators: []string{"Narrator One", "Narrator Two"},
			expected:  "Narrator One",
		},
		{
			name:      "returns single narrator",
			narrators: []string{"Solo Narrator"},
			expected:  "Solo Narrator",
		},
		{
			name:      "returns empty string when no narrators",
			narrators: []string{},
			expected:  "",
		},
		{
			name:      "returns empty string when narrators is nil",
			narrators: nil,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := BookMetadata{Narrators: tt.narrators}
			got := book.PrimaryNarrator()
			if got != tt.expected {
				t.Errorf("PrimaryNarrator() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStatus_Constants(t *testing.T) {
	// Verify status constants have expected values
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPending, "pending"},
		{StatusProcessing, "processing"},
		{StatusDone, "done"},
		{StatusError, "error"},
		{StatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status = %q, want %q", tt.status, tt.expected)
			}
		})
	}
}

func TestAudiobook_Fields(t *testing.T) {
	metadata := &BookMetadata{
		Title:   "Test Book",
		Authors: []string{"Test Author"},
	}

	files := []AudioFile{
		{Path: "/path/to/file1.mp3", Size: 1024, Duration: time.Minute * 5},
		{Path: "/path/to/file2.mp3", Size: 2048, Duration: time.Minute * 10},
	}

	audiobook := Audiobook{
		Path:     "/path/to/book",
		Files:    files,
		Metadata: metadata,
		Status:   StatusPending,
		Error:    "",
	}

	if audiobook.Path != "/path/to/book" {
		t.Errorf("Path = %q, want %q", audiobook.Path, "/path/to/book")
	}

	if len(audiobook.Files) != 2 {
		t.Errorf("Files count = %d, want %d", len(audiobook.Files), 2)
	}

	if audiobook.Metadata.Title != "Test Book" {
		t.Errorf("Metadata.Title = %q, want %q", audiobook.Metadata.Title, "Test Book")
	}

	if audiobook.Status != StatusPending {
		t.Errorf("Status = %q, want %q", audiobook.Status, StatusPending)
	}
}

func TestAudioFile_Fields(t *testing.T) {
	file := AudioFile{
		Path:     "/path/to/audio.mp3",
		Size:     1048576, // 1MB
		Duration: time.Hour + time.Minute*30,
	}

	if file.Path != "/path/to/audio.mp3" {
		t.Errorf("Path = %q, want %q", file.Path, "/path/to/audio.mp3")
	}

	if file.Size != 1048576 {
		t.Errorf("Size = %d, want %d", file.Size, 1048576)
	}

	expectedDuration := time.Hour + time.Minute*30
	if file.Duration != expectedDuration {
		t.Errorf("Duration = %v, want %v", file.Duration, expectedDuration)
	}
}

func TestUnknownAuthor_Constant(t *testing.T) {
	if UnknownAuthor != "_unknown_" {
		t.Errorf("UnknownAuthor = %q, want %q", UnknownAuthor, "_unknown_")
	}
}
