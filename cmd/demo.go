package cmd

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run the interactive traffic generator demo",
	Run: func(cmd *cobra.Command, args []string) {
		// 'demo' is now simply an alias for 'serve --demo' 
		// to guarantee the dashboard always spins up for the user.
		demoMode = true
		if serveCmd.Run != nil {
			serveCmd.Run(cmd, args)
		}
	},
}

// GenerateDemoTraffic is shared by both `tkngate demo` and `tkngate serve --demo`
// to ensure the traffic generation logic lives in exactly one place.
func GenerateDemoTraffic(db *sql.DB, isStandalone bool) {
	// Seed mesh_reputation demo peers if table is empty
	var repCount int
	db.QueryRow("SELECT COUNT(*) FROM mesh_reputation").Scan(&repCount)
	if repCount == 0 {
		if isStandalone {
			fmt.Println("Seeding demo reputation data for Peer Leaderboard...")
		}
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
		if isStandalone {
			fmt.Printf("  → Seeded %d demo peers into mesh_reputation.\n", len(demoPeers))
		}
	}

	models := []string{"gpt-4o", "claude-3-5-sonnet-20240620", "deepseek-chat"}
	states := []string{"GREEN", "GREEN", "GREEN", "GREEN", "AMBER", "RED"}

	if isStandalone {
		fmt.Println("Simulating background traffic to budget_demo.db...")
		fmt.Println("Note: Dashboard is OFFLINE in standalone mode. Run './tkngate serve --demo' if you want to watch this live.")
	} else {
		fmt.Println()
		fmt.Println("■ DEMO MODE: Generating simulated traffic...")
		fmt.Println("Simulating live traffic. You can watch this live on the dashboard (http://localhost:7478)!")
	}

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

		_, err := db.Exec(`
			INSERT OR IGNORE INTO tkngate_sessions (session_id, allocated_budget_usd, consumed_budget_usd, current_state)
			VALUES (?, ?, ?, ?)
		`, sessionID, allocated, consumed, state)
		if err != nil && isStandalone {
			fmt.Println("Error inserting session:", err)
		}

		if consumed > 0 {
			_, err = db.Exec(`
				INSERT INTO transactions (session_id, provider, model, input_tokens, output_tokens, estimated_cost_usd)
				VALUES (?, ?, ?, ?, ?, ?)
			`, sessionID, "openai", models[rand.Intn(len(models))], rand.Intn(490)+10, rand.Intn(490)+10, consumed)
			if err != nil && isStandalone {
				fmt.Println("Error inserting tx:", err)
			}
		}

		time.Sleep(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
	}
}

func init() {
	rootCmd.AddCommand(demoCmd)
}
