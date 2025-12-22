package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"audiosort/internal/cache"
	"audiosort/internal/core"
	"audiosort/internal/metadata"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan and organize audiobooks",
	Long: `Scan a directory for audiobooks, fetch metadata from multiple sources,
and organize them into a structured format.`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringP("output", "o", "", "output directory")
	scanCmd.Flags().String("format", "audiobookshelf", "output format (audiobookshelf|author-title|title|series-title)")
	scanCmd.Flags().Bool("copy", false, "copy files instead of moving")
	scanCmd.Flags().Bool("dry-run", false, "preview without making changes")
	scanCmd.Flags().Int("workers", 4, "number of parallel workers")
	scanCmd.Flags().Bool("force", false, "ignore cache, reprocess everything")
	scanCmd.Flags().Bool("skip-existing", true, "skip books that already exist in destination")
}

func runScan(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]

	// Validate source path
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("source path does not exist: %w", err)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get flags
	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath == "" {
		if cfg.DefaultOutput != "" {
			outputPath = cfg.DefaultOutput
		} else {
			outputPath = filepath.Join(sourcePath, "_organized")
		}
	}

	format, _ := cmd.Flags().GetString("format")
	copyMode, _ := cmd.Flags().GetBool("copy")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	workers, _ := cmd.Flags().GetInt("workers")
	force, _ := cmd.Flags().GetBool("force")
	skipExisting, _ := cmd.Flags().GetBool("skip-existing")

	// Override config with flags if provided
	if cmd.Flags().Changed("copy") {
		cfg.CopyMode = copyMode
	}
	if cmd.Flags().Changed("workers") {
		cfg.ParallelWorkers = workers
	} else {
		workers = cfg.ParallelWorkers
	}
	if cmd.Flags().Changed("skip-existing") {
		cfg.SkipExisting = skipExisting
	}

	// Convert format string to pattern
	pattern := core.Pattern(format)
	switch format {
	case "audiobookshelf", "author-title":
		pattern = core.PatternAuthorTitle
	case "author-series-title":
		pattern = core.PatternAuthorSeriesTitle
	case "title":
		pattern = core.PatternTitle
	case "series-title":
		pattern = core.PatternSeriesTitle
	}

	// Initialize cache
	var cacheStore *cache.Store
	if !force {
		cacheDir, err := getCacheDir()
		if err == nil {
			cacheStore, err = cache.NewStore(filepath.Join(cacheDir, "cache.db"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to initialize cache: %v\n", err)
			}
			if cacheStore != nil {
				defer cacheStore.Close()
			}
		}
	}

	// Initialize metadata sources
	sources := initMetadataSources(cfg)

	// Create metadata fetcher
	fetcher := metadata.NewFetcher(sources, cacheStore)

	// Create pipeline
	pipeline := core.NewPipeline(core.PipelineOptions{
		SourcePath: sourcePath,
		DestPath:   outputPath,
		Pattern:    pattern,
		Workers:    workers,
		CopyMode:   cfg.CopyMode,
		SkipExist:  cfg.SkipExisting,
		DryRun:     dryRun,
		Fetcher:    fetcher,
		Writers:    nil, // TODO: Add writers based on format
	})

	// Print configuration
	if verbose {
		fmt.Println("Configuration:")
		fmt.Printf("  Source: %s\n", sourcePath)
		fmt.Printf("  Output: %s\n", outputPath)
		fmt.Printf("  Format: %s\n", pattern)
		fmt.Printf("  Mode: %s\n", modeString(cfg.CopyMode, dryRun))
		fmt.Printf("  Workers: %d\n", workers)
		fmt.Printf("  Skip existing: %v\n", cfg.SkipExisting)
		fmt.Println()
	}

	// Run pipeline
	ctx := context.Background()
	summary, err := pipeline.Run(ctx, sourcePath)
	if err != nil {
		return fmt.Errorf("pipeline failed: %w", err)
	}

	// Print summary
	fmt.Println("\nSummary:")
	fmt.Printf("  Total: %d\n", summary.Total)
	fmt.Printf("  Processed: %d\n", summary.Processed)
	fmt.Printf("  Skipped: %d\n", summary.Skipped)
	fmt.Printf("  Errors: %d\n", summary.Errors)
	fmt.Printf("  Duration: %s\n", summary.Duration)

	if summary.Errors > 0 && verbose {
		fmt.Println("\nFailures:")
		for _, failure := range summary.Failures {
			if failure.Audiobook != nil {
				fmt.Printf("  - %s: %v\n", failure.Audiobook.Path, failure.Error)
			}
		}
	}

	if summary.Errors > 0 {
		return fmt.Errorf("completed with %d errors", summary.Errors)
	}

	return nil
}

func modeString(copyMode, dryRun bool) string {
	if dryRun {
		return "dry-run"
	}
	if copyMode {
		return "copy"
	}
	return "move"
}
