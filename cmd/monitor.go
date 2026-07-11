package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tkngate/internal/config"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Live Traffic & Telemetry Monitor (TUI)",
	Run: func(cmd *cobra.Command, args []string) {
		spinner, _ := pterm.DefaultSpinner.Start("Initializing TUI and connecting to daemon...")
		
		if err := config.LoadConfig(); err != nil {
			spinner.Fail("Failed to load config: ", err.Error())
			return
		}

		key := os.Getenv("TKNGATE_MASTER_KEY")
		if key == "" {
			spinner.Fail("TKNGATE_MASTER_KEY is missing from environment. Cannot authenticate to telemetry server.")
			return
		}

		spinner.Success("Connected to Telemetry API")
		time.Sleep(500 * time.Millisecond) // brief pause for effect

		area, _ := pterm.DefaultArea.Start()
		defer area.Stop()

		// Handle Ctrl+C to cleanly exit the area
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			area.Stop()
			fmt.Println("\nExited monitor mode.")
			os.Exit(0)
		}()

		client := &http.Client{Timeout: 2 * time.Second}

		// Validate that telemetry host is a safe localhost address (SSRF protection)
		allowedHosts := map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true, "": true}
		if !allowedHosts[config.Cfg.Telemetry.Host] {
			spinner.Fail("Telemetry host must be localhost/127.0.0.1 for security. Got: " + config.Cfg.Telemetry.Host)
			return
		}
		host := config.Cfg.Telemetry.Host
		if host == "" {
			host = "127.0.0.1"
		}
		if config.Cfg.Telemetry.Port < 1 || config.Cfg.Telemetry.Port > 65535 {
			spinner.Fail("Telemetry port must be between 1 and 65535.")
			return
		}
		endpoint := fmt.Sprintf("http://%s:%d/api/v1/overview", host, config.Cfg.Telemetry.Port)

		for {
			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				drawErrorBox(area, "Failed to create request: "+err.Error())
				time.Sleep(2 * time.Second)
				continue
			}
			req.Header.Set("Authorization", "Bearer "+key)

			resp, err := client.Do(req)
			if err != nil {
				drawErrorBox(area, "Connection Refused. Is the tkngate daemon running? ('tkngate serve')")
				time.Sleep(2 * time.Second)
				continue
			}

			if resp.StatusCode != 200 {
				drawErrorBox(area, fmt.Sprintf("API returned status: %d (Auth failure?)", resp.StatusCode))
				resp.Body.Close()
				time.Sleep(2 * time.Second)
				continue
			}

			var data struct {
				Status        string  `json:"status"`
				TotalSpendUSD float64 `json:"total_spend_usd"`
				GlobalLimit   float64 `json:"global_limit"`
				TotalRequests int     `json:"total_requests"`
				CacheStats    struct {
					Hits    int     `json:"hits"`
					Misses  int     `json:"misses"`
					Entries int     `json:"entries"`
					Savings float64 `json:"savings"`
				} `json:"cache_stats"`
				Timestamp time.Time `json:"timestamp"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				drawErrorBox(area, "Failed to decode response: "+err.Error())
				resp.Body.Close()
				time.Sleep(2 * time.Second)
				continue
			}
			resp.Body.Close()

			// Calculate hit rate
			totalCacheReqs := data.CacheStats.Hits + data.CacheStats.Misses
			hitRate := 0.0
			if totalCacheReqs > 0 {
				hitRate = float64(data.CacheStats.Hits) / float64(totalCacheReqs) * 100
			}

			// Render the TUI
			var content string
			content += Gold(fmt.Sprintf("■ TKNGATE ENTERPRISE LIVE MONITOR    [ %s ]\n", data.Timestamp.Format("15:04:05")))
			content += Forest("--------------------------------------------------------------------------------\n")
			content += "\n"
			
			content += fmt.Sprintf("  %s %s\n", Gold("Status:         "), Forest("● ONLINE"))
			content += fmt.Sprintf("  %s %s\n", Gold("Total Traffic:  "), fmt.Sprintf("%d requests processed", data.TotalRequests))
			
			budgetStr := fmt.Sprintf("$%.5f / $%.2f", data.TotalSpendUSD, data.GlobalLimit)
			if data.TotalSpendUSD >= data.GlobalLimit*0.9 {
				budgetStr = pterm.LightRed(budgetStr)
			} else {
				budgetStr = Forest(budgetStr)
			}
			content += fmt.Sprintf("  %s %s\n", Gold("Active Spend:   "), budgetStr)
			content += "\n"

			content += Gold("■ SEMANTIC CACHE ENGINE\n")
			content += fmt.Sprintf("  %s %s\n", Gold("Cached Entries: "), fmt.Sprintf("%d items", data.CacheStats.Entries))
			content += fmt.Sprintf("  %s %s\n", Gold("Cache Hits:     "), fmt.Sprintf("%d hits", data.CacheStats.Hits))
			content += fmt.Sprintf("  %s %s\n", Gold("Hit Rate:       "), fmt.Sprintf("%.1f%%", hitRate))
			content += fmt.Sprintf("  %s %s\n", Gold("Saved Budget:   "), Forest(fmt.Sprintf("$%.5f", data.CacheStats.Savings)))
			content += "\n"
			
			content += Parch("Press Ctrl+C to exit monitor mode.")

			box := pterm.DefaultBox.WithTitle("LIVE TUI").WithRightPadding(5).WithLeftPadding(5).Sprint(content)
			area.Update(box)

			// Poll every 1 second
			time.Sleep(1 * time.Second)
		}
	},
}

func drawErrorBox(area *pterm.AreaPrinter, msg string) {
	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		pterm.LightRed("■ TKNGATE MONITOR ERROR"),
		pterm.LightRed(msg),
		Parch("Retrying in 2 seconds... (Press Ctrl+C to exit)"))
	box := pterm.DefaultBox.WithTitle("ERROR").WithRightPadding(5).WithLeftPadding(5).Sprint(content)
	area.Update(box)
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}
