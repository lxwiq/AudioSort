package cli

import (
	"audiosort/internal/config"
	"audiosort/internal/tui"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for book metadata",
	Long: `Search for audiobook metadata across multiple sources.

This command launches an interactive TUI where you can:
- Enter search queries
- Browse results from Google Books, Open Library, etc.
- View detailed book information`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Launch search TUI
	return tui.RunSearch(query, cfg)
}
