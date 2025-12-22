package cli

import (
	"path/filepath"

	"audiosort/internal/tui"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage metadata cache",
	Long: `View and manage the metadata cache.

This command launches an interactive TUI where you can:
- View cache statistics
- Clear cached metadata`,
	RunE: runCache,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
}

func runCache(cmd *cobra.Command, args []string) error {
	cacheDir, _ := getCacheDir()
	cachePath := filepath.Join(cacheDir, "metadata.db")
	return tui.RunCache(cachePath)
}
