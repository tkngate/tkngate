package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
	"tkngate/internal/api"
	"tkngate/internal/budget"
	"tkngate/internal/cache"
	"tkngate/internal/config"
	"tkngate/internal/limiter"
	"tkngate/internal/logging"
	"tkngate/internal/pool"
	"tkngate/internal/proxy"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the reverse proxy daemon",
	Run: func(cmd *cobra.Command, args []string) {
		// Init logger
		logging.InitLogger()

		// Print sexy banner
		pterm.DefaultBigText.WithLetters(
			pterm.NewLettersFromStringWithStyle("TKN", pterm.NewStyle(pterm.FgCyan)),
			pterm.NewLettersFromStringWithStyle("GATE", pterm.NewStyle(pterm.FgLightBlue)),
		).Render()
		pterm.DefaultCenter.Println(pterm.LightMagenta("✦ ENGINE STARTUP SEQUENCE ✦\n"))

		spinner, _ := pterm.DefaultSpinner.Start("Loading configuration...")
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Failed to load config: ", err.Error())
			logging.Logger.Error("failed to load config", "error", err)
			os.Exit(1)
		}
		spinner.Success("Configuration loaded")

		spinner, _ = pterm.DefaultSpinner.Start("Initializing budget ledger...")
		if err := budget.InitLedger(); err != nil {
			spinner.Fail("Failed to init ledger: ", err.Error())
			logging.Logger.Error("failed to init ledger", "error", err)
			os.Exit(1)
		}
		spinner.Success("Ledger & Budget Guard active")

		if err := ensureCryptoInitialized(); err != nil {
			pterm.Error.Println("Crypto engine failed to initialise:", err.Error())
			logging.Logger.Error("FATAL: crypto engine failed to initialise", "error", err)
			os.Exit(1)
		}

		spinner, _ = pterm.DefaultSpinner.Start("Initializing DRR mesh pool...")
		pool.InitDRR()
		spinner.Success("DRR Mesh Pool active")

		limiter.GlobalManager.StartCleanup()
		if config.Cfg.RateLimit.Enabled {
			pterm.Success.Printf("Rate Limiter active (%d req/min)\n", config.Cfg.RateLimit.RequestsPerMinute)
		} else {
			pterm.Info.Println("Rate Limiter disabled")
		}

		if config.Cfg.Cache.Enabled {
			cache.InitCache(config.Cfg.Cache.MaxEntries, config.Cfg.Cache.TTLSeconds, config.Cfg.Cache.RedisURI)
			pterm.Success.Printf("Semantic Cache active (Max: %d entries)\n", config.Cfg.Cache.MaxEntries)
		} else {
			pterm.Info.Println("Semantic Cache disabled")
		}

		if config.Cfg.Telemetry.Enabled {
			go func() {
				if err := api.StartTelemetryServer(config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port); err != nil && err != http.ErrServerClosed {
					logging.Logger.Error("Telemetry API server error", "error", err)
				}
			}()
			pterm.Success.Printf("Telemetry API active on port %d\n", config.Cfg.Telemetry.Port)
		}

		spinner, _ = pterm.DefaultSpinner.Start("Starting proxy server...")
		p, err := proxy.NewProxy()
		if err != nil {
			spinner.Fail("Failed to create proxy: ", err.Error())
			logging.Logger.Error("failed to create proxy", "error", err)
			os.Exit(1)
		}
		spinner.Success("Proxy engine ready")

		addr := fmt.Sprintf("%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port)

		fmt.Println()

		boxContent := pterm.Sprintf("Tkngate Daemon %s is online!\nProxy Address: %s", rootCmd.Version, pterm.LightCyan("http://"+addr))
		if config.Cfg.Telemetry.Enabled {
			boxContent += pterm.Sprintf("\nTelemetry API: %s", pterm.LightCyan(fmt.Sprintf("http://%s:%d", config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port)))
		}

		pterm.DefaultBox.WithRightPadding(2).WithLeftPadding(2).Println(boxContent)

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
		pterm.Info.Println("\nShutting down server...")
		logging.Logger.Info("shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logging.Logger.Error("Server forced to shutdown", "error", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
