package cmd

import (
	"fmt"
	"runtime"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// These variables are injected at build time using -ldflags
var (
	// Version is the current version of tkngate
	Version   = "v2.8.1"
	BuildDate = "2026-07-21"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of TknGate",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println()
		pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).WithTextStyle(pterm.NewStyle(pterm.FgBlack)).Println("TKNGATE ENTERPRISE")
		fmt.Println()
		
		tableData := pterm.TableData{
			{"Version", Gold(Version)},
			{"Build Date", BuildDate},
			{"OS/Arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
			{"Go Version", runtime.Version()},
		}
		
		pterm.DefaultTable.WithData(tableData).Render()
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
