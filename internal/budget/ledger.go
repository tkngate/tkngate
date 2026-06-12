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
		session_id TEXT DEFAULT '',
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		estimated_cost_usd REAL DEFAULT 0.0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS tkngate_sessions (
		session_id TEXT PRIMARY KEY,
		allocated_budget_usd REAL NOT NULL,
		consumed_budget_usd REAL DEFAULT 0.0,
		current_state TEXT DEFAULT 'GREEN',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS token_pool_nodes (
		node_id TEXT PRIMARY KEY,
		provider_type TEXT NOT NULL,
		blinded_key_hash TEXT NOT NULL,
		measured_tpm_limit INTEGER NOT NULL,
		remaining_tokens_quota INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(query); err != nil {
		return err
	}
	
	// Add session_id to existing table if migrating from v0.0.1
	db.Exec(`ALTER TABLE transactions ADD COLUMN session_id TEXT DEFAULT ''`)

	GlobalLedger = &Ledger{db: db}
	return nil
}

func (l *Ledger) RecordTransaction(tx Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	query := `INSERT INTO transactions (session_id, provider, model, input_tokens, output_tokens, estimated_cost_usd, timestamp) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := l.db.Exec(query, tx.SessionID, tx.Provider, tx.Model, tx.InputTokens, tx.OutputTokens, tx.EstimatedCostUSD, time.Now())
	if err != nil {
		return err
	}
	
	if tx.SessionID != "" {
		_, err = l.db.Exec(`UPDATE tkngate_sessions SET consumed_budget_usd = consumed_budget_usd + ? WHERE session_id = ?`, tx.EstimatedCostUSD, tx.SessionID)
	}
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
	if err == nil {
		l.db.Exec(`DELETE FROM tkngate_sessions`)
	}
	return err
}

func (l *Ledger) GetSessionSpend(sessionID string) (float64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var total float64
	err := l.db.QueryRow(`SELECT COALESCE(SUM(estimated_cost_usd), 0.0) FROM transactions WHERE session_id = ?`, sessionID).Scan(&total)
	return total, err
}

func (l *Ledger) EnsureSession(sessionID string, allocatedBudget float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	_, err := l.db.Exec(`INSERT OR IGNORE INTO tkngate_sessions (session_id, allocated_budget_usd) VALUES (?, ?)`, sessionID, allocatedBudget)
	return err
}

func (l *Ledger) AddPoolNode(node PoolNode) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	query := `INSERT INTO token_pool_nodes (node_id, provider_type, blinded_key_hash, measured_tpm_limit, remaining_tokens_quota) VALUES (?, ?, ?, ?, ?)`
	_, err := l.db.Exec(query, node.NodeID, node.ProviderType, node.BlindedKeyHash, node.MeasuredTpmLimit, node.RemainingTokensQuota)
	return err
}

func (l *Ledger) GetPoolNodes(provider string) ([]PoolNode, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	rows, err := l.db.Query(`SELECT node_id, provider_type, blinded_key_hash, measured_tpm_limit, remaining_tokens_quota FROM token_pool_nodes WHERE provider_type = ?`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []PoolNode
	for rows.Next() {
		var n PoolNode
		if err := rows.Scan(&n.NodeID, &n.ProviderType, &n.BlindedKeyHash, &n.MeasuredTpmLimit, &n.RemainingTokensQuota); err == nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}
