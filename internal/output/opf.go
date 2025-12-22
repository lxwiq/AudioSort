package output

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"audiosort/pkg/models"
)

// OPFWriter writes metadata.opf files
type OPFWriter struct {
	basePath string
}

// NewOPFWriter creates a new OPF metadata writer
func NewOPFWriter(basePath string) *OPFWriter {
	return &OPFWriter{basePath: basePath}
}

// OPF XML structure
type OPFPackage struct {
	XMLName  xml.Name     `xml:"package"`
	Version  string       `xml:"version,attr"`
	Xmlns    string       `xml:"xmlns,attr"`
	Metadata OPFMetadata  `xml:"metadata"`
}

type OPFMetadata struct {
	XMLName   xml.Name  `xml:"metadata"`
	XmlnsDC   string    `xml:"xmlns:dc,attr"`
	Title     string    `xml:"dc:title"`
	Creator   string    `xml:"dc:creator,omitempty"`
	Language  string    `xml:"dc:language,omitempty"`
	Publisher string    `xml:"dc:publisher,omitempty"`
	Date      string    `xml:"dc:date,omitempty"`
	Subject   []string  `xml:"dc:subject,omitempty"`
	Meta      []OPFMeta `xml:"meta,omitempty"`
}

type OPFMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

// Write generates and writes an OPF metadata file
func (w *OPFWriter) Write(ctx context.Context, book *models.Audiobook, destPath string) error {
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

	metadata := book.Metadata

	// Build OPF structure
	opf := OPFPackage{
		Version: "2.0",
		Xmlns:   "http://www.idpf.org/2007/opf",
		Metadata: OPFMetadata{
			XmlnsDC:  "http://purl.org/dc/elements/1.1/",
			Title:    metadata.Title,
			Language: metadata.Language,
		},
	}

	// Add primary author
	if len(metadata.Authors) > 0 {
		opf.Metadata.Creator = strings.Join(metadata.Authors, ", ")
	}

	// Add publisher
	if metadata.Publisher != "" {
		opf.Metadata.Publisher = metadata.Publisher
	}

	// Add publish year
	if metadata.PublishYear != "" {
		opf.Metadata.Date = metadata.PublishYear
	}

	// Add genres as subjects
	if len(metadata.Genres) > 0 {
		opf.Metadata.Subject = metadata.Genres
	}

	// Add series metadata if available
	var metaTags []OPFMeta
	if metadata.Series != "" {
		metaTags = append(metaTags, OPFMeta{
			Name:    "calibre:series",
			Content: metadata.Series,
		})
	}
	if metadata.SeriesPosition != "" {
		metaTags = append(metaTags, OPFMeta{
			Name:    "calibre:series_index",
			Content: metadata.SeriesPosition,
		})
	}

	// Add narrators if available
	if len(metadata.Narrators) > 0 {
		metaTags = append(metaTags, OPFMeta{
			Name:    "calibre:narrators",
			Content: strings.Join(metadata.Narrators, ", "),
		})
	}

	opf.Metadata.Meta = metaTags

	// Marshal to XML
	output, err := xml.MarshalIndent(opf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OPF: %w", err)
	}

	// Add XML declaration
	xmlContent := []byte(xml.Header + string(output))

	// Create destination directory if needed
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	opfPath := filepath.Join(destPath, "metadata.opf")
	if err := os.WriteFile(opfPath, xmlContent, 0644); err != nil {
		return fmt.Errorf("failed to write OPF file: %w", err)
	}

	return nil
}
