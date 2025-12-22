package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"audiosort/internal/config"
	"audiosort/internal/metadata"
)

// initMetadataSources creates metadata sources based on configuration
func initMetadataSources(cfg *config.Config) []metadata.MetadataSource {
	var sources []metadata.MetadataSource
	httpClient := &http.Client{}

	for _, sourceName := range cfg.Sources {
		switch sourceName {
		case "googlebooks":
			sources = append(sources, metadata.NewGoogleBooks(httpClient))
		case "openlibrary":
			sources = append(sources, metadata.NewOpenLibrary(httpClient))
		// TODO: Add other sources (audible, bnf, etc.)
		}
	}

	return sources
}

// loadConfig loads the configuration from file or returns default
func loadConfig() (*config.Config, error) {
	if cfgFile != "" {
		// Load from specified file
		// TODO: Implement custom config file loading
		return config.Load()
	}
	return config.Load()
}

// getCacheDir returns the cache directory path, creating it if needed
func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".cache", "audiosort")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	return cacheDir, nil
}
