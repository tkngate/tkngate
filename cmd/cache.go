package cmd

import (
	"fmt"
	"tkngate/internal/cache"

	"github.com/pterm/pterm"
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
			pterm.Warning.Println("Semantic cache is not initialised. Enable it in tkngate.yaml.")
			return
		}
		hits, misses, size, savings := cache.GlobalCache.Stats()
		total := hits + misses
		hitRate := float64(0)
		if total > 0 {
			hitRate = float64(hits) / float64(total) * 100
		}

		fmt.Println()
		pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgMagenta)).Println("Semantic Cache Status")
		fmt.Println()

		stats := pterm.Sprintf("Entries:   %s\n", pterm.LightCyan(size))
		stats += pterm.Sprintf("Hits:      %s\n", pterm.LightGreen(hits))
		stats += pterm.Sprintf("Misses:    %s\n", pterm.LightRed(misses))
		stats += pterm.Sprintf("Hit Rate:  %s\n", pterm.LightYellow(fmt.Sprintf("%.1f%%", hitRate)))
		stats += pterm.Sprintf("Saved:     %s", pterm.LightYellow(fmt.Sprintf("$%.5f", savings)))

		pterm.DefaultBox.WithRightPadding(4).WithLeftPadding(4).Println(stats)
		fmt.Println()
	},
}

func init() {
	cacheCmd.AddCommand(cacheStatusCmd)
	rootCmd.AddCommand(cacheCmd)
}
