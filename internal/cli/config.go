package cli

import (
	"fmt"
	"strings"

	"audiosort/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `View and manage AudioSort configuration.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration values.`,
	RunE:  runConfigShow,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration",
	Long:  `Create a default configuration file at ~/.config/audiosort/config.yaml.`,
	RunE:  runConfigInit,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)

	configInitCmd.Flags().Bool("force", false, "overwrite existing configuration")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Current Configuration:")
	fmt.Println()
	fmt.Printf("  Sources: %s\n", strings.Join(cfg.Sources, ", "))
	fmt.Printf("  Output Format: %s\n", cfg.OutputFormat)
	fmt.Printf("  Default Output: %s\n", displayString(cfg.DefaultOutput))
	fmt.Printf("  Copy Mode: %v\n", cfg.CopyMode)
	fmt.Printf("  Parallel Workers: %d\n", cfg.ParallelWorkers)
	fmt.Printf("  Skip Existing: %v\n", cfg.SkipExisting)
	fmt.Printf("  Preferred Language: %s\n", cfg.PreferredLanguage)

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	// Check if config already exists
	if !force {
		if _, err := config.Load(); err == nil {
			return fmt.Errorf("configuration already exists, use --force to overwrite")
		}
	}

	// Create default config
	cfg := config.DefaultConfig()
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("Configuration initialized successfully!")
	fmt.Println()
	fmt.Println("Default configuration created at:")
	fmt.Println("  ~/.config/audiosort/config.yaml")
	fmt.Println()
	fmt.Println("Edit this file to customize your settings.")

	return nil
}

func displayString(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}
