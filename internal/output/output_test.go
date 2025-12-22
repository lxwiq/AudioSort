package output

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"audiosort/pkg/models"
)

// === Format Constants Tests ===

func TestFormat_Constants(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
	}{
		{FormatAudiobookShelf, "audiobookshelf"},
		{FormatPlex, "plex"},
		{FormatJSON, "json"},
		{FormatAll, "all"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.format) != tt.expected {
				t.Errorf("Format = %q, want %q", tt.format, tt.expected)
			}
		})
	}
}

func TestNewWriters(t *testing.T) {
	tests := []struct {
		name          string
		format        Format
		expectedCount int
	}{
		{
			name:          "audiobookshelf format",
			format:        FormatAudiobookShelf,
			expectedCount: 2, // OPF + Cover
		},
		{
			name:          "plex format",
			format:        FormatPlex,
			expectedCount: 2, // OPF + Cover
		},
		{
			name:          "json format",
			format:        FormatJSON,
			expectedCount: 1, // JSON only
		},
		{
			name:          "all format",
			format:        FormatAll,
			expectedCount: 3, // OPF + JSON + Cover
		},
		{
			name:          "unknown format",
			format:        "unknown",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writers := NewWriters(tt.format, "/base")
			if len(writers) != tt.expectedCount {
				t.Errorf("NewWriters() returned %d writers, want %d", len(writers), tt.expectedCount)
			}
		})
	}
}

// === OPF Writer Tests ===

func TestNewOPFWriter(t *testing.T) {
	writer := NewOPFWriter("/base/path")
	if writer == nil {
		t.Fatal("NewOPFWriter() returned nil")
	}
	if writer.basePath != "/base/path" {
		t.Errorf("basePath = %q, want %q", writer.basePath, "/base/path")
	}
}

func TestOPFWriter_Write_NilBook(t *testing.T) {
	writer := NewOPFWriter("/tmp")
	err := writer.Write(context.Background(), nil, "dest")
	if err == nil {
		t.Error("Write() should return error for nil book")
	}
}

func TestOPFWriter_Write_NilMetadata(t *testing.T) {
	writer := NewOPFWriter("/tmp")
	book := &models.Audiobook{Path: "/test", Metadata: nil}
	err := writer.Write(context.Background(), book, "dest")
	if err == nil {
		t.Error("Write() should return error for nil metadata")
	}
}

func TestOPFWriter_Write_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	writer := NewOPFWriter("/tmp")
	book := &models.Audiobook{
		Path:     "/test",
		Metadata: &models.BookMetadata{Title: "Test"},
	}

	err := writer.Write(ctx, book, "dest")
	if err == nil {
		t.Error("Write() should return error for cancelled context")
	}
}

func TestOPFWriter_Write_CreatesFile(t *testing.T) {
	tempDir := t.TempDir()

	writer := NewOPFWriter(tempDir)
	book := &models.Audiobook{
		Path: "/test/path",
		Metadata: &models.BookMetadata{
			Title:          "Test Book",
			Authors:        []string{"Test Author", "Second Author"},
			Publisher:      "Test Publisher",
			PublishYear:    "2023",
			Language:       "en",
			Genres:         []string{"Fiction", "Adventure"},
			Series:         "Test Series",
			SeriesPosition: "1",
			Narrators:      []string{"Test Narrator"},
		},
	}

	err := writer.Write(context.Background(), book, "TestBook")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file exists
	opfPath := filepath.Join(tempDir, "TestBook", "metadata.opf")
	if _, err := os.Stat(opfPath); err != nil {
		t.Fatalf("OPF file should exist: %v", err)
	}

	// Read and verify content
	data, err := os.ReadFile(opfPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)

	// Verify XML declaration
	if !strings.HasPrefix(content, "<?xml") {
		t.Error("OPF should start with XML declaration")
	}

	// Verify key content
	expectations := []string{
		"Test Book",
		"Test Author, Second Author",
		"Test Publisher",
		"2023",
		"en",
		"calibre:series",
		"Test Series",
		"calibre:series_index",
		"calibre:narrators",
		"Test Narrator",
	}

	for _, expected := range expectations {
		if !strings.Contains(content, expected) {
			t.Errorf("OPF should contain %q", expected)
		}
	}
}

