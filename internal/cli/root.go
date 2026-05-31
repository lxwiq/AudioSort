package cli

import (
	"audiosort/internal/config"
	"audiosort/internal/tui"
	"audiosort/internal/version"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "audiosort",
	Short: "Organize audiobook collections with automatic metadata",
	Long: `AudioSort scans audiobook folders, fetches metadata from multiple sources,
and organizes files into structured formats compatible with popular players
like AudiobookShelf and SmartAudioBookPlayer.

Run without arguments to launch the interactive menu.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version.String(),
	RunE:          runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Launch main menu TUI
	return tui.RunMenuWithAction(cfg)
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/audiosort/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
