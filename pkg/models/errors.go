package models

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNoMetadataFound   = errors.New("no metadata found for query")
	ErrSourceTimeout     = errors.New("metadata source timeout")
	ErrInvalidPath       = errors.New("invalid audiobook path")
	ErrDestinationExists = errors.New("destination already exists")
)

type Result struct {
	Audiobook *Audiobook
	Success   bool
	Error     error
	Duration  time.Duration
}

type Summary struct {
	Total     int
	Processed int
	Skipped   int
	Errors    int
	Duration  time.Duration
	Failures  []Result
}

func (s Summary) String() string {
	return fmt.Sprintf("✓ %d processed | ○ %d skipped | ✗ %d errors",
		s.Processed, s.Skipped, s.Errors)
}
