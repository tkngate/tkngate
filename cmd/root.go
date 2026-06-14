package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var tkngateBanner = `
   ______   __   __    _  __    ______   ___   ______   ______ 
  /_  __/  / /  / /   / |/ /   / ____/  /   | /_  __/  / ____/ 
   / /    / /__/ /   /    /   / / __   / /| |  / /    / __/    
  / /    / /  / /   / /|  /  / /_/ /  / ___ | / /    / /___    
 /_/    /_/  /_/   /_/ |_/   \____/  /_/  |_|/_/    /_____/    
                                                               
        The Cloudflare for Autonomous AI Agents
`

var rootCmd = &cobra.Command{
	Use:     "tkngate",
	Version: "v1.7.0",
	Short:   "tkngate is an enterprise token-management reverse proxy",
	Long: color.GreenString(tkngateBanner) + `
tkngate is an open-source, zero-knowledge reverse proxy daemon for LLM APIs.
It provides P2P token pooling, real-time budget enforcement, and semantic caching.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags here if needed, like config file path
}
