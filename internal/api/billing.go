package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"tkngate/internal/budget"
)

func handleExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keyHash := r.URL.Query().Get("key_hash")
	limitStr := r.URL.Query().Get("limit")
	limit := 1000 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	txs, err := budget.GlobalLedger.GetTransactions(keyHash, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to fetch transactions: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="billing_export.csv"`)

	writer := csv.NewWriter(w)
	// Write header
	writer.Write([]string{"ID", "SessionID", "VirtualKeyHash", "Provider", "Model", "InputTokens", "OutputTokens", "EstimatedCostUSD", "Timestamp"})

	for _, tx := range txs {
		writer.Write([]string{
			fmt.Sprintf("%d", tx.ID),
			tx.SessionID,
			tx.VirtualKeyHash,
			tx.Provider,
			tx.Model,
			fmt.Sprintf("%d", tx.InputTokens),
			fmt.Sprintf("%d", tx.OutputTokens),
			fmt.Sprintf("%.6f", tx.EstimatedCostUSD),
			tx.Timestamp.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func handleForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	days := 30 // lookback period for ML prediction
	dailySpend, err := budget.GlobalLedger.GetDailySpend(days)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to fetch daily spend: %v"}`, err), http.StatusInternalServerError)
		return
	}

	if len(dailySpend) < 2 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "insufficient_data",
			"message": "Need at least 2 days of data for linear regression forecasting.",
		})
		return
	}

	// Simple Linear Regression (Least Squares)
	// For simplicity, we calculate the average burn rate over the period
	var totalSpend float64
	for _, d := range dailySpend {
		totalSpend += d.SpendUSD
	}
	avgDailyBurn := totalSpend / float64(len(dailySpend))

	status, _ := budget.CheckBudget()
	remaining := status.RemainingUSD

	var daysUntilLimit int
	if avgDailyBurn > 0 {
		daysUntilLimit = int(remaining / avgDailyBurn)
	} else {
		daysUntilLimit = -1 // Will not hit limit
	}
	
	var predictedDate string
	if daysUntilLimit >= 0 {
		predictedDate = time.Now().AddDate(0, 0, daysUntilLimit).Format("2006-01-02")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "success",
		"analyzed_days":     len(dailySpend),
		"average_burn_rate": avgDailyBurn,
		"days_until_limit":  daysUntilLimit,
		"predicted_date":    predictedDate,
		"remaining_budget":  remaining,
	})
}
