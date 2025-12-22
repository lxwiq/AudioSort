package cli

import (
	"audiosort/internal/config"
	"audiosort/internal/tui"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `View and manage AudioSort configuration.

This command launches an interactive TUI where you can:
- View current settings
- Save configuration
- Reset to defaults`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	return tui.RunConfig(cfg)
}
