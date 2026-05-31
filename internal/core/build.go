package core

import (
	"net/http"
	"strings"
	"time"

	"audiosort/internal/cache"
	"audiosort/internal/config"
	"audiosort/internal/metadata"
)

// metadataHTTPTimeout bounds every outbound metadata request.
const metadataHTTPTimeout = 15 * time.Second

// MetadataSources builds the metadata sources described by the configuration,
// sharing a single HTTP client. When the config lists no sources it falls back
// to config.AvailableSources (canonical "bookinfo first" order). This is the
// single place that maps source names to implementations, so the scan pipeline
// and the search view can no longer drift apart.
func MetadataSources(cfg *config.Config) []metadata.MetadataSource {
	client := &http.Client{Timeout: metadataHTTPTimeout}
	region := audibleRegion(cfg.PreferredLanguage)

	sources := buildSources(cfg.Sources, client, region)
	if len(sources) == 0 {
		sources = buildSources(config.AvailableSources, client, region)
	}
	return sources
}

func buildSources(names []string, client *http.Client, region string) []metadata.MetadataSource {
	var sources []metadata.MetadataSource
	for _, name := range names {
		switch name {
		case "audible":
			sources = append(sources, metadata.NewAudible(client, region))
		case "bookinfo":
			sources = append(sources, metadata.NewBookInfo(client))
		case "googlebooks":
			sources = append(sources, metadata.NewGoogleBooks(client))
		case "openlibrary":
			sources = append(sources, metadata.NewOpenLibrary(client))
		}
	}
	return sources
}

// audibleRegion maps a preferred-language hint to the matching Audible TLD.
func audibleRegion(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "fr", "fr-fr", "french":
		return "fr"
	case "de", "de-de", "german":
		return "de"
	case "es", "spanish":
		return "es"
	case "it", "italian":
		return "it"
	case "ja", "jp", "japanese":
		return "co.jp"
	case "en-gb", "uk":
		return "co.uk"
	case "en-au":
		return "com.au"
	default:
		return "com"
	}
}

// BuildFetcher assembles a metadata.Fetcher from the configured sources, reusing
// the caller-provided cache store. The store may be nil (the cache is optional).
func BuildFetcher(cfg *config.Config, store *cache.Store) *metadata.Fetcher {
	return metadata.NewFetcher(MetadataSources(cfg), store)
}

// BuildFetcherFromConfig is a convenience wrapper for callers that do not yet
// hold a cache handle: it opens the shared on-disk cache and returns it so the
// caller can reuse the single handle (bbolt takes an exclusive file lock).
// The returned store is nil if the cache could not be opened.
func BuildFetcherFromConfig(cfg *config.Config) (*metadata.Fetcher, *cache.Store) {
	store, _ := cache.NewStore(config.CachePath())
	return BuildFetcher(cfg, store), store
}

// BuildPipeline builds a processing pipeline for the given source/destination
// using the supplied fetcher.
func BuildPipeline(cfg *config.Config, sourcePath, destPath string, fetcher *metadata.Fetcher) *Pipeline {
	return NewPipeline(PipelineOptions{
		SourcePath: sourcePath,
		DestPath:   destPath,
		Workers:    cfg.ParallelWorkers,
		CopyMode:   cfg.CopyMode,
		SkipExist:  cfg.SkipExisting,
		DryRun:     false,
		Fetcher:    fetcher,
	})
}
