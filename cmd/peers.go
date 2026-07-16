package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"tkngate/internal/p2p"
	"tkngate/internal/config"
)

var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "Manage and view P2P mesh network peers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'tkngate peers list' to view connected nodes.")
	},
}

var listPeersCmd = &cobra.Command{
	Use:   "list",
	Short: "List all connected P2P peers",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.LoadConfig(); err != nil {
			fmt.Println("Error loading config:", err)
			os.Exit(1)
		}

		if !config.Cfg.P2P.Enabled {
			fmt.Println("P2P mesh is disabled in config.")
			os.Exit(1)
		}

		if err := p2p.LoadOrGenerateIdentity(); err != nil {
			fmt.Println("Error loading identity:", err)
			os.Exit(1)
		}

		ctx := context.Background()
		if err := p2p.InitHost(ctx); err != nil {
			fmt.Println("Error initializing P2P host:", err)
			os.Exit(1)
		}

		if err := p2p.InitProtocols(); err != nil {
			fmt.Println("Error initializing P2P protocols:", err)
			os.Exit(1)
		}

		fmt.Printf("\nYour Node ID: %s\n", p2p.GlobalIdentity.PeerID.String())
		fmt.Printf("Listening on: %s\n\n", p2p.GlobalHost.Addrs())
		
		fmt.Println("Discovering peers on the mesh... (waiting 5 seconds)")
		time.Sleep(5 * time.Second)

		peers := p2p.GlobalHost.Network().Peers()
		if len(peers) == 0 {
			fmt.Println("No peers connected.")
			return
		}

		fmt.Printf("Connected to %d peers:\n", len(peers))
		fmt.Println("-----------------------------------------------------")
		fmt.Printf("%-55s %s\n", "Peer ID", "Latency")
		fmt.Println("-----------------------------------------------------")

		for _, p := range peers {
			latency, err := p2p.SendPing(ctx, p)
			latencyStr := "N/A"
			if err == nil {
				latencyStr = latency.String()
			}
			fmt.Printf("%-55s %s\n", p.String(), latencyStr)
		}
		fmt.Println("-----------------------------------------------------")
	},
}

func init() {
	rootCmd.AddCommand(peersCmd)
	peersCmd.AddCommand(listPeersCmd)
}
