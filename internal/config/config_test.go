package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Check default sources
	expectedSources := []string{"googlebooks", "openlibrary", "audible", "bnf"}
	if len(cfg.Sources) != len(expectedSources) {
		t.Errorf("Sources length = %d, want %d", len(cfg.Sources), len(expectedSources))
	}
	for i, src := range expectedSources {
		if cfg.Sources[i] != src {
			t.Errorf("Sources[%d] = %q, want %q", i, cfg.Sources[i], src)
		}
	}

	// Check other defaults
	if cfg.OutputFormat != "audiobookshelf" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "audiobookshelf")
	}
	if cfg.DefaultOutput != "" {
		t.Errorf("DefaultOutput = %q, want empty string", cfg.DefaultOutput)
	}
	if cfg.CopyMode != false {
		t.Errorf("CopyMode = %v, want false", cfg.CopyMode)
	}
	if cfg.ParallelWorkers != 4 {
		t.Errorf("ParallelWorkers = %d, want 4", cfg.ParallelWorkers)
	}
	if cfg.SkipExisting != true {
		t.Errorf("SkipExisting = %v, want true", cfg.SkipExisting)
	}
	if cfg.PreferredLanguage != "en" {
		t.Errorf("PreferredLanguage = %q, want %q", cfg.PreferredLanguage, "en")
	}
}

func TestConfig_YAMLSerialization(t *testing.T) {
	cfg := &Config{
		Sources:           []string{"source1", "source2"},
		OutputFormat:      "json",
		DefaultOutput:     "/tmp/output",
		CopyMode:          true,
		ParallelWorkers:   8,
		SkipExisting:      false,
		PreferredLanguage: "fr",
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	// Unmarshal back
	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// Verify all fields
	if len(loaded.Sources) != 2 {
		t.Errorf("Sources length = %d, want 2", len(loaded.Sources))
	}
	if loaded.OutputFormat != "json" {
		t.Errorf("OutputFormat = %q, want %q", loaded.OutputFormat, "json")
	}
	if loaded.DefaultOutput != "/tmp/output" {
		t.Errorf("DefaultOutput = %q, want %q", loaded.DefaultOutput, "/tmp/output")
	}
	if loaded.CopyMode != true {
		t.Errorf("CopyMode = %v, want true", loaded.CopyMode)
	}
	if loaded.ParallelWorkers != 8 {
		t.Errorf("ParallelWorkers = %d, want 8", loaded.ParallelWorkers)
	}
	if loaded.SkipExisting != false {
		t.Errorf("SkipExisting = %v, want false", loaded.SkipExisting)
	}
	if loaded.PreferredLanguage != "fr" {
		t.Errorf("PreferredLanguage = %q, want %q", loaded.PreferredLanguage, "fr")
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	cfg := &Config{
		Sources:           []string{"test1", "test2"},
		OutputFormat:      "plex",
		DefaultOutput:     "/test/output",
		CopyMode:          true,
		ParallelWorkers:   2,
		SkipExisting:      false,
		PreferredLanguage: "de",
	}

	// Save to file
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	// Load from file
	loadedData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(loadedData, &loaded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// Verify
	if loaded.OutputFormat != "plex" {
		t.Errorf("OutputFormat = %q, want %q", loaded.OutputFormat, "plex")
	}
	if loaded.ParallelWorkers != 2 {
		t.Errorf("ParallelWorkers = %d, want 2", loaded.ParallelWorkers)
	}
}

func TestConfig_PartialYAML(t *testing.T) {
	// Test that partial YAML only overrides specified fields
	cfg := DefaultConfig()

	partialYAML := `
output_format: json
copy_mode: true
`
	if err := yaml.Unmarshal([]byte(partialYAML), cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// Overridden fields
	if cfg.OutputFormat != "json" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "json")
	}
	if cfg.CopyMode != true {
		t.Errorf("CopyMode = %v, want true", cfg.CopyMode)
	}

	// Preserved defaults
	if cfg.ParallelWorkers != 4 {
		t.Errorf("ParallelWorkers = %d, want 4 (should be preserved)", cfg.ParallelWorkers)
	}
	if cfg.PreferredLanguage != "en" {
		t.Errorf("PreferredLanguage = %q, want %q (should be preserved)", cfg.PreferredLanguage, "en")
	}
}

func TestConfig_EmptyYAML(t *testing.T) {
	cfg := DefaultConfig()

	if err := yaml.Unmarshal([]byte(""), cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// All defaults should be preserved
	if cfg.OutputFormat != "audiobookshelf" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "audiobookshelf")
	}
	if cfg.ParallelWorkers != 4 {
		t.Errorf("ParallelWorkers = %d, want 4", cfg.ParallelWorkers)
	}
}

func TestConfig_YAMLTags(t *testing.T) {
	// Verify YAML field names match expected tags
	cfg := &Config{
		Sources:           []string{"test"},
		OutputFormat:      "test",
		DefaultOutput:     "test",
		CopyMode:          true,
		ParallelWorkers:   1,
		SkipExisting:      true,
		PreferredLanguage: "test",
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	yamlStr := string(data)

	expectedFields := []string{
		"sources:",
		"output_format:",
		"default_output:",
		"copy_mode:",
		"parallel_workers:",
		"skip_existing:",
		"preferred_language:",
	}

	for _, field := range expectedFields {
		if !contains(yamlStr, field) {
			t.Errorf("YAML output missing field %q", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
