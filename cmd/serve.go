package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"tkngate/internal/budget"
	"tkngate/internal/config"
	"tkngate/internal/logging"
	"tkngate/internal/proxy"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the reverse proxy daemon",
	Run: func(cmd *cobra.Command, args []string) {
		// Init logger
		logging.InitLogger()

		// Load config
		if err := config.LoadConfig(); err != nil {
			logging.Logger.Error("failed to load config", "error", err)
			os.Exit(1)
		}

		// Init budget ledger
		if err := budget.InitLedger(); err != nil {
			logging.Logger.Error("failed to init ledger", "error", err)
			os.Exit(1)
		}

		// Init proxy
		p, err := proxy.NewProxy()
		if err != nil {
			logging.Logger.Error("failed to create proxy", "error", err)
			os.Exit(1)
		}

		addr := fmt.Sprintf("%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port)
		logging.Logger.Info("starting tkngate daemon", "address", addr)

		server := &http.Server{
			Addr:    addr,
			Handler: p,
		}

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logging.Logger.Error("server error", "error", err)
				os.Exit(1)
			}
		}()

		// Graceful shutdown
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		logging.Logger.Info("shutting down server...")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
