package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"audiosort/internal/config"
)

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
