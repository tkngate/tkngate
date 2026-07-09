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

// DB exposes the underlying database handle for shared use (e.g., mesh reputation tables).
func (l *Ledger) DB() *sql.DB {
	return l.db
}

func InitLedger() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(homeDir, ".tkngate")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	dbPath := filepath.Join(dir, "budget.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	
	// Enable Write-Ahead Logging (WAL) for high concurrency
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`); err != nil {
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

	CREATE TABLE IF NOT EXISTS tkngate_virtual_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_hash TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		allocated_budget_usd REAL NOT NULL,
		consumed_budget_usd REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

	if tx.VirtualKeyHash != "" {
		_, err = l.db.Exec(`UPDATE tkngate_virtual_keys SET consumed_budget_usd = consumed_budget_usd + ? WHERE key_hash = ?`, tx.EstimatedCostUSD, tx.VirtualKeyHash)
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

func (l *Ledger) GetSessions() ([]SessionState, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	rows, err := l.db.Query(`SELECT session_id, allocated_budget_usd, consumed_budget_usd, current_state, created_at FROM tkngate_sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionState
	for rows.Next() {
		var s SessionState
		if err := rows.Scan(&s.SessionID, &s.AllocatedBudget, &s.ConsumedBudget, &s.CurrentState, &s.CreatedAt); err == nil {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (l *Ledger) GetTransactionCount() (int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var count int
	err := l.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&count)
	return count, err
}

func (l *Ledger) DecrementPoolQuota(nodeID string, tokens int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.db.Exec(`UPDATE token_pool_nodes SET remaining_tokens_quota = remaining_tokens_quota - ? WHERE node_id = ? AND remaining_tokens_quota > 0`, tokens, nodeID)
}

// v1.3.0: Virtual Key Management

type VirtualKeyRecord struct {
	ID              int
	KeyHash         string
	Name            string
	AllocatedBudget float64
	ConsumedBudget  float64
	CreatedAt       string
}

func (l *Ledger) RegisterVirtualKey(keyHash, name string, allocatedBudget float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.db.Exec(`INSERT INTO tkngate_virtual_keys (key_hash, name, allocated_budget_usd) VALUES (?, ?, ?)`, keyHash, name, allocatedBudget)
	return err
}

func (l *Ledger) GetVirtualKeys() ([]VirtualKeyRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	rows, err := l.db.Query(`SELECT id, key_hash, name, allocated_budget_usd, consumed_budget_usd, created_at FROM tkngate_virtual_keys`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []VirtualKeyRecord
	for rows.Next() {
		var k VirtualKeyRecord
		if err := rows.Scan(&k.ID, &k.KeyHash, &k.Name, &k.AllocatedBudget, &k.ConsumedBudget, &k.CreatedAt); err == nil {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (l *Ledger) ChargeVirtualKey(keyHash string, amount float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.db.Exec(`UPDATE tkngate_virtual_keys SET consumed_budget_usd = consumed_budget_usd + ? WHERE key_hash = ?`, amount, keyHash)
	return err
}

func (l *Ledger) RevokeVirtualKey(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	res, err := l.db.Exec(`DELETE FROM tkngate_virtual_keys WHERE name = ?`, name)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// InitMemoryLedger creates an in-memory ledger strictly for unit testing.
func InitMemoryLedger() error {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}

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

	CREATE TABLE IF NOT EXISTS tkngate_virtual_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_hash TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		allocated_budget_usd REAL NOT NULL,
		consumed_budget_usd REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		return err
	}

	GlobalLedger = &Ledger{db: db}
	return nil
}
