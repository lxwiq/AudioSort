package cli

import (
	"audiosort/internal/config"
	"audiosort/internal/tui"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui [path]",
	Short: "Launch interactive TUI",
	Long: `Launch the interactive Terminal User Interface (TUI) for browsing
and organizing audiobooks visually.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUI,
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func runUI(cmd *cobra.Command, args []string) error {
	// Get source path (default to current directory)
	sourcePath := "."
	if len(args) > 0 {
		sourcePath = args[0]
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Run the TUI
	return tui.Run(sourcePath, cfg)
}
