package budget

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Ledger struct {
	db *sql.DB
	mu sync.RWMutex
}

var GlobalLedger *Ledger

func InitLedger() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(homeDir, ".tkngate")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(dir, "budget.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Create tables
	query := `
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		estimated_cost_usd REAL DEFAULT 0.0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		return err
	}

	GlobalLedger = &Ledger{db: db}
	return nil
}

func (l *Ledger) RecordTransaction(tx Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	query := `INSERT INTO transactions (provider, model, input_tokens, output_tokens, estimated_cost_usd, timestamp) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	_, err := l.db.Exec(query, tx.Provider, tx.Model, tx.InputTokens, tx.OutputTokens, tx.EstimatedCostUSD, time.Now())
	return err
}

func (l *Ledger) GetTotalSpend() (float64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var total float64
	err := l.db.QueryRow(`SELECT COALESCE(SUM(estimated_cost_usd), 0.0) FROM transactions`).Scan(&total)
	return total, err
}

func (l *Ledger) Reset() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.db.Exec(`DELETE FROM transactions`)
	return err
}
