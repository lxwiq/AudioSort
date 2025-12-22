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
	Error     error
	Duration  time.Duration
}

func (r Result) Success() bool {
	return r.Error == nil
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
	return fmt.Sprintf("[OK] %d processed | [-] %d skipped | [ERR] %d errors",
		s.Processed, s.Skipped, s.Errors)
}