func TestOPFWriter_Write_MinimalMetadata(t *testing.T) {
	tempDir := t.TempDir()

	writer := NewOPFWriter(tempDir)
	book := &models.Audiobook{
		Path: "/test",
		Metadata: &models.BookMetadata{
			Title: "Minimal Book",
		},
	}

	err := writer.Write(context.Background(), book, "MinimalBook")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file exists and is valid XML
	opfPath := filepath.Join(tempDir, "MinimalBook", "metadata.opf")
	data, err := os.ReadFile(opfPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)

	// Verify XML starts with declaration
	if !strings.HasPrefix(content, "<?xml") {
		t.Error("OPF should start with XML declaration")
	}

	// Verify title is present in content
	if !strings.Contains(content, "Minimal Book") {
		t.Error("OPF should contain the title 'Minimal Book'")
	}

	// Verify it has the package structure
	if !strings.Contains(content, "<package") {
		t.Error("OPF should contain package element")
	}
}

// === JSON Writer Tests ===

func TestNewJSONWriter(t *testing.T) {
	writer := NewJSONWriter("/base/path")
	if writer == nil {
		t.Fatal("NewJSONWriter() returned nil")
	}
	if writer.basePath != "/base/path" {
		t.Errorf("basePath = %q, want %q", writer.basePath, "/base/path")
	}
}

func TestJSONWriter_Write_NilBook(t *testing.T) {
	writer := NewJSONWriter("/tmp")
	err := writer.Write(context.Background(), nil, "dest")
	if err == nil {
		t.Error("Write() should return error for nil book")
	}
}

func TestJSONWriter_Write_NilMetadata(t *testing.T) {
	writer := NewJSONWriter("/tmp")
	book := &models.Audiobook{Path: "/test", Metadata: nil}
	err := writer.Write(context.Background(), book, "dest")
	if err == nil {
		t.Error("Write() should return error for nil metadata")
	}
}

func TestJSONWriter_Write_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	writer := NewJSONWriter("/tmp")
	book := &models.Audiobook{
		Path:     "/test",
		Metadata: &models.BookMetadata{Title: "Test"},
	}

	err := writer.Write(ctx, book, "dest")
	if err == nil {
		t.Error("Write() should return error for cancelled context")
	}
}

func TestJSONWriter_Write_CreatesFile(t *testing.T) {
	tempDir := t.TempDir()

	writer := NewJSONWriter(tempDir)
	book := &models.Audiobook{
		Path: "/test/path",
		Metadata: &models.BookMetadata{
			Title:       "Test Book",
			Authors:     []string{"Test Author"},
			ISBN:        "1234567890",
			PublishYear: "2023",
		},
	}

	err := writer.Write(context.Background(), book, "TestBook")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file exists
	jsonPath := filepath.Join(tempDir, "TestBook", "metadata.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("JSON file should exist: %v", err)
	}

	// Read and verify content
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Verify JSON is valid
	var metadata models.BookMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Errorf("JSON should be valid: %v", err)
	}

	if metadata.Title != "Test Book" {
		t.Errorf("Title = %q, want %q", metadata.Title, "Test Book")
	}
	if metadata.ISBN != "1234567890" {
		t.Errorf("ISBN = %q, want %q", metadata.ISBN, "1234567890")
	}
}

