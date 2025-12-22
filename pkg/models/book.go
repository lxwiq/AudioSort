package models

import "time"

type BookMetadata struct {
	Title          string   `json:"title"`
	Authors        []string `json:"authors"`
	Narrators      []string `json:"narrators"`
	Series         string   `json:"series,omitempty"`
	SeriesPosition string   `json:"series_position,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Publisher      string   `json:"publisher,omitempty"`
	PublishYear    string   `json:"publish_year,omitempty"`
	Language       string   `json:"language,omitempty"`
	Genres         []string `json:"genres,omitempty"`
	ISBN           string   `json:"isbn,omitempty"`
	ASIN           string   `json:"asin,omitempty"`
	CoverURL       string   `json:"cover_url,omitempty"`
	SourceURL      string   `json:"source_url,omitempty"`
	Source         string   `json:"source"`
}

func (b BookMetadata) PrimaryAuthor() string {
	if len(b.Authors) > 0 {
		return b.Authors[0]
	}
	return "_unknown_"
}

type Audiobook struct {
	Path     string
	Files    []AudioFile
	Metadata *BookMetadata
	Status   Status
	Error    error
}

type AudioFile struct {
	Path     string
	Size     int64
	Duration time.Duration
}

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusError      Status = "error"
	StatusSkipped    Status = "skipped"
)
