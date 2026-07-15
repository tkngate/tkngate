package cmd

import (
	"context"
	"fmt"
	"math/rand"
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

var demoMode bool

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
			fmt.Printf("%s %s\n", Gold("DASHBOARD ->"), highlight(fmt.Sprintf("http://%s:%d", config.Cfg.Telemetry.Host, config.Cfg.Telemetry.Port)))
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

		// Launch demo traffic generator if --demo flag is set
		if demoMode {
			go runDemoTraffic()
		}

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
	serveCmd.Flags().BoolVar(&demoMode, "demo", false, "Start the demo traffic generator alongside the proxy server")
	rootCmd.AddCommand(serveCmd)
}

// runDemoTraffic injects simulated traffic into the live budget.db so the dashboard shows activity.
func runDemoTraffic() {
	fmt.Println()
	fmt.Println(Gold("■ DEMO MODE: Generating simulated traffic..."))

	db := budget.GlobalLedger.DB()

	// Seed demo reputation peers
	var repCount int
	db.QueryRow("SELECT COUNT(*) FROM mesh_reputation").Scan(&repCount)
	if repCount == 0 {
		demoPeers := []struct {
			nodeID      string
			trust       float64
			requests    int
			violations  int
			blacklisted int
		}{
			{"a1f8c3e2-9b74-4d01-8e56-3fa912bc7d80", 95.2, 14832, 0, 0},
			{"b7d24f19-6a83-42e5-9c11-e8f0a5d63b21", 88.7, 11204, 1, 0},
			{"c4e91a56-3f28-4b7d-a042-7c8d6e5f9130", 72.4, 6891, 2, 0},
			{"d9b37e84-1c65-49af-b823-4a0f2d7e8c59", 65.1, 4523, 3, 0},
			{"e2c80d47-5ab9-4e13-9f76-1b3e8a4d2c06", 50.0, 1200, 0, 0},
			{"f5a62b93-8d14-4c0e-b357-9e1f4a6c7d28", 50.0, 340, 0, 0},
			{"07d4e1c8-2f96-4a5b-8e03-6b9c1d7f4a52", 42.3, 2100, 5, 0},
			{"18e5f2d9-3a07-4b6c-9f14-7c0d2e8a5b63", 12.0, 890, 8, 0},
			{"29f60e3a-4b18-4c7d-0a25-8d1e3f9b6c74", 0.0, 156, 12, 1},
		}
		for _, p := range demoPeers {
			db.Exec(`INSERT OR IGNORE INTO mesh_reputation (node_id, trust_score, total_requests, violations, blacklisted, last_activity) VALUES (?, ?, ?, ?, ?, datetime('now', '-' || ? || ' minutes'))`,
				p.nodeID, p.trust, p.requests, p.violations, p.blacklisted, rand.Intn(120))
		}
		fmt.Printf("  → Seeded %d demo peers into mesh_reputation.\n", len(demoPeers))
	}

	models := []string{"gpt-4o", "claude-3-5-sonnet-20240620", "deepseek-chat"}
	states := []string{"GREEN", "GREEN", "GREEN", "GREEN", "AMBER", "RED"}
	rand.Seed(time.Now().UnixNano())

	for {
		sessionID := fmt.Sprintf("demo-req-%d-%05d", time.Now().UnixNano(), rand.Intn(99999))

		var consumed float64
		var state string

		if rand.Float64() < 0.2 {
			consumed = 0.0
			state = "RED"
			fmt.Printf("[%s] WAF Block   -> %s\n", time.Now().Format("15:04:05"), sessionID)
		} else {
			consumed = 0.001 + rand.Float64()*0.049
			state = states[rand.Intn(len(states))]
			fmt.Printf("[%s] Normal Req  -> %s ($%.4f)\n", time.Now().Format("15:04:05"), sessionID, consumed)
		}

		allocated := 5.0 + rand.Float64()*45.0

		db.Exec(`INSERT OR IGNORE INTO tkngate_sessions (session_id, allocated_budget_usd, consumed_budget_usd, current_state) VALUES (?, ?, ?, ?)`,
			sessionID, allocated, consumed, state)

		if consumed > 0 {
			db.Exec(`INSERT INTO transactions (session_id, provider, model, input_tokens, output_tokens, estimated_cost_usd) VALUES (?, ?, ?, ?, ?, ?)`,
				sessionID, "openai", models[rand.Intn(len(models))], rand.Intn(490)+10, rand.Intn(490)+10, consumed)
		}

		time.Sleep(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
	}
}
