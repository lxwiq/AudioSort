package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"audiosort/internal/cache"
	"audiosort/internal/metadata"
	"audiosort/pkg/models"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for book metadata",
	Long: `Search for audiobook metadata from configured sources.
Returns detailed information about matching books.`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().IntP("limit", "n", 5, "maximum number of results")
	searchCmd.Flags().StringP("source", "s", "", "specific source to search (googlebooks, openlibrary)")
	searchCmd.Flags().Bool("no-cache", false, "ignore cached results")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get flags
	limit, _ := cmd.Flags().GetInt("limit")
	sourceName, _ := cmd.Flags().GetString("source")
	noCache, _ := cmd.Flags().GetBool("no-cache")

	// Initialize cache
	var cacheStore *cache.Store
	if !noCache {
		cacheDir, err := getCacheDir()
		if err == nil {
			cacheStore, err = cache.NewStore(filepath.Join(cacheDir, "cache.db"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to initialize cache: %v\n", err)
			}
			if cacheStore != nil {
				defer cacheStore.Close()

				// Check cache first
				if cached, ok := cacheStore.GetMetadata(query); ok {
					fmt.Println("Found in cache:")
					printMetadata(&cached.Metadata, 1)
					return nil
				}
			}
		}
	}

	// Initialize metadata sources
	var sources []metadata.MetadataSource
	httpClient := &http.Client{}

	if sourceName != "" {
		// Use specific source
		switch sourceName {
		case "googlebooks":
			sources = append(sources, metadata.NewGoogleBooks(httpClient))
		case "openlibrary":
			sources = append(sources, metadata.NewOpenLibrary(httpClient))
		default:
			return fmt.Errorf("unknown source: %s", sourceName)
		}
	} else {
		// Use all configured sources
		sources = initMetadataSources(cfg)
	}

	if len(sources) == 0 {
		return fmt.Errorf("no metadata sources configured")
	}

	// Search all sources
	ctx := context.Background()
	fmt.Printf("Searching for: %s\n\n", query)

	foundResults := false
	for _, source := range sources {
		if verbose {
			fmt.Printf("Searching %s...\n", source.Name())
		}

		results, err := source.Search(ctx, query)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			}
			continue
		}

		if len(results) == 0 {
			if verbose {
				fmt.Println("  No results")
			}
			continue
		}

		foundResults = true
		fmt.Printf("Results from %s:\n", source.Name())

		count := limit
		if count > len(results) {
			count = len(results)
		}

		for i := 0; i < count; i++ {
			printMetadata(&results[i], i+1)
		}

		fmt.Println()
	}

	if !foundResults {
		return fmt.Errorf("no results found for query: %s", query)
	}

	return nil
}

func printMetadata(m *models.BookMetadata, index int) {
	fmt.Printf("%d. %s\n", index, m.Title)
	if len(m.Authors) > 0 {
		fmt.Printf("   Author: %s\n", strings.Join(m.Authors, ", "))
	}
	if len(m.Narrators) > 0 {
		fmt.Printf("   Narrator: %s\n", strings.Join(m.Narrators, ", "))
	}
	if m.Series != "" {
		fmt.Printf("   Series: %s", m.Series)
		if m.SeriesPosition != "" {
			fmt.Printf(" #%s", m.SeriesPosition)
		}
		fmt.Println()
	}
	if m.Publisher != "" {
		fmt.Printf("   Publisher: %s", m.Publisher)
		if m.PublishYear != "" {
			fmt.Printf(" (%s)", m.PublishYear)
		}
		fmt.Println()
	}
	if m.Language != "" {
		fmt.Printf("   Language: %s\n", m.Language)
	}
	if len(m.Genres) > 0 {
		fmt.Printf("   Genres: %s\n", strings.Join(m.Genres, ", "))
	}
	if m.ISBN != "" {
		fmt.Printf("   ISBN: %s\n", m.ISBN)
	}
	if m.ASIN != "" {
		fmt.Printf("   ASIN: %s\n", m.ASIN)
	}
	if m.SourceURL != "" {
		fmt.Printf("   URL: %s\n", m.SourceURL)
	}
	if m.Summary != "" && verbose {
		summary := m.Summary
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		fmt.Printf("   Summary: %s\n", summary)
	}
}
