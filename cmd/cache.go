package cmd

import (
	"fmt"
	"os"
	"tkngate/internal/cache"
	"tkngate/internal/config"

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
		if err := config.LoadConfig(); err != nil {
			pterm.Error.Println("Failed to load config:", err)
			os.Exit(1)
		}
		
		cache.InitCache(config.Cfg.Cache.MaxEntries, config.Cfg.Cache.TTLSeconds, config.Cfg.Cache.RedisURI)

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
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println(Gold("■ SEMANTIC CACHE STATUS"))
		fmt.Printf("  Entries:   %s\n", Gold(size))
		fmt.Printf("  Hits:      %s\n", Forest(hits))
		fmt.Printf("  Misses:    %s\n", pterm.LightRed(misses))
		fmt.Printf("  Hit Rate:  %s\n", Gold(fmt.Sprintf("%.1f%%", hitRate)))
		fmt.Printf("  Saved:     %s\n", Gold(fmt.Sprintf("$%.5f", savings)))
		fmt.Println(Forest("--------------------------------------------------------------------------------"))
		fmt.Println()
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all entries in the semantic cache",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.LoadConfig(); err != nil {
			pterm.Error.Println("Failed to load config:", err)
			os.Exit(1)
		}
		
		cache.InitCache(config.Cfg.Cache.MaxEntries, config.Cfg.Cache.TTLSeconds, config.Cfg.Cache.RedisURI)

		if cache.GlobalCache == nil {
			pterm.Warning.Println("Semantic cache is not initialised.")
			return
		}

		spinner, _ := pterm.DefaultSpinner.Start("Clearing semantic cache...")
		err := cache.GlobalCache.Clear()
		if err != nil {
			spinner.Fail("Failed to clear cache: ", err.Error())
			return
		}
		spinner.Success("Semantic cache successfully cleared")
	},
}

func init() {
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}
