package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"audiosort/internal/cache"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage metadata cache",
	Long:  `View and manage the metadata cache database.`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached metadata",
	Long:  `Delete the cache database, removing all cached metadata.`,
	RunE:  runCacheClear,
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache information",
	Long:  `Display information about the metadata cache.`,
	RunE:  runCacheInfo,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheInfoCmd)

	cacheClearCmd.Flags().Bool("force", false, "skip confirmation prompt")
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	cacheDir, err := getCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	cachePath := filepath.Join(cacheDir, "cache.db")

	// Check if cache exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		fmt.Println("Cache does not exist.")
		return nil
	}

	// Confirm deletion
	if !force {
		fmt.Print("Are you sure you want to clear the cache? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Delete cache file
	if err := os.Remove(cachePath); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	fmt.Println("Cache cleared successfully.")
	return nil
}

func runCacheInfo(cmd *cobra.Command, args []string) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	cachePath := filepath.Join(cacheDir, "cache.db")

	// Check if cache exists
	info, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		fmt.Println("Cache Status: Not initialized")
		fmt.Printf("Cache Path: %s\n", cachePath)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat cache: %w", err)
	}

	fmt.Println("Cache Information:")
	fmt.Printf("  Path: %s\n", cachePath)
	fmt.Printf("  Size: %d bytes\n", info.Size())
	fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	// Try to open cache and get stats
	store, err := cache.NewStore(cachePath)
	if err != nil {
		fmt.Printf("  Status: Unable to open (may be in use)\n")
		return nil
	}
	defer store.Close()

	fmt.Println("  Status: OK")

	return nil
}
