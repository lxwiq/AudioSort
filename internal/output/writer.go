package output

import (
	"context"
	"net/http"

	"audiosort/pkg/models"
)

// Format represents an output format preset
type Format string

const (
	FormatAudiobookShelf Format = "audiobookshelf"
	FormatPlex           Format = "plex"
	FormatJSON           Format = "json"
	FormatAll            Format = "all"
)

// Writer interface matches core.Writer
type Writer interface {
	Write(ctx context.Context, book *models.Audiobook, destPath string) error
}

// NewWriters creates writers based on format preset
func NewWriters(format Format, basePath string) []Writer {
	client := &http.Client{}

	switch format {
	case FormatAudiobookShelf:
		return []Writer{
			NewOPFWriter(basePath),
			NewCoverWriter(basePath, client),
		}
	case FormatPlex:
		return []Writer{
			NewOPFWriter(basePath),
			NewCoverWriter(basePath, client),
		}
	case FormatJSON:
		return []Writer{
			NewJSONWriter(basePath),
		}
	case FormatAll:
		return []Writer{
			NewOPFWriter(basePath),
			NewJSONWriter(basePath),
			NewCoverWriter(basePath, client),
		}
	default:
		return []Writer{}
	}
}
