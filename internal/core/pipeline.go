package core

import (
	"context"
	"sync"
	"time"

	"audiosort/internal/metadata"
	"audiosort/pkg/models"
)

// Writer is an interface for components that write output files (metadata, covers, etc.)
type Writer interface {
	Write(ctx context.Context, book *models.Audiobook, destPath string) error
}

// PipelineOptions configures the pipeline execution
type PipelineOptions struct {
	SourcePath  string
	DestPath    string
	Pattern     Pattern
	Workers     int
	CopyMode    bool
	SkipExist   bool
	DryRun      bool
	Fetcher     *metadata.Fetcher
	Writers     []Writer
}

// Pipeline orchestrates the entire audiobook processing workflow
type Pipeline struct {
	scanner   *Scanner
	fetcher   *metadata.Fetcher
	organizer *Organizer
	detector  *Detector
	writers   []Writer
	workers   int
	dryRun    bool
}

// NewPipeline creates a new processing pipeline with the given options
func NewPipeline(opts PipelineOptions) *Pipeline {
	workers := opts.Workers
	if workers <= 0 {
		workers = 4
	}

	scanner := NewScanner(workers)
	detector := NewDetector()

	// Detect pattern if not explicitly set
	pattern := opts.Pattern
	if pattern == "" {
		pattern = detector.Detect(opts.DestPath)
	}

	var organizer *Organizer
	if !opts.DryRun {
		organizer = NewOrganizer(opts.DestPath, opts.CopyMode, opts.SkipExist, pattern)
	}

	return &Pipeline{
		scanner:   scanner,
		fetcher:   opts.Fetcher,
		organizer: organizer,
		detector:  detector,
		writers:   opts.Writers,
		workers:   workers,
		dryRun:    opts.DryRun,
	}
}

// Run executes the full pipeline: scan -> fetch metadata -> organize -> write outputs
func (p *Pipeline) Run(ctx context.Context, sourcePath string) (*models.Summary, error) {
	start := time.Now()

	// Phase 1: Scan for audiobooks
	audiobookChan, err := p.scanner.Scan(ctx, sourcePath)
	if err != nil {
		return nil, err
	}

	// Phase 2 & 3: Fetch metadata and organize (in parallel using worker pool)
	results := p.processAudiobooks(ctx, audiobookChan)

	// Phase 4: Collect and summarize results
	summary := p.collectResults(results)
	summary.Duration = time.Since(start)

	return summary, nil
}

// processAudiobooks processes audiobooks through the metadata fetching and organization stages
func (p *Pipeline) processAudiobooks(ctx context.Context, audiobookChan <-chan models.Audiobook) <-chan models.Result {
	results := make(chan models.Result)

	// Worker pool for parallel processing
	var wg sync.WaitGroup
	workerChan := make(chan models.Audiobook, p.workers)

	// Start workers
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker(ctx, workerChan, results)
		}()
	}

	// Feed workers
	go func() {
		for audiobook := range audiobookChan {
			select {
			case workerChan <- audiobook:
			case <-ctx.Done():
				close(workerChan)
				return
			}
		}
		close(workerChan)
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// worker processes individual audiobooks: fetch metadata -> organize -> write outputs
func (p *Pipeline) worker(ctx context.Context, books <-chan models.Audiobook, results chan<- models.Result) {
	for book := range books {
		result := p.processOne(ctx, &book)

		select {
		case results <- result:
		case <-ctx.Done():
			return
		}
	}
}

// processOne processes a single audiobook through all stages
func (p *Pipeline) processOne(ctx context.Context, book *models.Audiobook) models.Result {
	start := time.Now()

	// Check context
	select {
	case <-ctx.Done():
		return models.Result{
			Audiobook: book,
			Error:     ctx.Err(),
			Duration:  time.Since(start),
		}
	default:
	}

	// Stage 1: Fetch metadata
	if p.fetcher != nil {
		query := inferQuery(book)
		metadata, err := p.fetcher.Fetch(ctx, query)
		if err != nil {
			book.Status = models.StatusError
			book.Error = "metadata fetch failed"
			return models.Result{
				Audiobook: book,
				Error:     err,
				Duration:  time.Since(start),
			}
		}
		book.Metadata = metadata
	}

	// Stage 2: Organize (move/copy files)
	if !p.dryRun && p.organizer != nil {
		result := p.organizer.Organize(ctx, book)
		if result.Error != nil {
			return result
		}

		// Stage 3: Write additional outputs (metadata files, covers, etc.)
		if len(p.writers) > 0 {
			destPath := p.detector.BuildPath(p.organizer.pattern, book.Metadata)
			for _, writer := range p.writers {
				if err := writer.Write(ctx, book, destPath); err != nil {
					// Continue processing even if a writer fails
					continue
				}
			}
		}

		return result
	}

	// Dry run mode
	book.Status = models.StatusDone
	return models.Result{
		Audiobook: book,
		Error:     nil,
		Duration:  time.Since(start),
	}
}

// collectResults aggregates processing results into a summary
func (p *Pipeline) collectResults(results <-chan models.Result) *models.Summary {
	summary := &models.Summary{}

	for result := range results {
		summary.Total++

		if result.Error != nil {
			summary.Errors++
			summary.Failures = append(summary.Failures, result)
		} else if result.Audiobook != nil && result.Audiobook.Status == models.StatusSkipped {
			summary.Skipped++
		} else {
			summary.Processed++
		}
	}

	return summary
}

// inferQuery generates a search query from an audiobook's path and files
func inferQuery(book *models.Audiobook) string {
	if book == nil || book.Path == "" {
		return ""
	}

	// TODO: Could extract album/artist from ID3 tags in book.Files
	// For now, use the directory name as the query
	return book.Path
}
