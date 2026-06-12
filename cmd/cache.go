package cmd

import (
	"fmt"
	"tkngate/internal/cache"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Semantic cache management commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show semantic cache statistics",
	Run: func(cmd *cobra.Command, args []string) {
		if cache.GlobalCache == nil {
			fmt.Println("⚠️  Semantic cache is not initialised. Enable it in tkngate.yaml.")
			return
		}
		hits, misses, size, savings := cache.GlobalCache.Stats()
		total := hits + misses
		hitRate := float64(0)
		if total > 0 {
			hitRate = float64(hits) / float64(total) * 100
		}
		fmt.Println("🧠 Semantic Cache Status")
		fmt.Printf("   Entries:   %d\n", size)
		fmt.Printf("   Hits:      %d\n", hits)
		fmt.Printf("   Misses:    %d\n", misses)
		fmt.Printf("   Hit Rate:  %.1f%%\n", hitRate)
		fmt.Printf("   💰 Saved:  $%.5f\n", savings)
	},
}

func init() {
	cacheCmd.AddCommand(cacheStatusCmd)
	rootCmd.AddCommand(cacheCmd)
}
