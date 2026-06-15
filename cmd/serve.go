package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"tkngate/internal/budget"
	"tkngate/internal/cache"
	"tkngate/internal/config"
	"tkngate/internal/crypto"
	"tkngate/internal/limiter"
	"tkngate/internal/logging"
	"tkngate/internal/pool"
	"tkngate/internal/proxy"
	"tkngate/internal/api"

	"github.com/fatih/color"
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

		// Init crypto engine (FATAL if missing — mesh encryption requires this)
		if err := crypto.InitCrypto(); err != nil {
			logging.Logger.Error("FATAL: crypto engine failed to initialise. Set TKNGATE_MASTER_KEY (32 chars). Run: tkngate config generate-master-key", "error", err)
			os.Exit(1)
		}

		// Init DRR
		pool.InitDRR()

		// Init rate limiter cleanup (evicts stale limiters every 5 min)
		limiter.GlobalManager.StartCleanup()

		// Init semantic cache
		if config.Cfg.Cache.Enabled {
			cache.InitCache(config.Cfg.Cache.MaxEntries, config.Cfg.Cache.TTLSeconds, config.Cfg.Cache.RedisURI)
			logging.Logger.Info("Semantic cache enabled")
		}

		// Init Telemetry API
		if config.Cfg.Telemetry.Enabled {
			go func() {
				if err := api.StartTelemetryServer(config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port); err != nil && err != http.ErrServerClosed {
					logging.Logger.Error("Telemetry API server error", "error", err)
				}
			}()
		}

		// Init proxy
		p, err := proxy.NewProxy()
		if err != nil {
			logging.Logger.Error("failed to create proxy", "error", err)
			os.Exit(1)
		}

		addr := fmt.Sprintf("%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port)
		
		// Print sexy banner
		color.Green(tkngateBanner)
		color.White("Starting tkngate daemon (%s) on %s\n", rootCmd.Version, color.GreenString("http://%s", addr))
		color.White("Telemetry API active on %s\n", color.CyanString("http://%s:%d", config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port))
		fmt.Println(color.HiBlackString("─────────────────────────────────────────────────────────────────────────"))
		
		logging.Logger.Info("proxy engine online", "address", addr)

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
