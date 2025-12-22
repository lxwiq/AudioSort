package tui

import "audiosort/pkg/models"

// scanCompleteMsg is sent when the initial scan is complete
type scanCompleteMsg struct {
	books []models.Audiobook
	err   error
}

// metadataFetchedMsg is sent when metadata has been fetched for a book
type metadataFetchedMsg struct {
	index    int
	metadata *models.BookMetadata
	err      error
}

// processStartMsg is sent when processing begins
type processStartMsg struct{}

// processProgressMsg is sent to update processing progress
type processProgressMsg struct {
	current int
	total   int
}

// processCompleteMsg is sent when all processing is complete
type processCompleteMsg struct {
	summary *models.Summary
	err     error
}

// bookProcessedMsg is sent when a single book has been processed
type bookProcessedMsg struct {
	index  int
	result models.Result
}

// errorMsg wraps an error
type errorMsg struct {
	err error
}

// tickMsg is sent periodically for spinner animation
type tickMsg struct{}