func TestJSONWriter_Write_FormattedOutput(t *testing.T) {
	tempDir := t.TempDir()

	writer := NewJSONWriter(tempDir)
	book := &models.Audiobook{
		Path: "/test",
		Metadata: &models.BookMetadata{
			Title: "Test",
		},
	}

	err := writer.Write(context.Background(), book, "Formatted")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	jsonPath := filepath.Join(tempDir, "Formatted", "metadata.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Verify JSON is indented (has newlines and spaces)
	content := string(data)
	if !strings.Contains(content, "\n") {
		t.Error("JSON output should be formatted with newlines")
	}
}

// === OPF XML Structure Tests ===

func TestOPFPackage_XMLMarshaling(t *testing.T) {
	opf := OPFPackage{
		Version: "2.0",
		Xmlns:   "http://www.idpf.org/2007/opf",
		Metadata: OPFMetadata{
			XmlnsDC:   "http://purl.org/dc/elements/1.1/",
			Title:     "Test Title",
			Creator:   "Test Author",
			Language:  "en",
			Publisher: "Test Publisher",
			Date:      "2023",
			Subject:   []string{"Genre1", "Genre2"},
			Meta: []OPFMeta{
				{Name: "calibre:series", Content: "Test Series"},
			},
		},
	}

	output, err := xml.MarshalIndent(opf, "", "  ")
	if err != nil {
		t.Fatalf("xml.MarshalIndent() error = %v", err)
	}

	content := string(output)

	expectations := []string{
		`version="2.0"`,
		`xmlns="http://www.idpf.org/2007/opf"`,
		"Test Title",
		"Test Author",
		"dc:language",
		"dc:publisher",
		"calibre:series",
	}

	for _, expected := range expectations {
		if !strings.Contains(content, expected) {
			t.Errorf("XML should contain %q", expected)
		}
	}
}

// === Writer Interface Tests ===

func TestWriter_Interface(t *testing.T) {
	// Verify OPFWriter implements Writer
	var _ Writer = (*OPFWriter)(nil)

	// Verify JSONWriter implements Writer
	var _ Writer = (*JSONWriter)(nil)
}

// === Integration Tests ===

func TestWriters_CreateAllFormats(t *testing.T) {
	tempDir := t.TempDir()

	book := &models.Audiobook{
		Path: "/test",
		Metadata: &models.BookMetadata{
			Title:   "Integration Test Book",
			Authors: []string{"Integration Author"},
		},
	}

	// Test OPF Writer
	opfWriter := NewOPFWriter(tempDir)
	if err := opfWriter.Write(context.Background(), book, "OPFTest"); err != nil {
		t.Errorf("OPFWriter.Write() error = %v", err)
	}

	// Verify OPF file
	if _, err := os.Stat(filepath.Join(tempDir, "OPFTest", "metadata.opf")); err != nil {
		t.Errorf("OPF file should exist: %v", err)
	}

	// Test JSON Writer
	jsonWriter := NewJSONWriter(tempDir)
	if err := jsonWriter.Write(context.Background(), book, "JSONTest"); err != nil {
		t.Errorf("JSONWriter.Write() error = %v", err)
	}

	// Verify JSON file
	if _, err := os.Stat(filepath.Join(tempDir, "JSONTest", "metadata.json")); err != nil {
		t.Errorf("JSON file should exist: %v", err)
	}
}

func TestWriters_SameDestination(t *testing.T) {
	tempDir := t.TempDir()

	book := &models.Audiobook{
		Path: "/test",
		Metadata: &models.BookMetadata{
			Title:   "Multi-Format Book",
			Authors: []string{"Multi Author"},
		},
	}

	// Write both formats to same destination
	opfWriter := NewOPFWriter(tempDir)
	jsonWriter := NewJSONWriter(tempDir)

	destPath := "MultiFormat"

	if err := opfWriter.Write(context.Background(), book, destPath); err != nil {
		t.Fatalf("OPFWriter.Write() error = %v", err)
	}

	if err := jsonWriter.Write(context.Background(), book, destPath); err != nil {
		t.Fatalf("JSONWriter.Write() error = %v", err)
	}

	// Verify both files exist in same directory
	destDir := filepath.Join(tempDir, destPath)
	if _, err := os.Stat(filepath.Join(destDir, "metadata.opf")); err != nil {
		t.Error("metadata.opf should exist")
	}
	if _, err := os.Stat(filepath.Join(destDir, "metadata.json")); err != nil {
		t.Error("metadata.json should exist")
	}
}
