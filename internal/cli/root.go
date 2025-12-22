package cli

import (
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
like AudiobookShelf and SmartAudioBookPlayer.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/audiosort/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
