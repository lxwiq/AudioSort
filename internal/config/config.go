package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sources           []string `yaml:"sources"`
	OutputFormat      string   `yaml:"output_format"`
	DefaultOutput     string   `yaml:"default_output"`
	CopyMode          bool     `yaml:"copy_mode"`
	ParallelWorkers   int      `yaml:"parallel_workers"`
	SkipExisting      bool     `yaml:"skip_existing"`
	PreferredLanguage string   `yaml:"preferred_language"`
}

// AvailableSources is the canonical list of metadata sources, in the default
// query order ("bookinfo first" for better audiobook metadata). It is the
// single source of truth shared by the config defaults and the TUI editor.
var AvailableSources = []string{"bookinfo", "googlebooks", "openlibrary"}

// AvailableFormats is the canonical list of output presets exposed in the UI.
// Only presets that map to an organization pattern are listed; writer-only
// outputs (json/all) are intentionally excluded until they are wired up.
var AvailableFormats = []string{"audiobookshelf", "plex"}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	defaultOutput := filepath.Join(home, "Audiobooks-organized")

	return &Config{
		Sources:           append([]string(nil), AvailableSources...),
		OutputFormat:      "audiobookshelf",
		DefaultOutput:     defaultOutput,
		CopyMode:          false,
		ParallelWorkers:   4,
		SkipExisting:      true,
		PreferredLanguage: "fr",
	}
}

// CachePath returns the single canonical path to the on-disk metadata cache.
func CachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".cache", "audiosort", "metadata.db")
	}
	return filepath.Join(home, ".cache", "audiosort", "metadata.db")
}

// Validate performs lightweight sanity checks before persisting the config.
func (c *Config) Validate() error {
	if c.PreferredLanguage == "" {
		return fmt.Errorf("preferred language must not be empty")
	}
	if c.DefaultOutput == "" {
		return fmt.Errorf("default output directory must not be empty")
	}
	if c.ParallelWorkers < 1 {
		return fmt.Errorf("parallel workers must be at least 1")
	}
	return nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".config", "audiosort", "config.yaml"), nil
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save() error {
	if err := c.Validate(); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
