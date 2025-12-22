package cli

import (
	"fmt"

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
	// TUI will be implemented with the internal/tui package later
	return fmt.Errorf("TUI not implemented yet - coming soon!")
}
