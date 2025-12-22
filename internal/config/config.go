package config

import (
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

func DefaultConfig() *Config {
	return &Config{
		Sources:           []string{"googlebooks", "openlibrary", "audible", "bnf"},
		OutputFormat:      "audiobookshelf",
		DefaultOutput:     "",
		CopyMode:          false,
		ParallelWorkers:   4,
		SkipExisting:      true,
		PreferredLanguage: "en",
	}
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "audiosort", "config.yaml")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(configPath())
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
	path := configPath()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
