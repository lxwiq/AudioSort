package cli

import (
	"fmt"
	"os"

	"audiosort/internal/config"
	"audiosort/internal/tui"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan and organize audiobooks",
	Long: `Scan a directory for audiobooks, fetch metadata from multiple sources,
and organize them into a structured format.

This command launches an interactive TUI where you can:
- Review detected audiobooks
- Select which ones to process
- Monitor progress in real-time`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringP("output", "o", "", "output directory")
	scanCmd.Flags().String("format", "audiobookshelf", "output format (audiobookshelf|plex|json|all)")
	scanCmd.Flags().Bool("copy", false, "copy files instead of moving")
	scanCmd.Flags().Int("workers", 4, "number of parallel workers")
}

func runScan(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]

	// Validate source path
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("source path does not exist: %w", err)
	}

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Override with flags
	if output, _ := cmd.Flags().GetString("output"); output != "" {
		cfg.DefaultOutput = output
	}
	if format, _ := cmd.Flags().GetString("format"); format != "" {
		cfg.OutputFormat = format
	}
	if copyMode, _ := cmd.Flags().GetBool("copy"); copyMode {
		cfg.CopyMode = true
	}
	if workers, _ := cmd.Flags().GetInt("workers"); workers > 0 {
		cfg.ParallelWorkers = workers
	}

	// Launch TUI
	return tui.Run(sourcePath, cfg)
}
