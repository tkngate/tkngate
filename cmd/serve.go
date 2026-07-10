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
	"tkngate/internal/mesh"
	"tkngate/internal/pool"
	"tkngate/internal/proxy"
	"tkngate/internal/waf"
	"tkngate/internal/zkp"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the reverse proxy daemon",
	Run: func(cmd *cobra.Command, args []string) {
		// Init logger
		logging.InitLogger()

		// Silent brutalist startup for serve command
		fmt.Println()

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

		spinner, _ = pterm.DefaultSpinner.Start("Initializing WAF (Web Application Firewall)...")
		waf.InitWAF()
		if config.Cfg.WAF.Enabled {
			spinner.Success("WAF Engine active")
		} else {
			spinner.Warning("WAF Engine disabled")
		}

		if err := ensureCryptoInitialized(); err != nil {
			pterm.Error.Println("Crypto engine failed to initialise:", err.Error())
			logging.Logger.Error("FATAL: crypto engine failed to initialise", "error", err)
			os.Exit(1)
		}

		spinner, _ = pterm.DefaultSpinner.Start("Initializing DRR mesh pool...")
		pool.InitDRR()
		spinner.Success("DRR Mesh Pool active")

		if config.Cfg.Mesh.ReputationEnabled {
			spinner, _ = pterm.DefaultSpinner.Start("Initializing Stake-and-Slash Reputation Engine...")
			if err := mesh.InitReputation(budget.GlobalLedger.DB()); err != nil {
				spinner.Fail("Failed to init reputation: ", err.Error())
				logging.Logger.Error("failed to init reputation engine", "error", err)
				os.Exit(1)
			}
			spinner.Success("Reputation Engine active")
		} else {
			pterm.Info.Println("Reputation Engine disabled")
		}

		if config.Cfg.Mesh.StrictZKPMode {
			spinner, _ = pterm.DefaultSpinner.Start("Compiling ZK-SNARK circuit (Groth16)...")
			if err := zkp.Setup(); err != nil {
				spinner.Fail("Failed to compile ZKP circuit: ", err.Error())
				logging.Logger.Error("failed to init ZKP engine", "error", err)
				os.Exit(1)
			}
			spinner.Success("ZK-SNARK Engine active (Groth16 BN254)")
		} else {
			pterm.Info.Println("ZK-SNARK Engine disabled (enable with mesh.strict_zkp_mode)")
		}

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

		fmt.Println(Gold("----------------------------------------------------"))
		fmt.Println(Gold(fmt.Sprintf("> DAEMON %s ONLINE", rootCmd.Version)))
		
		highlight := pterm.NewRGBStyle(pterm.NewRGB(22, 43, 29), pterm.NewRGB(184, 151, 82)).Sprint
		fmt.Printf("%s %s\n", Gold("PROXY ->"), highlight("http://"+addr))
		if config.Cfg.Telemetry.Enabled {
			fmt.Printf("%s %s\n", Gold("ADMIN ->"), highlight(fmt.Sprintf("http://%s:%d", config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port)))
		}
		fmt.Println()
		fmt.Println(Gold("■ ENGINE ONLINE. AWAITING SWARM."))

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
